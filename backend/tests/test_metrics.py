from app.services.metrics import read_system_metrics


def test_read_system_metrics_shape():
    metrics = read_system_metrics()

    assert metrics.ram_total_bytes > 0
    assert metrics.uptime_seconds >= 0
    assert metrics.cpu_percent >= 0
    assert len(metrics.load_average) == 3
