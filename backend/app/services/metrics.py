import os
import time

import psutil

from app.schemas.metrics import DiskUsage, NetworkCounters, SystemMetrics


def _read_pi_temperature() -> float | None:
    thermal_path = "/sys/class/thermal/thermal_zone0/temp"
    try:
        with open(thermal_path, encoding="utf-8") as temp_file:
            raw_value = temp_file.read().strip()
    except OSError:
        return None

    try:
        return round(int(raw_value) / 1000, 1)
    except ValueError:
        return None


def _read_disks() -> list[DiskUsage]:
    disks: list[DiskUsage] = []
    for partition in psutil.disk_partitions(all=False):
        try:
            usage = psutil.disk_usage(partition.mountpoint)
        except OSError:
            continue
        disks.append(
            DiskUsage(
                mountpoint=partition.mountpoint,
                device=partition.device,
                filesystem=partition.fstype,
                total_bytes=usage.total,
                used_bytes=usage.used,
                free_bytes=usage.free,
                percent=usage.percent,
            )
        )
    return disks


def read_system_metrics() -> SystemMetrics:
    """Read CPU, memory, temperature, disk and network metrics."""
    memory = psutil.virtual_memory()
    network = psutil.net_io_counters()
    boot_time = psutil.boot_time()

    return SystemMetrics(
        cpu_percent=psutil.cpu_percent(interval=None),
        ram_total_bytes=memory.total,
        ram_used_bytes=memory.used,
        ram_percent=memory.percent,
        temperature_celsius=_read_pi_temperature(),
        uptime_seconds=max(0, int(time.time() - boot_time)),
        load_average=os.getloadavg() if hasattr(os, "getloadavg") else (0.0, 0.0, 0.0),
        disks=_read_disks(),
        network=NetworkCounters(
            bytes_sent=network.bytes_sent,
            bytes_recv=network.bytes_recv,
            packets_sent=network.packets_sent,
            packets_recv=network.packets_recv,
        ),
    )
