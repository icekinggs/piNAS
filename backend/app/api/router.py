from fastapi import APIRouter

from app.api.routes_auth import router as auth_router
from app.api.routes_metrics import router as metrics_router
from app.api.routes_ws import router as ws_router

api_router = APIRouter()
api_router.include_router(auth_router, prefix="/auth", tags=["auth"])
api_router.include_router(metrics_router, prefix="/metrics", tags=["metrics"])
api_router.include_router(ws_router, prefix="/ws", tags=["websocket"])
