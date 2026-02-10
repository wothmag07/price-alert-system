import { KAFKA_TOPICS } from "@price-alert/shared";

console.log("[Price Ingestion] Service starting...");
console.log(`[Price Ingestion] Will publish to topic: ${KAFKA_TOPICS.PRICE_UPDATES}`);

// TODO: Connect to Binance WebSocket
// TODO: Parse and normalize price data
// TODO: Publish price events to Kafka
// TODO: Cache latest prices in Redis

process.on("SIGTERM", () => {
  console.log("[Price Ingestion] Shutting down...");
  process.exit(0);
});
