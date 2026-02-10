// ── Price Events ──

export interface PriceUpdateEvent {
  symbol: string;
  price: number;
  volume: number;
  change24h: number;
  timestamp: number;
}

// ── Alert Types ──

export type AlertCondition =
  | "PRICE_ABOVE"
  | "PRICE_BELOW"
  | "PCT_CHANGE_ABOVE"
  | "PCT_CHANGE_BELOW";

export type AlertStatus = "ACTIVE" | "TRIGGERED" | "CANCELLED";

export interface Alert {
  id: string;
  userId: string;
  symbol: string;
  condition: AlertCondition;
  threshold: number;
  status: AlertStatus;
  createdAt: Date;
  triggeredAt: Date | null;
}

export interface AlertTriggerEvent {
  alertId: string;
  userId: string;
  symbol: string;
  condition: AlertCondition;
  threshold: number;
  triggeredPrice: number;
  timestamp: number;
}

// ── User Types ──

export interface User {
  id: string;
  email: string;
  password: string;
  createdAt: Date;
  updatedAt: Date;
}

// ── WebSocket Messages ──

export type WsServerMessage =
  | { type: "price"; data: PriceUpdateEvent }
  | { type: "alert-triggered"; data: AlertTriggerEvent }
  | { type: "error"; message: string };

export type WsClientMessage =
  | { type: "subscribe"; symbols: string[] }
  | { type: "unsubscribe"; symbols: string[] };

// ── Kafka Topics ──

export const KAFKA_TOPICS = {
  PRICE_UPDATES: "price-updates",
  ALERT_TRIGGERS: "alert-triggers",
} as const;

// ── Redis Keys ──

export const REDIS_KEYS = {
  latestPrice: (symbol: string) => `price:latest:${symbol}`,
  activeAlerts: (symbol: string) => `alerts:active:${symbol}`,
  rateLimit: (userId: string, window: string) =>
    `rate-limit:${userId}:${window}`,
  topDrops: (window: string) => `top-drops:${window}`,
  wsNotify: (userId: string) => `ws:notify:${userId}`,
} as const;
