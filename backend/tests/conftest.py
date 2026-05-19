import os
from collections.abc import AsyncGenerator

import pytest
from httpx import ASGITransport, AsyncClient

os.environ["MEUNAS_DB_URL"] = "sqlite+aiosqlite:///:memory:"
os.environ["MEUNAS_SECRET_KEY"] = "test-secret-key-with-at-least-thirty-two-chars"
os.environ["MEUNAS_ADMIN_USERNAME"] = "admin"
os.environ["MEUNAS_ADMIN_PASSWORD"] = "admin12345"

from main import app  # noqa: E402


@pytest.fixture
async def client() -> AsyncGenerator[AsyncClient, None]:
    async with AsyncClient(
        transport=ASGITransport(app=app),
        base_url="http://testserver",
    ) as test_client:
        yield test_client
