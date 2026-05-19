import { Moon, Server, Sun } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { Navigate, Route, Routes } from "react-router-dom";

import { DashboardPage } from "./pages/DashboardPage";
import { LoginPage } from "./pages/LoginPage";
import { getAccessToken } from "./services/session";

export default function App() {
  const [dark, setDark] = useState(() => localStorage.getItem("theme") === "dark");

  useEffect(() => {
    document.documentElement.classList.toggle("dark", dark);
    localStorage.setItem("theme", dark ? "dark" : "light");
  }, [dark]);

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="border-b border-border">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3">
          <div className="flex items-center gap-2 font-semibold">
            <Server className="h-5 w-5 text-primary" />
            <span>Meu NAS</span>
          </div>
          <button
            className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border hover:bg-muted"
            type="button"
            aria-label="Alternar tema"
            onClick={() => setDark((value) => !value)}
          >
            {dark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </button>
        </div>
      </header>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/"
          element={
            <RequireAuth>
              <DashboardPage />
            </RequireAuth>
          }
        />
      </Routes>
    </div>
  );
}

function RequireAuth({ children }: { children: ReactNode }) {
  return getAccessToken() ? children : <Navigate replace to="/login" />;
}
