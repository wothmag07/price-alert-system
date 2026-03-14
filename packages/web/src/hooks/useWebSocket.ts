import { useEffect, useRef, useState, useCallback } from "react";

export interface PriceUpdate {
  symbol: string;
  price: number;
  volume: number;
  change24h: number;
  timestamp: number;
}

export interface AlertTriggered {
  alertId: string;
  userId: string;
  symbol: string;
  condition: string;
  threshold: number;
  triggeredPrice: number;
  timestamp: number;
}

type WsMessage =
  | { type: "price"; data: PriceUpdate }
  | { type: "alert-triggered"; data: AlertTriggered };

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const [prices, setPrices] = useState<Record<string, PriceUpdate>>({});
  const [notifications, setNotifications] = useState<AlertTriggered[]>([]);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const token = localStorage.getItem("accessToken");
    if (!token) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const ws = new WebSocket(`${protocol}//${window.location.host}/api/ws?token=${token}`);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
    };

    ws.onmessage = (event) => {
      const msg: WsMessage = JSON.parse(event.data);
      if (msg.type === "price") {
        setPrices((prev) => ({ ...prev, [msg.data.symbol]: msg.data }));
      } else if (msg.type === "alert-triggered") {
        setNotifications((prev) => [msg.data, ...prev.slice(0, 19)]);
      }
    };

    ws.onclose = () => setConnected(false);
    ws.onerror = () => setConnected(false);

    return () => {
      ws.close();
    };
  }, []);

  const subscribe = useCallback((symbols: string[]) => {
    wsRef.current?.send(JSON.stringify({ type: "subscribe", symbols }));
  }, []);

  const dismissNotification = useCallback((index: number) => {
    setNotifications((prev) => prev.filter((_, i) => i !== index));
  }, []);

  return { prices, notifications, connected, subscribe, dismissNotification };
}
