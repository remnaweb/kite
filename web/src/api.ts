export type Inbound = {
  id: number;
  remark: string;
  enable: boolean;
  listen: string;
  port: number;
  protocol: string;
  network: string;
  security: string;
  dest: string;
  serverNames: string;
  publicKey: string;
  shortIds: string;
  fingerprint: string;
  spiderX: string;
  wsPath: string;
  wsHost: string;
  sniffing: boolean;
  clients?: Client[];
};

export type Client = {
  id: number;
  inboundId: number;
  uuid: string;
  email: string;
  name: string;
  enable: boolean;
  expiryTime: number;
  totalBytes: number;
  up: number;
  down: number;
  subId: string;
};

export type SystemStatus = {
  cpu: number;
  memUsed: number;
  memTotal: number;
  diskUsed: number;
  diskTotal: number;
  netUp: number;
  netDown: number;
  hostname: string;
  os: string;
  uptime: number;
  panelUptime: number;
  inbounds: number;
  clients: number;
  active: number;
  trafficUp: number;
  trafficDown: number;
  xrayVersion: string;
  xray: {
    running: boolean;
    binary: string;
    binaryExists: boolean;
    lastError: string;
  };
};

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(init?.headers || {}) },
    ...init,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok || data.success === false) {
    throw new Error(data.message || `HTTP ${res.status}`);
  }
  return data.data as T;
}

export const api = {
  login: (username: string, password: string) =>
    req("/api/login", { method: "POST", body: JSON.stringify({ username, password }) }),
  logout: () => req("/api/logout", { method: "POST" }),
  me: () => req<{ username: string }>("/api/me"),
  changePassword: (current: string, next: string) =>
    req("/api/me/password", { method: "POST", body: JSON.stringify({ current, next }) }),
  system: () => req<SystemStatus>("/api/system"),
  inbounds: () => req<Inbound[]>("/api/inbounds"),
  inbound: (id: number) => req<Inbound>(`/api/inbounds/${id}`),
  createInbound: (body: Partial<Inbound>) =>
    req<Inbound>("/api/inbounds", { method: "POST", body: JSON.stringify(body) }),
  updateInbound: (id: number, body: Partial<Inbound>) =>
    req<Inbound>(`/api/inbounds/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteInbound: (id: number) => req(`/api/inbounds/${id}`, { method: "DELETE" }),
  createClient: (inboundId: number, body: Record<string, unknown>) =>
    req<Client>(`/api/inbounds/${inboundId}/clients`, { method: "POST", body: JSON.stringify(body) }),
  updateClient: (id: number, body: Record<string, unknown>) =>
    req<Client>(`/api/clients/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteClient: (id: number) => req(`/api/clients/${id}`, { method: "DELETE" }),
  resetTraffic: (id: number) => req<Client>(`/api/clients/${id}/reset-traffic`, { method: "POST" }),
  share: (id: number) => req<{ link: string; sub: string }>(`/api/clients/${id}/share`),
  restartXray: () => req("/api/xray/restart", { method: "POST" }),
  settings: () => req<{ publicHost: string; publicPort: string; panelTitle: string }>("/api/settings"),
  saveSettings: (body: { publicHost: string; publicPort: string; panelTitle: string }) =>
    req("/api/settings", { method: "PUT", body: JSON.stringify(body) }),
};

export function fmtBytes(n: number) {
  if (!n) return "0 B";
  const u = ["B", "KB", "MB", "GB", "TB"];
  let i = 0;
  let v = n;
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(v >= 10 || i === 0 ? 0 : 1)} ${u[i]}`;
}

export function fmtUptime(sec: number) {
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const m = Math.floor((sec % 3600) / 60);
  if (d) return `${d}д ${h}ч`;
  if (h) return `${h}ч ${m}м`;
  return `${m}м`;
}
