from app.core.database import init_db
from app.services.auth import ensure_initial_admin


async def test_login_and_me(client):
    await init_db()
    await ensure_initial_admin("admin", "admin12345")

    login_response = await client.post(
        "/api/v1/auth/login",
        json={"username": "admin", "password": "admin12345"},
    )

    assert login_response.status_code == 200
    tokens = login_response.json()
    assert tokens["access_token"]
    assert tokens["refresh_token"]

    me_response = await client.get(
        "/api/v1/auth/me",
        headers={"Authorization": f"Bearer {tokens['access_token']}"},
    )

    assert me_response.status_code == 200
    assert me_response.json()["username"] == "admin"


async def test_rejects_invalid_login(client):
    await init_db()
    await ensure_initial_admin("admin", "admin12345")

    response = await client.post(
        "/api/v1/auth/login",
        json={"username": "admin", "password": "senha-errada"},
    )

    assert response.status_code == 401
