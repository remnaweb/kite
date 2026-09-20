import { FormEvent, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api, Inbound } from "../api";
import Modal from "../components/Modal";

export default function Inbounds() {
  const [items, setItems] = useState<Inbound[]>([]);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    setItems(await api.inbounds());
  }

  useEffect(() => {
    load().catch((e) => setError(e.message));
  }, []);

  return (
    <>
      <div className="top">
        <div>
          <h1>Инбаунды</h1>
          <p>Порты, Reality и WS. Клиенты живут внутри каждого инбаунда.</p>
        </div>
        <button className="btn primary" onClick={() => setOpen(true)}>
          Создать
        </button>
      </div>
      {error && <div className="error">{error}</div>}
      <div className="sheet table-wrap">
        <table>
          <thead>
            <tr>
              <th>Имя</th>
              <th>Порт</th>
              <th>Протокол</th>
              <th>Клиенты</th>
              <th>Статус</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {items.map((ib) => (
              <tr key={ib.id}>
                <td>
                  <Link to={`/inbounds/${ib.id}`}>{ib.remark}</Link>
                </td>
                <td className="mono">{ib.port}</td>
                <td>
                  VLESS {ib.security === "reality" ? "Reality" : ib.network.toUpperCase()}
                </td>
                <td className="mono">{ib.clients?.length ?? 0}</td>
                <td>
                  <span className={"pill " + (ib.enable ? "good" : "bad")}>{ib.enable ? "вкл" : "выкл"}</span>
                </td>
                <td className="row-actions">
                  <button
                    className="btn"
                    onClick={async () => {
                      await api.updateInbound(ib.id, { ...ib, enable: !ib.enable });
                      load();
                    }}
                  >
                    {ib.enable ? "Выкл" : "Вкл"}
                  </button>
                  <button
                    className="btn danger"
                    onClick={async () => {
                      if (!confirm("Удалить инбаунд и всех клиентов?")) return;
                      await api.deleteInbound(ib.id);
                      load();
                    }}
                  >
                    Удалить
                  </button>
                </td>
              </tr>
            ))}
            {!items.length && (
              <tr>
                <td colSpan={6} className="hint">
                  Пока пусто — создай первый инбаунд.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {open && (
        <CreateInbound
          onClose={() => setOpen(false)}
          onCreated={() => {
            setOpen(false);
            load();
          }}
        />
      )}
    </>
  );
}

function CreateInbound({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [kind, setKind] = useState<"reality" | "ws">("reality");
  const [remark, setRemark] = useState("");
  const [port, setPort] = useState(kind === "reality" ? 443 : 8080);
  const [dest, setDest] = useState("www.cloudflare.com:443");
  const [serverNames, setServerNames] = useState("www.cloudflare.com");
  const [wsPath, setWsPath] = useState("/ws");
  const [error, setError] = useState("");

  async function submit(e: FormEvent) {
    e.preventDefault();
    try {
      await api.createInbound({
        remark,
        port,
        protocol: "vless",
        network: kind === "reality" ? "tcp" : "ws",
        security: kind === "reality" ? "reality" : "none",
        dest,
        serverNames,
        wsPath,
        sniffing: true,
        enable: true,
      });
      onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "ошибка");
    }
  }

  return (
    <Modal title="Новый инбаунд" onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-grid">
          <div className="full">
            <label>Тип</label>
            <select
              value={kind}
              onChange={(e) => {
                const k = e.target.value as "reality" | "ws";
                setKind(k);
                setPort(k === "reality" ? 443 : 8080);
              }}
            >
              <option value="reality">VLESS + Reality</option>
              <option value="ws">VLESS + WebSocket</option>
            </select>
          </div>
          <div>
            <label>Имя</label>
            <input value={remark} onChange={(e) => setRemark(e.target.value)} placeholder="NL-1" />
          </div>
          <div>
            <label>Порт</label>
            <input type="number" value={port} onChange={(e) => setPort(Number(e.target.value))} />
          </div>
          {kind === "reality" ? (
            <>
              <div>
                <label>Dest</label>
                <input value={dest} onChange={(e) => setDest(e.target.value)} />
              </div>
              <div>
                <label>SNI / serverNames</label>
                <input value={serverNames} onChange={(e) => setServerNames(e.target.value)} />
              </div>
            </>
          ) : (
            <div className="full">
              <label>WS path</label>
              <input value={wsPath} onChange={(e) => setWsPath(e.target.value)} />
            </div>
          )}
        </div>
        {error && <div className="error">{error}</div>}
        <div className="modal-actions">
          <button type="button" className="btn" onClick={onClose}>
            Отмена
          </button>
          <button className="btn primary">Создать</button>
        </div>
      </form>
    </Modal>
  );
}
