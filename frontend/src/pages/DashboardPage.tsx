import { useQuery } from "@tanstack/react-query";
import { Activity, Cpu, HardDrive, LogOut, MemoryStick, Thermometer } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

import { fetchMetrics, metricsWebSocketUrl, type SystemMetrics } from "../services/api";
import { clearTokens } from "../services/session";

type CpuPoint = { time: string; cpu: number };

function formatBytes(value: number) {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = value;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${size.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

function formatUptime(seconds: number) {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return `${days}d ${hours}h ${minutes}min`;
}

export function DashboardPage() {
  const [liveMetrics, setLiveMetrics] = useState<SystemMetrics | null>(null);
  const [cpuHistory, setCpuHistory] = useState<CpuPoint[]>([]);
  const metricsQuery = useQuery({ queryKey: ["metrics"], queryFn: fetchMetrics });
  const metrics = liveMetrics ?? metricsQuery.data;

  useEffect(() => {
    const ws = new WebSocket(metricsWebSocketUrl());
    ws.onmessage = (event) => {
      const nextMetrics = JSON.parse(event.data) as SystemMetrics;
      setLiveMetrics(nextMetrics);
      setCpuHistory((history) => [
        ...history.slice(-29),
        { time: new Date().toLocaleTimeString("pt-BR"), cpu: nextMetrics.cpu_percent }
      ]);
    };
    return () => ws.close();
  }, []);

  function logout() {
    clearTokens();
    location.href = "/login";
  }

  return (
    <main className="mx-auto max-w-7xl space-y-6 px-4 py-6">
      <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
        <div>
          <h1 className="text-2xl font-semibold">Dashboard</h1>
          <p className="text-sm text-muted-foreground">Métricas do Raspberry Pi em tempo real.</p>
        </div>
        <button className="button-secondary" type="button" onClick={logout}>
          <LogOut className="h-4 w-4" />
          Sair
        </button>
      </div>

      {metricsQuery.isLoading && !metrics ? <p>Carregando métricas...</p> : null}

      {metrics ? (
        <>
          <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <MetricCard
              icon={<Cpu className="h-5 w-5" />}
              label="CPU"
              value={`${metrics.cpu_percent.toFixed(1)}%`}
              detail={`Load ${metrics.load_average.map((value) => value.toFixed(2)).join(" / ")}`}
            />
            <MetricCard
              icon={<MemoryStick className="h-5 w-5" />}
              label="Memória"
              value={`${metrics.ram_percent.toFixed(1)}%`}
              detail={`${formatBytes(metrics.ram_used_bytes)} de ${formatBytes(metrics.ram_total_bytes)}`}
            />
            <MetricCard
              icon={<Thermometer className="h-5 w-5" />}
              label="Temperatura"
              value={metrics.temperature_celsius ? `${metrics.temperature_celsius.toFixed(1)} °C` : "N/D"}
              detail="Sensor térmico do sistema"
            />
            <MetricCard
              icon={<Activity className="h-5 w-5" />}
              label="Uptime"
              value={formatUptime(metrics.uptime_seconds)}
              detail={`Rede: ${formatBytes(metrics.network.bytes_recv)} recebidos`}
            />
          </section>

          <section className="grid gap-6 lg:grid-cols-[1.3fr_1fr]">
            <div className="panel">
              <h2 className="panel-title">Uso de CPU</h2>
              <div className="h-72">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={cpuHistory}>
                    <XAxis dataKey="time" hide />
                    <YAxis domain={[0, 100]} tickLine={false} axisLine={false} />
                    <Tooltip />
                    <Area type="monotone" dataKey="cpu" stroke="#0f766e" fill="#99f6e4" />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </div>

            <div className="panel">
              <h2 className="panel-title">Volumes montados</h2>
              <div className="space-y-4">
                {metrics.disks.map((disk) => (
                  <div key={`${disk.device}-${disk.mountpoint}`} className="space-y-2">
                    <div className="flex items-center justify-between gap-3 text-sm">
                      <span className="flex min-w-0 items-center gap-2 font-medium">
                        <HardDrive className="h-4 w-4 shrink-0" />
                        <span className="truncate">{disk.mountpoint}</span>
                      </span>
                      <span>{disk.percent.toFixed(1)}%</span>
                    </div>
                    <div className="h-2 rounded bg-muted">
                      <div
                        className="h-2 rounded bg-primary"
                        style={{ width: `${Math.min(disk.percent, 100)}%` }}
                      />
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {disk.device} · {disk.filesystem} · {formatBytes(disk.used_bytes)} de{" "}
                      {formatBytes(disk.total_bytes)}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          </section>
        </>
      ) : null}
    </main>
  );
}

function MetricCard({
  icon,
  label,
  value,
  detail
}: {
  icon: ReactNode;
  label: string;
  value: string;
  detail: string;
}) {
  return (
    <div className="panel">
      <div className="mb-3 flex items-center justify-between text-muted-foreground">
        <span className="text-sm font-medium">{label}</span>
        {icon}
      </div>
      <div className="text-2xl font-semibold">{value}</div>
      <p className="mt-1 text-sm text-muted-foreground">{detail}</p>
    </div>
  );
}
