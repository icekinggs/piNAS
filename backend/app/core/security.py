from datetime import UTC, datetime, timedelta
from typing import Any, Literal

import bcrypt
import jwt
from fastapi import HTTPException, status

from app.core.config import get_settings

TokenType = Literal["access", "refresh"]


def hash_password(password: str) -> str:
    """Hash a plaintext password with bcrypt."""
    return bcrypt.hashpw(password.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")


def verify_password(password: str, password_hash: str) -> bool:
    """Return whether a plaintext password matches a bcrypt hash."""
    return bcrypt.checkpw(password.encode("utf-8"), password_hash.encode("utf-8"))


def create_token(subject: str, token_type: TokenType) -> str:
    """Create a signed JWT for the given subject."""
    settings = get_settings()
    now = datetime.now(UTC)
    if token_type == "access":
        expires_at = now + timedelta(minutes=settings.access_token_minutes)
    else:
        expires_at = now + timedelta(days=settings.refresh_token_days)

    payload: dict[str, Any] = {
        "sub": subject,
        "type": token_type,
        "iat": int(now.timestamp()),
        "exp": int(expires_at.timestamp()),
    }
    return jwt.encode(payload, settings.secret_key, algorithm="HS256")


def decode_token(token: str, expected_type: TokenType) -> dict[str, Any]:
    """Decode and validate a JWT."""
    settings = get_settings()
    try:
        payload = jwt.decode(token, settings.secret_key, algorithms=["HS256"])
    except jwt.PyJWTError as exc:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token inválido ou expirado.",
        ) from exc

    if payload.get("type") != expected_type or not payload.get("sub"):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token inválido para esta operação.",
        )
    return payload
