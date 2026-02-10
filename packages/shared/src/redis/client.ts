import IORedis from "ioredis";

const { Redis } = IORedis;

let redis: IORedis.Redis | null = null;

export function getRedis(): IORedis.Redis {
  if (redis) return redis;

  redis = new Redis({
    host: process.env.REDIS_HOST ?? "localhost",
    port: Number(process.env.REDIS_PORT ?? 6379),
  });

  return redis;
}

export async function disconnectRedis(): Promise<void> {
  if (redis) {
    await redis.quit();
    redis = null;
  }
}
