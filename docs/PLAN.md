# Price Alert System — Project Plan

## Overview

A real-time price alert system for cryptocurrencies. Users set price thresholds, and the system monitors live prices via Binance WebSocket, triggering notifications (email + in-app) when conditions are met.

## System Design Concepts Covered

| # | Concept | How It Maps |
|---|---------|-------------|
| 1 | **Notification Service** | Email + WebSocket push with delivery tracking, fan-out per user |
| 2 | **Rate Limiter** | Redis sliding window on API endpoints + per-user notification throttling |
| 3 | **Analytics / Time-Series** | Price history aggregation, alert trigger stats |
| 4 | **Top-K (Trending)** | Redis sorted sets tracking biggest price drops in rolling windows |
| 5 | **Stock/Crypto Prices** | Real-time price ingestion via WebSocket, Kafka streaming |
| 6 | **Event-Driven Architecture** | Kafka-based decoupled services, pub/sub pattern |
| 7 | **Pub/Sub System** | Price updates fan out to alert engine + analytics + WebSocket clients |

## Tech Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Language | TypeScript (Node.js) | All services, shared types |
| Event Streaming | Apache Kafka | Price events, alert triggers |
| Primary DB | PostgreSQL | Users, alerts, price history |
| Cache / Real-time | Redis | Rate limiting, Top-K, pub/sub, caching |
| Email | Resend | Transactional alert emails |
| Real-time Push | WebSocket (ws) | Live prices + alert notifications to frontend |
| Frontend | React + Vite + Tailwind | Dashboard UI |
| Auth | JWT (jsonwebtoken + bcrypt) | User authentication |
| Infra | Docker Compose | Local development orchestration |

## Architecture (5 Services + Frontend)

```
Binance WS ──► Price Ingestion ──► Kafka [price-updates]
                                        │
                        ┌───────────────┼───────────────┐
                        ▼               ▼               ▼
                  Alert Engine    Analytics Service   API Server
                        │               │           (WebSocket)
                        ▼               ▼
              Kafka [alert-triggers]  Redis (Top-K)
                        │
                        ▼
              Notification Service
                   │         │
                   ▼         ▼
                Email    WebSocket Push
```

## Milestones

### Milestone 1: Foundation
- [x] Project plan and architecture docs
- [ ] Monorepo scaffold (npm workspaces)
- [ ] Docker Compose (Kafka, Postgres, Redis)
- [ ] Shared package (types, DB/Redis/Kafka clients)

### Milestone 2: Price Ingestion Pipeline
- [ ] Binance WebSocket connection with reconnection logic
- [ ] Price events published to Kafka `price-updates` topic
- [ ] Price caching in Redis (latest price per asset)
- [ ] Price history persisted to PostgreSQL

### Milestone 3: API Server + Auth
- [ ] User registration and login (JWT)
- [ ] CRUD endpoints for alert rules
- [ ] WebSocket server streaming live prices to frontend
- [ ] Rate limiting middleware (Redis sliding window)

### Milestone 4: Alert Engine
- [ ] Kafka consumer for price updates
- [ ] Alert rule matching (price > threshold, price < threshold, % change)
- [ ] Active alert caching in Redis for fast lookups
- [ ] Triggered alerts published to Kafka `alert-triggers`

### Milestone 5: Notification Service
- [ ] Kafka consumer for alert triggers
- [ ] Email notifications via Resend
- [ ] WebSocket push notifications
- [ ] Per-user rate limiting (max N notifications per hour)
- [ ] Delivery tracking (alert_history table)

### Milestone 6: Analytics + Top-K
- [ ] Kafka consumer for price updates
- [ ] Rolling window price change calculation
- [ ] Top-K trending drops via Redis sorted sets
- [ ] API endpoint to query trending data

### Milestone 7: Frontend
- [ ] Login / Register pages
- [ ] Live price dashboard (WebSocket)
- [ ] Alert CRUD interface
- [ ] Real-time alert notifications (toast/banner)
- [ ] Top-K trending drops display

### Milestone 8: Polish
- [ ] Error handling and graceful shutdown across services
- [ ] Health check endpoints
- [ ] Dockerfiles for all services
- [ ] End-to-end testing
- [ ] README with setup instructions

## Build Order

Start with Milestones 1-2-3 sequentially (foundation must exist first). Then Milestones 4 and 5 build on each other. Milestone 6 is independent and can be built in parallel with 4-5. Milestone 7 can start after Milestone 3 (API must exist). Milestone 8 is final polish.

```
M1 ──► M2 ──► M3 ──► M4 ──► M5
                │             │
                ├──► M6 ──────┤
                │             │
                └──► M7 ──────┴──► M8
```
