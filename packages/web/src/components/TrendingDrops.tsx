import { useState, useEffect } from "react";
import { analytics, type TopDrop } from "../lib/api";

const WINDOWS = ["1m", "5m", "1h", "24h"];

export default function TrendingDrops() {
  const [window, setWindow] = useState("1h");
  const [drops, setDrops] = useState<TopDrop[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    analytics
      .topDrops(window)
      .then((res) => setDrops(res.drops))
      .catch(() => setDrops([]))
      .finally(() => setLoading(false));

    const interval = setInterval(() => {
      analytics
        .topDrops(window)
        .then((res) => setDrops(res.drops))
        .catch(() => {});
    }, 10_000);

    return () => clearInterval(interval);
  }, [window]);

  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-300">Trending Drops</h2>
        <div className="flex gap-1">
          {WINDOWS.map((w) => (
            <button
              key={w}
              onClick={() => setWindow(w)}
              className={`rounded px-2 py-0.5 text-xs ${
                window === w
                  ? "bg-blue-600 text-white"
                  : "bg-gray-800 text-gray-400 hover:text-white"
              }`}
            >
              {w}
            </button>
          ))}
        </div>
      </div>

      {loading ? (
        <p className="text-gray-500">Loading analytics...</p>
      ) : drops.length === 0 ? (
        <p className="text-gray-500">No significant drops detected.</p>
      ) : (
        <div className="space-y-2">
          {drops.map((drop, i) => (
            <div
              key={drop.symbol}
              className="flex items-center justify-between rounded bg-gray-800/50 px-3 py-2"
            >
              <div className="flex items-center gap-2">
                <span className="w-5 text-center text-xs text-gray-500">
                  {i + 1}
                </span>
                <span className="font-mono text-sm text-white">
                  {drop.symbol.replace("USDT", "")}
                </span>
              </div>
              <span className="font-mono text-sm text-red-400">
                -{drop.dropPct.toFixed(2)}%
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
