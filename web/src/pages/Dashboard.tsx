import { useEffect, useState } from "react";
import { api, fmtBytes, fmtUptime, SystemStatus } from "../api";

export default function Dashboard() {
  const [sys, setSys] = useState<SystemStatus | null>(null);
  const [error, setError] = useState("");

  async function load() {
    try {
      setSys(await api.system());
    } catch (e) {
      setError(e instanceof Error ? e.message : "ошибка");
    }
  }

  useEffect(() => {
    load();
    const t = setInterval(load, 5000);
    return () => clearInterval(t);
  }, []);

  if (!sys) return <div>{error || "загрузка…"}</div>;

  const memPct = sys.memTotal ? Math.round((sys.memUsed / sys.memTotal) * 100) : 0;
  const diskPct = sys.diskTotal ? Math.round((sys.diskUsed / sys.diskTotal) * 100) : 0;

  return (
    <>
      <div className="top">
        <div>
          <h1>Обзор</h1>
          <p>
            {sys.hostname} · {sys.os}
          </p>
        </div>
        <button className="btn" onClick={() => api.restartXray().then(load).catch((e) => setError(e.message))}>
          Перезапустить Xray
        </button>
      </div>
      {error && <div className="error">{error}</div>}
      <div className="statrow">
        <div className="stat">
          <small>CPU</small>
          <b className="mono">{sys.cpu.toFixed(0)}%</b>
        </div>
        <div className="stat">
          <small>Память</small>
          <b className="mono">{fmtBytes(sys.memUsed)}</b>
          <div className="hint">
            {fmtBytes(sys.memTotal)} · {memPct}%
          </div>
        </div>
        <div className="stat">
          <small>Диск</small>
          <b className="mono">{fmtBytes(sys.diskUsed)}</b>
          <div className="hint">
            {fmtBytes(sys.diskTotal)} · {diskPct}%
          </div>
        </div>
        <div className="stat">
          <small>Аптайм</small>
          <b className="mono">{fmtUptime(sys.uptime)}</b>
          <div className="hint">Kite {fmtUptime(sys.panelUptime)}</div>
        </div>
      </div>
      <div className="statrow">
        <div className="stat">
          <small>Инбаунды</small>
          <b className="mono">{sys.inbounds}</b>
        </div>
        <div className="stat">
          <small>Клиенты</small>
          <b className="mono">{sys.active}</b>
          <div className="hint">всего {sys.clients}</div>
        </div>
        <div className="stat">
          <small>Трафик</small>
          <b className="mono">{fmtBytes(sys.trafficUp + sys.trafficDown)}</b>
          <div className="hint">
            ↑ {fmtBytes(sys.trafficUp)} · ↓ {fmtBytes(sys.trafficDown)}
          </div>
        </div>
        <div className="stat">
          <small>Xray</small>
          <b>{sys.xray.running ? "online" : "offline"}</b>
          <div className="hint">{sys.xrayVersion || sys.xray.binary}</div>
          {sys.xray.lastError && <div className="error">{sys.xray.lastError}</div>}
        </div>
      </div>
    </>
  );
}
