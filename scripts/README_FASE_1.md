# Fase 1 - Fundação

## Decisões técnicas

- FastAPI com lifespan inicializa o SQLite e cria o administrador inicial.
- SQLite usa SQLAlchemy 2.0 async com `aiosqlite`.
- JWT tem access token curto e refresh token separado.
- bcrypt é usado diretamente para evitar dependências extras de autenticação.
- Métricas usam `psutil`; temperatura do Raspberry Pi vem de `/sys/class/thermal/thermal_zone0/temp`.
- WebSocket exige access token na query string e envia métricas a cada 2 segundos.
- O serviço systemd deve rodar como usuário dedicado, não como root.

## Armadilhas comuns

- Não exponha a senha padrão `admin12345`; defina `MEUNAS_ADMIN_PASSWORD` antes do primeiro boot.
- Não rode o backend inteiro como root. Para fases futuras, use sudoers por comando.
- Em Raspberry Pi OS Lite, instale `python3.11-venv`, `build-essential` e headers se o bcrypt precisar compilar.
- O nginx deve servir apenas o frontend estático e encaminhar `/api/` para Uvicorn.
- WebSocket atrás de nginx precisa dos headers `Upgrade` e `Connection`.

## Validações rápidas

```bash
curl http://127.0.0.1:8000/docs
curl -X POST http://127.0.0.1:8000/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin12345"}'
```

Depois use o access token:

```bash
curl http://127.0.0.1:8000/api/v1/metrics \
  -H "Authorization: Bearer TOKEN_AQUI"
```
