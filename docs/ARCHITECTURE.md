# Price Alert System — Architecture

## High-Level System Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           EXTERNAL                                       │
│   ┌─────────────┐                          ┌──────────┐                  │
│   │ Binance WS  │                          │  Resend  │                  │
│   │   (Prices)  │                          │  (Email) │                  │
│   └──────┬──────┘                          └────▲─────┘                  │
└──────────┼──────────────────────────────────────┼────────────────────────┘
           │                                      │
┌──────────▼──────────────────────────────────────┼────────────────────────┐
│                        BACKEND SERVICES                                  │
│                                                 │                        │
│  ┌──────────────────┐    ┌──────────────────────┴───────────────────┐    │
│  │ Price Ingestion  │    │         Notification Service             │    │
│  │   Service        │    │  ┌─────────────┐  ┌──────────────────┐  │    │
│  │                  │    │  │ Email Sender │  │ WebSocket Pusher │  │    │
│  └────────┬─────────┘    │  └─────────────┘  └──────────────────┘  │    │
│           │              │  ┌──────────────────────────────────┐   │    │
│           │              │  │ Rate Limiter (per-user throttle) │   │    │
│           │              └──┴──────────────────▲───────────────┘    │    │
│           │                                    │                        │
│     ┌─────▼──────────────────────────┐   ┌─────┴──────────────────┐    │
│     │        Apache Kafka            │   │     Apache Kafka       │    │
│     │   topic: price-updates         │   │  topic: alert-triggers │    │
│     └─────┬──────────┬──────────┬────┘   └─────▲──────────────────┘    │
│           │          │          │               │                        │
│     ┌─────▼────┐ ┌───▼────┐ ┌──▼───────┐ ┌────┴─────────┐             │
│     │  Alert   │ │Analytics│ │   API    │ │    Alert     │             │
│     │  Engine  │ │ Service │ │  Server  │ │    Engine    │             │
│     │          │─┘         │ │          │ │  (produces)  │             │
│     │(consumes)│  (Top-K)  │ │(WebSocket│ └──────────────┘             │
│     └────┬─────┘ └────┬────┘ │ + REST)  │                              │
│          │            │      └──┬───────┘                              │
│          │            │         │                                        │
│     ┌────▼────────────▼─────────▼──────────────────────┐               │
│     │              DATA LAYER                           │               │
│     │  ┌────────────┐  ┌───────┐  ┌────────────────┐  │               │
│     │  │ PostgreSQL │  │ Redis │  │ Redis (Top-K)  │  │               │
│     │  │            │  │       │  │                 │  │               │
│     │  │ - users    │  │-cache │  │ - sorted sets   │  │               │
│     │  │ - alerts   │  │-rates │  │ - trending data │  │               │
│     │  │ - history  │  │-pub/  │  │                 │  │               │
│     │  │ - prices   │  │ sub   │  │                 │  │               │
│     │  └────────────┘  └───────┘  └────────────────┘  │               │
│     └──────────────────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────────────────────┘
           ▲
           │ WebSocket + REST
           │
┌──────────┴──────────────────────────────────────────────────────────────┐
│                          FRONTEND                                        │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  React + Vite + Tailwind                                        │    │
│  │  ┌──────────┐  ┌──────────────┐  ┌──────────┐  ┌────────────┐ │    │
│  │  │  Login/  │  │ Live Price   │  │  Alert   │  │  Trending  │ │    │
│  │  │ Register │  │  Dashboard   │  │  Manager │  │   Top-K    │ │    │
│  │  └──────────┘  └──────────────┘  └──────────┘  └────────────┘ │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

## Data Flow

### 1. Price Ingestion Flow
```
Binance WebSocket API
  → Price Ingestion Service (parse, normalize)
    → Kafka topic: price-updates (partitioned by asset symbol)
    → Redis SET price:latest:{symbol} (cache latest price)
```

### 2. Alert Matching Flow
```
Kafka topic: price-updates
  → Alert Engine (consume price events)
    → Redis GET active-alerts:{symbol} (cached alert rules)
    → Match price against each rule (>, <, % change)
    → If triggered:
      → PostgreSQL: update alert status, insert alert_history
      → Kafka topic: alert-triggers (publish trigger event)
      → Redis DEL active-alerts:{symbol} cache entry (invalidate)
```

### 3. Notification Flow
```
Kafka topic: alert-triggers
  → Notification Service (consume trigger events)
    → Redis: check rate limit for user (sliding window)
    → If within limit:
      → Email: send via Resend API
      → WebSocket: push to connected client via API Server
      → PostgreSQL: update alert_history with delivery status
    → If rate limited:
      → Queue for later / skip with log
```

### 4. Real-Time Client Flow
```
Frontend (React)
  → WebSocket connection to API Server
    → API Server subscribes to Kafka price-updates
    → Streams price ticks to connected clients
    → Pushes alert trigger notifications
```

### 5. Analytics Flow
```
Kafka topic: price-updates
  → Analytics Service (consume price events)
    → Calculate % change over rolling windows (1m, 5m, 1h, 24h)
    → Redis ZADD top-drops:{window} (sorted set, score = % drop)
    → Redis ZREVRANGE top-drops:{window} 0 9 (query Top-10)
```

---

## Service Details

### 1. Price Ingestion Service
**Responsibility:** Connect to Binance WebSocket, normalize price data, publish to Kafka.

- Subscribes to Binance combined stream for multiple trading pairs
- Normalizes ticker data into a `PriceUpdate` event
- Publishes to Kafka `price-updates` topic (key = symbol for partition affinity)
- Caches latest price in Redis
- Handles disconnection/reconnection with exponential backoff

**Key config:** List of tracked symbols (BTC, ETH, SOL, DOGE, etc.)

### 2. Alert Engine
**Responsibility:** Match incoming prices against user-defined alert rules.

- Consumes from Kafka `price-updates` (consumer group: `alert-engine`)
- On each price event, loads active alerts for that symbol from Redis cache
- Cache miss → query PostgreSQL, populate cache
- Supports conditions: `PRICE_ABOVE`, `PRICE_BELOW`, `PCT_CHANGE_ABOVE`, `PCT_CHANGE_BELOW`
- Triggered alerts: update DB status, publish to Kafka `alert-triggers`
- Alert states: `ACTIVE` → `TRIGGERED` → `NOTIFIED`

### 3. Notification Service
**Responsibility:** Deliver notifications to users when alerts trigger.

- Consumes from Kafka `alert-triggers` (consumer group: `notification-service`)
- Per-user rate limiting via Redis sliding window (default: 10 notifications/hour)
- Email delivery via Resend API (with retry on failure)
- WebSocket push via internal event to API Server (Redis pub/sub channel)
- Tracks delivery status in `alert_history` table

### 4. API Server
**Responsibility:** REST API + WebSocket server for frontend clients.

- **Auth:** JWT-based. `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`
- **Alerts CRUD:** `GET/POST/PUT/DELETE /alerts`
- **Prices:** `GET /prices/latest`, `GET /prices/history/:symbol`
- **Analytics:** `GET /analytics/top-drops?window=1h`
- **WebSocket:** `/ws` endpoint — streams live prices + alert notifications
- **Middleware:** JWT validation, rate limiting (Redis), request logging

### 5. Analytics Service
**Responsibility:** Compute trending price movements using Top-K pattern.

- Consumes from Kafka `price-updates` (consumer group: `analytics-service`)
- Maintains rolling price windows in Redis (sorted sets with TTL)
- Calculates percentage change over configurable windows
- Updates Redis sorted sets for Top-K queries
- API Server queries Redis directly for Top-K results

### 6. Web Frontend
**Responsibility:** User interface for the system.

- **Pages:** Login, Register, Dashboard, Alert Management
- **Live Prices:** WebSocket connection shows real-time price ticks
- **Alerts:** Create/edit/delete alert rules, see trigger history
- **Trending:** Top-K biggest drops display
- **Notifications:** Toast/banner when an alert triggers in real-time

---

## Database Schema

### PostgreSQL

```sql
-- Users table
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       VARCHAR(255) UNIQUE NOT NULL,
    password    VARCHAR(255) NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

-- Alert rules
CREATE TABLE alerts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol      VARCHAR(20) NOT NULL,           -- e.g., 'BTCUSDT'
    condition   VARCHAR(30) NOT NULL,           -- PRICE_ABOVE, PRICE_BELOW, PCT_CHANGE_ABOVE, PCT_CHANGE_BELOW
    threshold   DECIMAL(20, 8) NOT NULL,        -- target price or percentage
    status      VARCHAR(20) DEFAULT 'ACTIVE',   -- ACTIVE, TRIGGERED, CANCELLED
    created_at  TIMESTAMP DEFAULT NOW(),
    triggered_at TIMESTAMP,
    INDEX idx_alerts_symbol_status (symbol, status),
    INDEX idx_alerts_user_id (user_id)
);

-- Alert trigger history
CREATE TABLE alert_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id        UUID NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    triggered_price DECIMAL(20, 8) NOT NULL,
    notification    JSONB,                       -- { email: "sent", websocket: "delivered" }
    created_at      TIMESTAMP DEFAULT NOW(),
    INDEX idx_alert_history_alert_id (alert_id)
);

-- Price history (time-series)
CREATE TABLE price_history (
    id          BIGSERIAL PRIMARY KEY,
    symbol      VARCHAR(20) NOT NULL,
    price       DECIMAL(20, 8) NOT NULL,
    volume      DECIMAL(20, 8),
    timestamp   TIMESTAMP NOT NULL,
    INDEX idx_price_history_symbol_ts (symbol, timestamp DESC)
);
```

### Redis Keys

| Key Pattern | Type | Purpose | TTL |
|-------------|------|---------|-----|
| `price:latest:{symbol}` | String (JSON) | Latest price for quick reads | 60s |
| `alerts:active:{symbol}` | Set (JSON) | Cached active alerts per symbol | 5m |
| `rate-limit:{userId}:{window}` | Sorted Set | Sliding window rate limiter | auto |
| `top-drops:{window}` | Sorted Set | Top-K price drops (score = % drop) | matches window |
| `ws:notify:{userId}` | Pub/Sub channel | Push notifications to API Server | - |

---

## Kafka Topics

### `price-updates`
- **Partitions:** 6 (one per major symbol, or hash-based)
- **Key:** symbol (e.g., `BTCUSDT`) — ensures ordering per asset
- **Retention:** 24 hours
- **Consumers:** Alert Engine, Analytics Service, API Server

```typescript
interface PriceUpdateEvent {
  symbol: string;        // "BTCUSDT"
  price: number;         // 67432.50
  volume: number;        // 24h volume
  change24h: number;     // percentage
  timestamp: number;     // unix ms
}
```

### `alert-triggers`
- **Partitions:** 3
- **Key:** userId
- **Retention:** 7 days
- **Consumers:** Notification Service

```typescript
interface AlertTriggerEvent {
  alertId: string;
  userId: string;
  symbol: string;
  condition: string;
  threshold: number;
  triggeredPrice: number;
  timestamp: number;
}
```

---

## API Endpoints

### Auth
| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Create new user |
| POST | `/auth/login` | Login, returns JWT |
| POST | `/auth/refresh` | Refresh JWT token |

### Alerts
| Method | Path | Description |
|--------|------|-------------|
| GET | `/alerts` | List user's alerts (paginated) |
| POST | `/alerts` | Create new alert rule |
| GET | `/alerts/:id` | Get alert details + history |
| PUT | `/alerts/:id` | Update alert rule |
| DELETE | `/alerts/:id` | Cancel/delete alert |

### Prices
| Method | Path | Description |
|--------|------|-------------|
| GET | `/prices/latest` | Latest prices for all tracked symbols |
| GET | `/prices/history/:symbol` | Price history (query params: interval, limit) |

### Analytics
| Method | Path | Description |
|--------|------|-------------|
| GET | `/analytics/top-drops` | Top-K biggest drops (query: window=1h\|24h) |
| GET | `/analytics/stats` | System stats (total alerts, triggers today) |

### WebSocket
| Path | Description |
|------|-------------|
| `/ws` | Real-time prices + alert notifications (JWT auth via query param) |

**WebSocket message types:**
```typescript
// Server → Client
{ type: "price", data: PriceUpdateEvent }
{ type: "alert-triggered", data: AlertTriggerEvent }
{ type: "error", message: string }

// Client → Server
{ type: "subscribe", symbols: string[] }
{ type: "unsubscribe", symbols: string[] }
```

---

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Kafka over RabbitMQ | Kafka | Partition-based ordering, consumer groups, event replay. Better system design talking point. |
| PostgreSQL over MongoDB | PostgreSQL | Alert rules are relational (user → alerts). Structured queries for history. ACID for alert state transitions. |
| Redis for rate limiting | Sliding window | More accurate than fixed window. O(1) check with sorted sets. |
| Redis for Top-K | Sorted sets | Native ZADD/ZREVRANGE. No custom data structure needed. O(log N) updates. |
| JWT over sessions | JWT | Stateless auth. No session store needed. Works naturally with WebSocket (token in query). |
| Monorepo (npm workspaces) | Single repo | Shared types, easy local dev, single Docker Compose. Simpler than multi-repo for a side project. |
| Resend over SendGrid | Resend | Modern API, generous free tier (100 emails/day), simpler DX. |
| Separate services (not microservices) | Process-level separation | Each service is a separate Node.js process sharing a codebase. Not full microservices (no service mesh, no K8s). Right level of complexity for a side project. |

---

## Rate Limiting Strategy

### API Rate Limiting
- **Algorithm:** Redis sliding window log
- **Default limits:**
  - Unauthenticated: 20 requests/minute
  - Authenticated: 100 requests/minute
  - Alert creation: 10/hour per user

### Notification Rate Limiting
- **Per-user:** Max 10 notifications/hour (configurable)
- **Global:** Max 1000 emails/hour (Resend free tier)
- **Implementation:** Redis sorted set with timestamps, ZRANGEBYSCORE to count recent entries
