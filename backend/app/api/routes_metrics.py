from typing import Annotated

from fastapi import APIRouter, Depends

from app.api.deps import get_current_user
from app.models.user import User
from app.schemas.metrics import SystemMetrics
from app.services.metrics import read_system_metrics

router = APIRouter()


@router.get("", response_model=SystemMetrics)
async def get_metrics(_: Annotated[User, Depends(get_current_user)]) -> SystemMetrics:
    """Return a snapshot of system metrics."""
    return read_system_metrics()
