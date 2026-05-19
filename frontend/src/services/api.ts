import { getAccessToken } from "./session";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

export type TokenPair = {
  access_token: string;
  refresh_token: string;
  token_type: string;
};

export type SystemMetrics = {
  cpu_percent: number;
  ram_total_bytes: number;
  ram_used_bytes: number;
  ram_percent: number;
  temperature_celsius: number | null;
  uptime_seconds: number;
  load_average: [number, number, number];
  disks: Array<{
    mountpoint: string;
    device: string;
    filesystem: string;
    total_bytes: number;
    used_bytes: number;
    free_bytes: number;
    percent: number;
  }>;
  network: {
    bytes_sent: number;
    bytes_recv: number;
    packets_sent: number;
    packets_recv: number;
  };
};

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers);
  headers.set("Content-Type", "application/json");

  const token = getAccessToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, { ...options, headers });
  if (!response.ok) {
    const errorBody = await response.json().catch(() => ({ detail: "Erro inesperado." }));
    throw new Error(errorBody.detail ?? "Erro inesperado.");
  }
  return response.json() as Promise<T>;
}

export function login(username: string, password: string) {
  return request<TokenPair>("/auth/login", {
    method: "POST",
    body: JSON.stringify({ username, password })
  });
}

export function fetchMetrics() {
  return request<SystemMetrics>("/metrics");
}

export function metricsWebSocketUrl() {
  const token = getAccessToken();
  const apiBase = API_BASE_URL.startsWith("http")
    ? API_BASE_URL.replace(/^http/, "ws")
    : `${location.origin.replace(/^http/, "ws")}${API_BASE_URL}`;
  return `${apiBase}/ws/metrics?token=${encodeURIComponent(token ?? "")}`;
}
