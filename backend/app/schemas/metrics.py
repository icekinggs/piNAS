from pydantic import BaseModel


class DiskUsage(BaseModel):
    mountpoint: str
    device: str
    filesystem: str
    total_bytes: int
    used_bytes: int
    free_bytes: int
    percent: float


class NetworkCounters(BaseModel):
    bytes_sent: int
    bytes_recv: int
    packets_sent: int
    packets_recv: int


class SystemMetrics(BaseModel):
    cpu_percent: float
    ram_total_bytes: int
    ram_used_bytes: int
    ram_percent: float
    temperature_celsius: float | None
    uptime_seconds: int
    load_average: tuple[float, float, float]
    disks: list[DiskUsage]
    network: NetworkCounters
