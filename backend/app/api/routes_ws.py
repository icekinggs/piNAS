import asyncio

from fastapi import APIRouter, WebSocket, WebSocketDisconnect

from app.core.security import decode_token
from app.services.metrics import read_system_metrics

router = APIRouter()


@router.websocket("/metrics")
async def metrics_websocket(websocket: WebSocket) -> None:
    token = websocket.query_params.get("token")
    if not token:
        await websocket.close(code=1008)
        return

    try:
        decode_token(token, expected_type="access")
    except Exception:
        await websocket.close(code=1008)
        return

    await websocket.accept()
    try:
        while True:
            metrics = read_system_metrics()
            await websocket.send_json(metrics.model_dump())
            await asyncio.sleep(2)
    except WebSocketDisconnect:
        return
