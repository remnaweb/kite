import { FormEvent, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { QRCodeSVG } from "qrcode.react";
import { api, fmtBytes, Inbound } from "../api";
import Modal from "../components/Modal";

export default function InboundDetail() {
  const { id } = useParams();
  const inboundId = Number(id);
  const [ib, setIb] = useState<Inbound | null>(null);
  const [open, setOpen] = useState(false);
  const [share, setShare] = useState<{ name: string; link: string; sub: string } | null>(null);
  const [error, setError] = useState("");

  async function load() {
    setIb(await api.inbound(inboundId));
  }

  useEffect(() => {
    load().catch((e) => setError(e.message));
  }, [inboundId]);

  if (!ib) return <div>{error || "загрузка…"}</div>;

  return (
    <>
      <div className="top">
        <div>
          <h1>{ib.remark}</h1>
          <p>
            <Link to="/inbounds">Инбаунды</Link> · порт {ib.port} · VLESS{" "}
            {ib.security === "reality" ? "Reality" : "WS"}
          </p>
        </div>
        <button className="btn primary" onClick={() => setOpen(true)}>
          Клиент
        </button>
      </div>
      {error && <div className="error">{error}</div>}
      <div className="sheet table-wrap">
        <table>
          <thead>
            <tr>
              <th>Имя</th>
              <th>Email</th>
              <th>Трафик</th>
              <th>Лимит</th>
              <th>Срок</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {(ib.clients || []).map((c) => (
              <tr key={c.id}>
                <td>{c.name}</td>
                <td className="mono">{c.email}</td>
                <td className="mono">
                  {fmtBytes(c.up + c.down)}
                  <div className="hint">
                    ↑ {fmtBytes(c.up)} ↓ {fmtBytes(c.down)}
                  </div>
                </td>
                <td className="mono">{c.totalBytes ? fmtBytes(c.totalBytes) : "∞"}</td>
                <td>{c.expiryTime ? new Date(c.expiryTime).toLocaleDateString() : "∞"}</td>
                <td className="row-actions">
                  <button
                    className="btn"
                    onClick={async () => {
                      const s = await api.share(c.id);
                      setShare({ name: c.name, link: s.link, sub: s.sub });
                    }}
                  >
                    QR
                  </button>
                  <button className="btn" onClick={() => api.resetTraffic(c.id).then(load)}>
                    Сброс
                  </button>
                  <button
                    className="btn"
                    onClick={() => api.updateClient(c.id, { ...c, enable: !c.enable }).then(load)}
                  >
                    {c.enable ? "Выкл" : "Вкл"}
                  </button>
                  <button
                    className="btn danger"
                    onClick={async () => {
                      if (!confirm("Удалить клиента?")) return;
                      await api.deleteClient(c.id);
                      load();
                    }}
                  >
                    Удалить
                  </button>
                </td>
              </tr>
            ))}
            {!ib.clients?.length && (
              <tr>
                <td colSpan={6} className="hint">
                  Клиентов нет.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {open && (
        <CreateClient
          inboundId={ib.id}
          onClose={() => setOpen(false)}
          onCreated={() => {
            setOpen(false);
            load();
          }}
        />
      )}
      {share && (
        <Modal title={share.name} onClose={() => setShare(null)}>
          <div className="qr">
            <QRCodeSVG value={share.link} size={196} />
          </div>
          <div className="share-box mono">{share.link}</div>
          <div className="hint">Подписка: {window.location.origin + share.sub}</div>
          <div className="modal-actions">
            <button
              className="btn"
              onClick={() => navigator.clipboard.writeText(share.link)}
            >
              Копировать ссылку
            </button>
            <button
              className="btn primary"
              onClick={() => navigator.clipboard.writeText(window.location.origin + share.sub)}
            >
              Копировать подписку
            </button>
          </div>
        </Modal>
      )}
    </>
  );
}

function CreateClient({
  inboundId,
  onClose,
  onCreated,
}: {
  inboundId: number;
  onClose: () => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState("");
  const [gb, setGb] = useState(0);
  const [expiry, setExpiry] = useState("");
  const [error, setError] = useState("");

  async function submit(e: FormEvent) {
    e.preventDefault();
    try {
      await api.createClient(inboundId, {
        name,
        totalGB: gb,
        expiryTime: expiry ? new Date(expiry).getTime() : 0,
      });
      onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "ошибка");
    }
  }

  return (
    <Modal title="Новый клиент" onClose={onClose}>
      <form onSubmit={submit}>
        <label>Имя</label>
        <input value={name} onChange={(e) => setName(e.target.value)} required placeholder="ivan" />
        <div className="form-grid">
          <div>
            <label>Лимит, ГБ (0 = без лимита)</label>
            <input type="number" min={0} step="0.1" value={gb} onChange={(e) => setGb(Number(e.target.value))} />
          </div>
          <div>
            <label>Срок</label>
            <input type="date" value={expiry} onChange={(e) => setExpiry(e.target.value)} />
          </div>
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
