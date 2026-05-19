from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import async_session_factory
from app.core.security import hash_password, verify_password
from app.models.user import User


async def authenticate_user(session: AsyncSession, username: str, password: str) -> User | None:
    """Return the active user when credentials are valid."""
    result = await session.execute(select(User).where(User.username == username))
    user = result.scalar_one_or_none()
    if user is None or not user.is_active:
        return None
    if not verify_password(password, user.password_hash):
        return None
    return user


async def ensure_initial_admin(username: str, password: str) -> None:
    """Create the initial administrator if no user exists yet."""
    async with async_session_factory() as session:
        result = await session.execute(select(User).limit(1))
        existing_user = result.scalar_one_or_none()
        if existing_user is not None:
            return

        session.add(
            User(
                username=username,
                password_hash=hash_password(password),
                is_admin=True,
                is_active=True,
            )
        )
        await session.commit()
