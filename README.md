# Price Alert System

A real-time cryptocurrency price alert system. Users set price thresholds and receive notifications (email + in-app) when conditions are met. Built with an event-driven microservices architecture using Go, Kafka, Redis, and PostgreSQL.

## Architecture

```
Binance WS ──► Price Ingestion ──► Kafka [price-updates]
                                        │
                        ┌───────────────┼───────────────┐
                        ▼               ▼               ▼
                  Alert Engine    Analytics Service   API Server
                        │               │           (REST + WS)
                        ▼               ▼
              Kafka [alert-triggers]  Redis (Top-K)
                        │
                        ▼
              Notification Service
                   │         │
                   ▼         ▼
                Email    WebSocket Push
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend Services | Go (Gin, gorilla/websocket, kafka-go, go-redis, pgx) |
| Event Streaming | Apache Kafka |
| Database | PostgreSQL 16 |
| Cache / Real-time | Redis 7 |
| Email | Resend API |
| Frontend | React 18 + Vite + Tailwind CSS |
| Infrastructure | Docker Compose |

## Services

| Service | Description | Port |
|---------|-------------|------|
| **price-ingestion** | Connects to Binance WebSocket, publishes price events to Kafka | — |
| **api-server** | REST API + WebSocket server (auth, alerts CRUD, prices, analytics) | 3000 |
| **alert-engine** | Matches prices against user alert rules, publishes triggers | — |
| **notification-service** | Delivers email + WebSocket push notifications | — |
| **analytics-service** | Computes Top-K price drops across rolling windows | — |
| **web** | React frontend dashboard | 5173 |

## Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/)

## Quick Start

### 1. Clone and configure

```bash
git clone https://github.com/wothmag07/price-alert-system.git
cd price-alert-system
cp .env.example .env
```

### 2. Start infrastructure

```bash
docker compose up -d zookeeper kafka postgres redis
```

Wait ~10 seconds for Kafka to be ready.

### 3. Run database migrations

Migrations run automatically when the api-server starts. Alternatively:

```bash
cd services/api-server && go run .
# The server will create tables on startup, then Ctrl+C to stop
```

### 4. Start backend services

Open separate terminals for each:

```bash
# Terminal 1 — API Server
cd services/api-server && go run .

# Terminal 2 — Price Ingestion
cd services/price-ingestion && go run .

# Terminal 3 — Alert Engine
cd services/alert-engine && go run .

# Terminal 4 — Analytics Service
cd services/analytics-service && go run .

# Terminal 5 — Notification Service
cd services/notification-service && go run .
```

Or use the npm scripts from the project root:

```bash
npm run dev:api-server
npm run dev:price-ingestion
npm run dev:alert-engine
npm run dev:analytics
npm run dev:notification-service
```

### 5. Start frontend

```bash
cd packages/web
npm install
npm run dev
```

Open http://localhost:5173 — register an account and start creating alerts.

### 6. Run everything with Docker

```bash
docker compose up --build
```

## API Endpoints

### Auth
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | — | Create account |
| POST | `/auth/login` | — | Login, returns JWT |
| POST | `/auth/refresh` | — | Refresh tokens |
| GET | `/auth/me` | JWT | Current user info |

### Alerts
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/alerts` | JWT | List alerts (paginated) |
| POST | `/alerts` | JWT | Create alert rule |
| GET | `/alerts/:id` | JWT | Alert detail + trigger history |
| PUT | `/alerts/:id` | JWT | Update alert |
| DELETE | `/alerts/:id` | JWT | Delete alert |

### Prices
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/prices/latest` | JWT | Latest prices from Redis cache |
| GET | `/prices/history/:symbol` | JWT | Aggregated price history |

### Analytics
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/analytics/top-drops?window=1h` | JWT | Top-K biggest price drops |

### WebSocket
| Path | Auth | Description |
|------|------|-------------|
| `/ws?token=JWT` | Token in query | Live prices + alert notifications |

## Alert Conditions

| Condition | Description |
|-----------|-------------|
| `PRICE_ABOVE` | Triggers when price >= threshold |
| `PRICE_BELOW` | Triggers when price <= threshold |
| `PCT_CHANGE_ABOVE` | Triggers when 24h % change >= threshold |
| `PCT_CHANGE_BELOW` | Triggers when 24h % change <= -threshold |

## Environment Variables

See [.env.example](.env.example) for all configuration options.

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_HOST` | localhost | PostgreSQL host |
| `REDIS_HOST` | localhost | Redis host |
| `KAFKA_BROKERS` | localhost:9092 | Kafka broker addresses |
| `JWT_SECRET` | dev-secret... | JWT signing secret |
| `API_PORT` | 3000 | API server port |
| `RESEND_API_KEY` | — | Resend API key for email notifications |
| `TRACKED_SYMBOLS` | btcusdt,ethusdt,solusdt,... | Binance symbols to track |

## Project Structure

```
├── services/
│   ├── price-ingestion/     # Go — Binance WS → Kafka
│   ├── api-server/          # Go — REST + WS (Gin)
│   ├── alert-engine/        # Go — Price matching
│   ├── notification-service/# Go — Email + WS push
│   └── analytics-service/   # Go — Top-K drops
├── packages/
│   └── web/                 # React frontend
├── docs/
│   ├── ARCHITECTURE.md      # System design
│   └── PLAN.md              # Project milestones
├── scripts/
│   └── seed.ts              # DB seed script
├── docker-compose.yml
└── .env.example
```

## License

See [LICENSE](LICENSE).
