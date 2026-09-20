import { FormEvent, useEffect, useState } from "react";
import { api } from "../api";

export default function Settings() {
  const [publicHost, setPublicHost] = useState("");
  const [publicPort, setPublicPort] = useState("");
  const [panelTitle, setPanelTitle] = useState("Kite");
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    api.settings().then((s) => {
      setPublicHost(s.publicHost);
      setPublicPort(s.publicPort);
      setPanelTitle(s.panelTitle);
    });
  }, []);

  async function save(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await api.saveSettings({ publicHost, publicPort, panelTitle });
      setMsg("Настройки сохранены");
    } catch (err) {
      setError(err instanceof Error ? err.message : "ошибка");
    }
  }

  async function password(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await api.changePassword(current, next);
      setCurrent("");
      setNext("");
      setMsg("Пароль изменён");
    } catch (err) {
      setError(err instanceof Error ? err.message : "ошибка");
    }
  }

  return (
    <>
      <div className="top">
        <div>
          <h1>Настройки</h1>
          <p>Публичный адрес попадает в vless-ссылки и подписку.</p>
        </div>
      </div>
      {error && <div className="error">{error}</div>}
      {msg && <div className="hint">{msg}</div>}
      <div className="grid two">
        <form className="card" onSubmit={save}>
          <h3>Ссылки</h3>
          <label>Публичный хост</label>
          <input value={publicHost} onChange={(e) => setPublicHost(e.target.value)} placeholder="vpn.example.com" />
          <label>Публичный порт (пусто = порт инбаунда)</label>
          <input value={publicPort} onChange={(e) => setPublicPort(e.target.value)} placeholder="443" />
          <label>Название панели</label>
          <input value={panelTitle} onChange={(e) => setPanelTitle(e.target.value)} />
          <button className="btn primary" style={{ marginTop: 16 }}>
            Сохранить
          </button>
        </form>
        <form className="card" onSubmit={password}>
          <h3>Пароль админа</h3>
          <label>Текущий</label>
          <input type="password" value={current} onChange={(e) => setCurrent(e.target.value)} />
          <label>Новый</label>
          <input type="password" value={next} onChange={(e) => setNext(e.target.value)} />
          <button className="btn primary" style={{ marginTop: 16 }}>
            Сменить пароль
          </button>
        </form>
      </div>
    </>
  );
}
