import { FormEvent, useState } from "react";
import { api } from "../api";
import Logo from "../components/Logo";
import ThemeToggle from "../components/ThemeToggle";

export default function Login({ onLogin }: { onLogin: (user: string) => void }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api.login(username, password);
      onLogin(username);
    } catch (err) {
      setError(err instanceof Error ? err.message : "ошибка входа");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="login-wrap">
      <section className="login-hero">
        <div className="login-hero-top">
          <span className="logo">
            <Logo size={28} />
            Kite
          </span>
          <ThemeToggle />
        </div>
        <div>
          <h1>
            Свой
            <br />
            Xray.
            <br />
            Своя
            <br />
            панель.
          </h1>
        </div>
        <p>Kite крутит ядро на этой машине. Клиенты, ключи, трафик — без чужих панелей.</p>
      </section>
      <form className="login-card" onSubmit={submit}>
        <h2>Вход</h2>
        <p className="lead">Логин, который задал при установке.</p>
        <label>Логин</label>
        <input value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" autoFocus />
        <label>Пароль</label>
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
        />
        {error && <div className="error">{error}</div>}
        <button className="btn primary" style={{ width: "100%", marginTop: 22 }} disabled={busy}>
          {busy ? "вхожу…" : "Войти"}
        </button>
      </form>
    </div>
  );
}
