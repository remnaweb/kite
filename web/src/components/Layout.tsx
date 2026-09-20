import { NavLink } from "react-router-dom";
import { ReactNode, useEffect, useState } from "react";
import { api, SystemStatus } from "../api";
import Logo from "./Logo";
import ThemeToggle from "./ThemeToggle";

export default function Layout({
  children,
  user,
  onLogout,
}: {
  children: ReactNode;
  user: string;
  onLogout: () => void;
}) {
  const [sys, setSys] = useState<SystemStatus | null>(null);
  useEffect(() => {
    let alive = true;
    const load = () =>
      api
        .system()
        .then((s) => alive && setSys(s))
        .catch(() => {});
    load();
    const t = setInterval(load, 8000);
    return () => {
      alive = false;
      clearInterval(t);
    };
  }, []);

  return (
    <div className="shell">
      <header className="topbar">
        <NavLink to="/" className="logo">
          <Logo />
          Kite
        </NavLink>
        <nav>
          <NavLink to="/" end className={({ isActive }) => "nav-link" + (isActive ? " active" : "")}>
            Обзор
          </NavLink>
          <NavLink to="/inbounds" className={({ isActive }) => "nav-link" + (isActive ? " active" : "")}>
            Инбаунды
          </NavLink>
          <NavLink to="/settings" className={({ isActive }) => "nav-link" + (isActive ? " active" : "")}>
            Настройки
          </NavLink>
        </nav>
        <div className="top-meta">
          <span>
            <span className={"dot " + (sys?.xray.running ? "on" : "")} />
            {sys?.xray.running ? "Xray живой" : sys?.xray.binaryExists ? "Xray стоит" : "нет Xray"}
          </span>
          <span>{user}</span>
          <ThemeToggle />
          <button className="btn ghost" onClick={onLogout}>
            Выйти
          </button>
        </div>
      </header>
      <main className="page">{children}</main>
    </div>
  );
}
