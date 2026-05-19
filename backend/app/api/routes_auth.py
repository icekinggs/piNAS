from collections import defaultdict, deque
from time import monotonic
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, Request, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.api.deps import get_current_user
from app.core.database import get_db_session
from app.core.security import create_token, decode_token
from app.models.user import User
from app.schemas.auth import LoginRequest, RefreshRequest, TokenPair, UserRead
from app.services.auth import authenticate_user

router = APIRouter()

LOGIN_WINDOW_SECONDS = 60
LOGIN_MAX_ATTEMPTS = 8
_login_attempts: dict[str, deque[float]] = defaultdict(deque)


def _check_login_rate_limit(client_id: str) -> None:
    now = monotonic()
    attempts = _login_attempts[client_id]
    while attempts and now - attempts[0] > LOGIN_WINDOW_SECONDS:
        attempts.popleft()
    if len(attempts) >= LOGIN_MAX_ATTEMPTS:
        raise HTTPException(
            status_code=status.HTTP_429_TOO_MANY_REQUESTS,
            detail="Muitas tentativas de login. Aguarde um minuto.",
        )
    attempts.append(now)


@router.post("/login", response_model=TokenPair)
async def login(
    payload: LoginRequest,
    request: Request,
    session: Annotated[AsyncSession, Depends(get_db_session)],
) -> TokenPair:
    client_id = request.client.host if request.client else "unknown"
    _check_login_rate_limit(client_id)

    user = await authenticate_user(session, payload.username, payload.password)
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Credenciais inválidas.",
        )

    return TokenPair(
        access_token=create_token(user.username, "access"),
        refresh_token=create_token(user.username, "refresh"),
    )


@router.post("/refresh", response_model=TokenPair)
async def refresh_token(payload: RefreshRequest) -> TokenPair:
    token_payload = decode_token(payload.refresh_token, expected_type="refresh")
    username = token_payload["sub"]
    return TokenPair(
        access_token=create_token(username, "access"),
        refresh_token=create_token(username, "refresh"),
    )


@router.get("/me", response_model=UserRead)
async def read_current_user(
    current_user: Annotated[User, Depends(get_current_user)],
) -> UserRead:
    return UserRead(
        id=current_user.id,
        username=current_user.username,
        is_admin=current_user.is_admin,
        is_active=current_user.is_active,
    )
