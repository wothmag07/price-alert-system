# Price Ingestion Service - How It Works Internally

## What Does This Service Do?

This service is the **data entry point** of the entire price alert system. It:

1. Connects to Binance (a crypto exchange) via WebSocket
2. Receives real-time price updates for coins like BTC, ETH, SOL
3. Publishes those prices to three places: Kafka, Redis, and PostgreSQL

Think of it as a **pipe** that takes live price data from the outside world and feeds it into your system.

---

## The Big Picture

```
                                    +------------------+
                                    |      Kafka       |
                                +-->| topic:           |---> Alert Engine (downstream)
                                |   | "price-updates"  |
  +---------+    +----------+   |   +------------------+
  | Binance |--->| This     |---+
  | WebSocket|   | Service  |---+   +------------------+
  +---------+    +----------+   |   |      Redis       |
                                +-->| key:             |---> API Server (reads latest)
                                |   | price:latest:BTC |
                                |   +------------------+
                                |
                                |   +------------------+
                                +-->|   PostgreSQL      |
                                    | table:            |---> Web UI (price history)
                                    | price_history     |
                                    +------------------+
```

---

## File-by-File Walkthrough

### 1. `main.go` - The Orchestrator

This is where everything starts. Think of it as the **manager** that hires workers and tells them what to do.

```
main() starts
  |
  |-- loadConfig()              --> Read env vars (what symbols to track, where is Kafka, etc.)
  |
  |-- signal.NotifyContext()    --> Set up a "kill switch" that activates on Ctrl+C or SIGTERM
  |
  |-- NewPublisher()            --> Connect to Kafka, Redis, PostgreSQL (if any fail, crash immediately)
  |
  |-- make(chan []byte, 256)    --> Create a "mailbox" (channel) with room for 256 messages
  |
  |-- go connectBinance()       --> Start the WebSocket reader in a separate goroutine (worker thread)
  |
  |-- for { select { ... } }   --> Main loop: wait for messages or shutdown signal
        |
        |-- case raw := <-messages:    --> Got a price message!
        |     |-- parseMiniTicker()    --> Parse it into a clean struct
        |     |-- go pub.Publish()     --> Fire-and-forget: publish to Kafka/Redis/PG
        |
        |-- case <-ctx.Done():         --> Got shutdown signal!
              |-- return               --> Exit (defer pub.Close() cleans up connections)
```

#### Key Concept: `signal.NotifyContext`

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
```

This creates a `context` that automatically cancels when the process receives SIGTERM (docker stop) or SIGINT (Ctrl+C). Every function in this service receives this `ctx` — when it cancels, **everyone stops**.

```
You press Ctrl+C
      |
      v
ctx gets cancelled
      |
      +---> main loop's `case <-ctx.Done()` fires --> exits
      +---> connectBinance's `case <-ctx.Done()` fires --> stops reconnecting
      +---> readLoop's goroutine closes the WebSocket connection
      +---> `defer pub.Close()` runs --> disconnects Kafka, Redis, PostgreSQL
```

It's like pulling one plug and the whole system shuts down gracefully.

#### Key Concept: The Channel (`messages`)

```go
messages := make(chan []byte, 256)
```

A channel is a **thread-safe queue** that goroutines use to communicate.

```
  connectBinance goroutine                 main goroutine
  ========================                 ===============
  reads from WebSocket                     waits on channel
        |                                       |
        |--- messages <- rawBytes ------>  case raw := <-messages
        |                                       |
  (keeps reading)                         (processes + publishes)
```

The `256` buffer means: "hold up to 256 messages if the main loop is busy." Without the buffer, the WebSocket reader would block every time it sends a message until the main loop reads it.

#### Key Concept: Fire-and-Forget with `go`

```go
go pub.Publish(ctx, event)
```

This starts `Publish` in a **new goroutine** and immediately returns. The main loop doesn't wait for Kafka/Redis/PG to finish writing — it goes right back to reading the next message.

Why? Binance sends data fast. If we waited for each publish to complete, messages would pile up and we'd fall behind on live prices.

---

### 2. `config.go` - Environment Variables

This is the simplest file. It reads settings from environment variables with sensible defaults.

```go
type Config struct {
    BinanceWsURL   string     // Where to connect for price data
    TrackedSymbols []string   // Which coins to track: ["btcusdt", "ethusdt", "solusdt"]
    KafkaBrokers   []string   // Kafka server addresses
    RedisAddr      string     // Redis address (host:port)
    PostgresHost   string     // PostgreSQL connection details
    PostgresPort   int
    PostgresDB     string
    PostgresUser   string
    PostgresPassword string
}
```

#### How `envOrDefault` works:

```go
func envOrDefault(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

```
os.Getenv("KAFKA_BROKERS")
  |
  |--> "kafka:9092"  (set in docker-compose) --> use this
  |--> ""            (not set, running locally) --> use "localhost:9092"
```

#### How `splitComma` works:

```
Input:  "btcusdt, ethusdt, solusdt"
                    |
    strings.Split(",") --> ["btcusdt", " ethusdt", " solusdt"]
                    |
    TrimSpace + ToLower --> ["btcusdt", "ethusdt", "solusdt"]
```

---

### 3. `binance.go` - WebSocket Client

This file handles the live connection to Binance. It has two functions:

#### `connectBinance` - The Reconnection Loop

This function **never returns** (until ctx is cancelled). It runs in a loop:

```
connect --> read messages --> connection drops --> wait (backoff) --> connect again
```

**URL Construction:**

```
Base:    wss://stream.binance.com:9443/ws
Symbols: [btcusdt, ethusdt, solusdt]

Result:  wss://stream.binance.com:9443/stream?streams=btcusdt@miniTicker/ethusdt@miniTicker/solusdt@miniTicker
```

**Exponential Backoff:**

If the connection drops, don't immediately reconnect — wait longer each time:

```
1st disconnect:  wait 1s   then reconnect
2nd disconnect:  wait 2s   then reconnect
3rd disconnect:  wait 4s   then reconnect
4th disconnect:  wait 8s   then reconnect
5th disconnect:  wait 16s  then reconnect
6th disconnect:  wait 30s  then reconnect (capped at 30s)
7th disconnect:  wait 30s  then reconnect
...
Successful connection: reset backoff to 0
```

Why? If Binance is down, hammering it with reconnection attempts is wasteful and might get you rate-limited.

**The backoff wait is interruptible:**

```go
select {
case <-time.After(backoff):    // backoff timer expired, reconnect
case <-ctx.Done():             // shutdown signal, stop entirely
    return
}
```

If you press Ctrl+C while waiting for backoff, it exits immediately instead of waiting the full 30 seconds.

#### `readLoop` - One Connection Lifetime

This function handles a single WebSocket connection:

```
Dial the WebSocket
  |
  +--> Start a "watchdog" goroutine:
  |      go func() {
  |          <-ctx.Done()      // wait for shutdown
  |          conn.Close()      // force-close the connection
  |      }()
  |
  +--> Loop: read messages forever
         |
         |-- conn.ReadMessage()
         |     |
         |     |--> Success: parse and send to channel
         |     |--> Error (connection dropped): return error
         |
         |-- Parse combined stream format:
               Binance sends: {"stream":"btcusdt@miniTicker","data":{...actual data...}}
               We extract:    {... actual data ...}
               And send it to the messages channel
```

**Why `json.RawMessage`?**

```go
type combinedStream struct {
    Stream string          `json:"stream"`
    Data   json.RawMessage `json:"data"`    // <-- this is key
}
```

`json.RawMessage` means "don't parse this field, keep it as raw bytes." We don't need to understand the inner data here — we just need to extract it and pass it along. The `processor.go` will parse it later. This avoids parsing the same JSON twice.

---

### 4. `processor.go` - Data Transformer

This file converts raw Binance JSON into our clean `PriceUpdateEvent` struct.

#### What Binance Sends:

```json
{
  "e": "24hrMiniTicker",
  "s": "BTCUSDT",
  "c": "67543.21000000",
  "o": "66890.00000000",
  "v": "12345.67800000",
  "E": 1707667200000
}
```

Single-letter field names! Binance is optimized for bandwidth, not readability.

#### What We Convert It To:

```json
{
  "symbol": "BTCUSDT",
  "price": 67543.21,
  "volume": 12345.678,
  "change24h": 0.9768,
  "timestamp": 1707667200000
}
```

Clean, readable, and consistent across our entire system.

#### The Parsing Pipeline:

```
Raw JSON bytes
      |
      v
json.Unmarshal into binanceMiniTicker struct
      |
      |--> Is EventType == "24hrMiniTicker"?  No --> return nil (ignore)
      |--> Are all required fields present?    No --> return nil (ignore)
      |
      v
strconv.ParseFloat for close, open, volume
      |
      |--> Any parse errors?                  Yes --> return nil (ignore)
      |--> Any NaN values?                    Yes --> return nil (ignore)
      |
      v
Calculate change24h = ((close - open) / open) * 100
      |
      v
Return &PriceUpdateEvent{...}
```

#### Key Concept: Returning `*PriceUpdateEvent` (pointer)

```go
func parseMiniTicker(raw []byte) *PriceUpdateEvent {
```

Returning a pointer (`*`) allows us to return `nil` for invalid messages. This is Go's way of saying "no result" — similar to returning `null` in TypeScript.

The caller checks:

```go
event := parseMiniTicker(raw)
if event != nil {        // only publish valid events
    go pub.Publish(ctx, event)
}
```

---

### 5. `publisher.go` - The Fan-Out Engine

This is the most complex file. It sends each price event to three systems simultaneously.

#### The Publisher Struct:

```go
type Publisher struct {
    kafkaWriter *kafka.Writer       // sends to Kafka
    redisClient *redis.Client       // sends to Redis
    pgPool      *pgxpool.Pool       // sends to PostgreSQL

    mu          sync.Mutex          // lock for lastDBWrite map
    lastDBWrite map[string]int64    // tracks when we last wrote to DB per symbol
}
```

#### `NewPublisher` - Connecting Everything:

```
NewPublisher(ctx, cfg)
      |
      |-- Create kafka.Writer (lazy connect, no ping needed)
      |
      |-- Create redis.Client --> Ping! --> fail? crash the service
      |
      |-- Create pgxpool.Pool --> Ping! --> fail? crash the service
      |
      |-- Return &Publisher{...}
```

The pings are a "fail-fast" pattern: if infrastructure is down, crash immediately instead of accepting messages and silently failing later.

#### `Publish` - The Parallel Fan-Out:

```go
func (p *Publisher) Publish(ctx context.Context, event *PriceUpdateEvent) {
    var wg sync.WaitGroup
    wg.Add(3)

    go func() { defer wg.Done(); /* write to Kafka */ }()
    go func() { defer wg.Done(); /* write to Redis */ }()
    go func() { defer wg.Done(); /* write to PostgreSQL */ }()

    wg.Wait()  // wait for all 3 to finish
}
```

Visualized:

```
Publish() called with BTCUSDT price
      |
      +-----> goroutine 1: Kafka
      |       kafkaWriter.WriteMessages(topic:"price-updates", key:"BTCUSDT", value:JSON)
      |
      +-----> goroutine 2: Redis
      |       SET price:latest:BTCUSDT <json> EX 60
      |       (expires in 60 seconds — always fresh)
      |
      +-----> goroutine 3: PostgreSQL
      |       INSERT INTO price_history (if 10s since last write)
      |
      +-----> wg.Wait() -- all 3 done, Publish() returns
```

All three happen **at the same time** (parallel), not one after another. This is faster because we don't wait for Kafka to finish before starting Redis.

#### Key Concept: `sync.WaitGroup`

```
wg.Add(3)      --> "I'm expecting 3 tasks to complete"

go func() {
    defer wg.Done()    --> "Task 1 done" (runs even if the goroutine panics)
    ...
}()

wg.Wait()      --> "Block here until all 3 call Done()"
```

It's like telling 3 friends to go buy different groceries and waiting at the car until all 3 come back.

#### Key Concept: `sync.Mutex` (Database Throttling)

Binance sends price updates **every second**. Writing every single one to PostgreSQL would be wasteful. So we throttle: **max 1 write per 10 seconds per symbol**.

```go
func (p *Publisher) maybePersist(ctx context.Context, event *PriceUpdateEvent) {
    p.mu.Lock()                                           // lock the map
    lastWrite := p.lastDBWrite[event.Symbol]
    if event.Timestamp - lastWrite < 10000 {              // less than 10s ago?
        p.mu.Unlock()                                     // unlock and skip
        return
    }
    p.lastDBWrite[event.Symbol] = event.Timestamp         // update timestamp
    p.mu.Unlock()                                         // unlock before DB call

    // INSERT INTO price_history ...
}
```

**Why the Mutex?**

Multiple goroutines call `maybePersist` at the same time (one for each incoming price). Go maps are **NOT safe for concurrent access**. If two goroutines read/write the map at the same time, the program crashes.

```
Without Mutex:
  goroutine A reads lastDBWrite["BTCUSDT"]     <-- reading
  goroutine B writes lastDBWrite["BTCUSDT"]    <-- writing at the same time!
  CRASH: concurrent map read and write

With Mutex:
  goroutine A: Lock() --> read --> write --> Unlock()
  goroutine B: Lock() --> (waits...) --> A unlocks --> read --> write --> Unlock()
  SAFE: only one goroutine accesses the map at a time
```

Notice we `Unlock()` **before** the database INSERT. This is important — the DB call is slow (network I/O), and we don't want to hold the lock during it. Other goroutines can check the map while one is doing the INSERT.

#### Where Each Piece of Data Goes:

| Destination | What's Stored | Why | Who Reads It |
|------------|---------------|-----|-------------|
| **Kafka** | Every price event as JSON | Stream processing pipeline | Alert Engine consumes it |
| **Redis** | Latest price per symbol, expires in 60s | Fast reads for "current price" | API Server reads it |
| **PostgreSQL** | Price every 10s per symbol | Historical data for charts | Web UI queries it |

---

## The Complete Data Flow (Step by Step)

```
1. Binance sends WebSocket message:
   {"stream":"btcusdt@miniTicker","data":{"e":"24hrMiniTicker","s":"BTCUSDT","c":"67543.21",...}}

2. readLoop (binance.go) receives it:
   - Parses combined stream format
   - Extracts inner "data" as raw bytes
   - Sends bytes into the `messages` channel

3. Main loop (main.go) receives from channel:
   - Calls parseMiniTicker(raw)
   - Gets back &PriceUpdateEvent{Symbol:"BTCUSDT", Price:67543.21, ...}
   - Launches `go pub.Publish(ctx, event)`

4. Publish (publisher.go) fans out in parallel:
   - Goroutine 1: Kafka --> topic "price-updates", key "BTCUSDT"
   - Goroutine 2: Redis --> SET price:latest:BTCUSDT
   - Goroutine 3: PostgreSQL --> INSERT INTO price_history (if 10s passed)

5. All 3 goroutines finish, Publish() returns, goroutine dies.

6. Meanwhile, main loop already picked up the next message (step 3 again).
```

---

## Goroutine Map (What's Running Concurrently)

At any given moment while the service is running:

```
Goroutine 1 (main):         for { select { ... } }  -- reading from channel
Goroutine 2 (binance):      connectBinance()         -- reading from WebSocket
Goroutine 3 (watchdog):     <-ctx.Done(); conn.Close() -- waiting for shutdown
Goroutine 4 (publish):      pub.Publish() for BTCUSDT  -- writing to Kafka/Redis/PG
Goroutine 5 (publish):      pub.Publish() for ETHUSDT  -- writing to Kafka/Redis/PG
Goroutine 6 (kafka):        kafkaWriter.WriteMessages   -- network I/O to Kafka
Goroutine 7 (redis):        redisClient.Set             -- network I/O to Redis
...
```

Goroutines 4-7+ are **short-lived** — they spawn, do their work, and die. Goroutines 1-3 live for the entire lifetime of the service.

---

## Shutdown Sequence

```
Ctrl+C pressed (SIGINT)
      |
      v
ctx gets cancelled (signal.NotifyContext)
      |
      +---> main loop: case <-ctx.Done() fires
      |       |
      |       v
      |     return from main()
      |       |
      |       v
      |     defer pub.Close() runs:
      |       - kafkaWriter.Close()   --> flushes pending messages
      |       - redisClient.Close()   --> closes Redis connection
      |       - pgPool.Close()        --> closes all PG connections
      |
      +---> connectBinance: ctx.Done() fires
      |       - stops reconnection loop
      |
      +---> readLoop watchdog: ctx.Done() fires
              - conn.Close() --> WebSocket connection drops
              - readLoop returns error
              - connectBinance sees ctx is done, returns
```

Everything shuts down cleanly. No leaked connections, no orphaned goroutines.
