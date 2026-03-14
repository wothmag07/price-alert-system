import type { PriceUpdate } from "../hooks/useWebSocket";

interface Props {
  prices: Record<string, PriceUpdate>;
  connected: boolean;
}

function formatPrice(price: number): string {
  if (price >= 1) return price.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  return price.toFixed(6);
}

function formatVolume(vol: number): string {
  if (vol >= 1_000_000_000) return (vol / 1_000_000_000).toFixed(2) + "B";
  if (vol >= 1_000_000) return (vol / 1_000_000).toFixed(2) + "M";
  if (vol >= 1_000) return (vol / 1_000).toFixed(2) + "K";
  return vol.toFixed(2);
}

export default function PriceTicker({ prices, connected }: Props) {
  const symbols = Object.values(prices).sort((a, b) => a.symbol.localeCompare(b.symbol));

  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-300">Live Prices</h2>
        <span
          className={`h-2 w-2 rounded-full ${connected ? "bg-green-500" : "bg-red-500"}`}
          title={connected ? "Connected" : "Disconnected"}
        />
      </div>

      {symbols.length === 0 ? (
        <p className="text-gray-500">
          {connected ? "Waiting for price data..." : "Connecting to WebSocket..."}
        </p>
      ) : (
        <div className="space-y-2">
          {symbols.map((p) => (
            <div
              key={p.symbol}
              className="flex items-center justify-between rounded bg-gray-800/50 px-3 py-2"
            >
              <div>
                <span className="font-mono font-medium text-white">
                  {p.symbol.replace("USDT", "")}
                </span>
                <span className="ml-1 text-xs text-gray-500">/ USDT</span>
              </div>
              <div className="text-right">
                <div className="font-mono text-white">${formatPrice(p.price)}</div>
                <div className="flex items-center gap-2">
                  <span
                    className={`text-xs font-mono ${
                      p.change24h >= 0 ? "text-green-400" : "text-red-400"
                    }`}
                  >
                    {p.change24h >= 0 ? "+" : ""}
                    {p.change24h.toFixed(2)}%
                  </span>
                  <span className="text-xs text-gray-500">
                    Vol: {formatVolume(p.volume)}
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
