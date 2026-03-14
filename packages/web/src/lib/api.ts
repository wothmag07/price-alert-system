const BASE = "/api";

async function request<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const token = localStorage.getItem("accessToken");
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options?.headers as Record<string, string>) ?? {}),
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`${BASE}${path}`, { ...options, headers });

  if (res.status === 401) {
    // Try refresh
    const refreshed = await tryRefresh();
    if (refreshed) {
      headers["Authorization"] = `Bearer ${localStorage.getItem("accessToken")}`;
      const retry = await fetch(`${BASE}${path}`, { ...options, headers });
      if (!retry.ok) throw new Error(await retry.text());
      return retry.json();
    }
    localStorage.removeItem("accessToken");
    localStorage.removeItem("refreshToken");
    window.location.href = "/login";
    throw new Error("Unauthorized");
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }

  return res.json();
}

async function tryRefresh(): Promise<boolean> {
  const refreshToken = localStorage.getItem("refreshToken");
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${BASE}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refreshToken }),
    });
    if (!res.ok) return false;
    const data = await res.json();
    localStorage.setItem("accessToken", data.accessToken);
    localStorage.setItem("refreshToken", data.refreshToken);
    return true;
  } catch {
    return false;
  }
}

// Auth
export const auth = {
  register: (email: string, password: string) =>
    request<{ user: { id: string; email: string }; accessToken: string; refreshToken: string }>(
      "/auth/register",
      { method: "POST", body: JSON.stringify({ email, password }) }
    ),
  login: (email: string, password: string) =>
    request<{ user: { id: string; email: string }; accessToken: string; refreshToken: string }>(
      "/auth/login",
      { method: "POST", body: JSON.stringify({ email, password }) }
    ),
  me: () => request<{ user: { id: string; email: string } }>("/auth/me"),
};

// Alerts
export interface Alert {
  id: string;
  symbol: string;
  condition: string;
  threshold: number;
  status: string;
  createdAt: string;
  triggeredAt: string | null;
}

export const alerts = {
  list: (page = 1, limit = 20) =>
    request<{ alerts: Alert[]; total: number; page: number; limit: number }>(
      `/alerts?page=${page}&limit=${limit}`
    ),
  create: (data: { symbol: string; condition: string; threshold: number }) =>
    request<Alert>("/alerts", { method: "POST", body: JSON.stringify(data) }),
  get: (id: string) =>
    request<Alert & { history: unknown[] }>(`/alerts/${id}`),
  delete: (id: string) =>
    request<{ message: string }>(`/alerts/${id}`, { method: "DELETE" }),
};

// Prices
export const prices = {
  latest: () =>
    request<{ prices: Record<string, { symbol: string; price: number; volume: number; change24h: number; timestamp: number }> }>(
      "/prices/latest"
    ),
};

// Analytics
export interface TopDrop {
  symbol: string;
  dropPct: number;
}

export const analytics = {
  topDrops: (window = "1h", limit = 10) =>
    request<{ window: string; drops: TopDrop[] }>(
      `/analytics/top-drops?window=${window}&limit=${limit}`
    ),
};
