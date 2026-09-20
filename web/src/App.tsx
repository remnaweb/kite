import { Navigate, Route, Routes } from "react-router-dom";
import { useEffect, useState } from "react";
import { api } from "./api";
import Layout from "./components/Layout";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import Inbounds from "./pages/Inbounds";
import InboundDetail from "./pages/InboundDetail";
import Settings from "./pages/Settings";

export default function App() {
  const [user, setUser] = useState<string | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    api
      .me()
      .then((m) => setUser(m.username))
      .catch(() => setUser(null))
      .finally(() => setReady(true));
  }, []);

  if (!ready) return <div className="login-wrap">загрузка…</div>;
  if (!user) return <Login onLogin={setUser} />;

  return (
    <Layout
      user={user}
      onLogout={async () => {
        await api.logout();
        setUser(null);
      }}
    >
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/inbounds" element={<Inbounds />} />
        <Route path="/inbounds/:id" element={<InboundDetail />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Layout>
  );
}
