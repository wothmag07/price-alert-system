import { Kafka, Producer } from "kafkajs";

let producer: Producer | null = null;

export async function getProducer(): Promise<Producer> {
  if (producer) return producer;

  const kafka = new Kafka({
    clientId: "price-alert-system",
    brokers: (process.env.KAFKA_BROKERS ?? "localhost:9092").split(","),
  });

  producer = kafka.producer();
  await producer.connect();
  return producer;
}

export async function disconnectProducer(): Promise<void> {
  if (producer) {
    await producer.disconnect();
    producer = null;
  }
}
