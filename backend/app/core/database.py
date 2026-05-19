from collections.abc import AsyncGenerator

from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine
from sqlalchemy.orm import DeclarativeBase
from sqlalchemy.pool import StaticPool

from app.core.config import get_settings


class Base(DeclarativeBase):
    """Base class for SQLAlchemy models."""


settings = get_settings()
engine_kwargs = {"echo": False, "future": True}
if settings.db_url.endswith(":memory:"):
    engine_kwargs["poolclass"] = StaticPool
    engine_kwargs["connect_args"] = {"check_same_thread": False}

engine = create_async_engine(settings.db_url, **engine_kwargs)
async_session_factory = async_sessionmaker(engine, expire_on_commit=False)


async def init_db() -> None:
    """Create database tables if they do not exist."""
    from app.models import user  # noqa: F401

    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)


async def get_db_session() -> AsyncGenerator[AsyncSession, None]:
    """Yield an async database session for request handlers."""
    async with async_session_factory() as session:
        yield session
