# Meu NAS

Sistema NAS leve para Raspberry Pi 4/5 baseado em Raspberry Pi OS Lite 64-bit.

Este repositório começa pela Fase 1: API FastAPI, autenticação JWT, dashboard com métricas em tempo real e frontend React em português brasileiro.

## Estrutura

```text
backend/   API FastAPI, SQLite, autenticação e métricas
frontend/  Interface React + Vite + TypeScript + TailwindCSS
data/      Banco SQLite e arquivos locais de configuração
scripts/   Instalação, systemd e nginx
```

## Rodando o backend em desenvolvimento

```bash
cd backend
python3.11 -m venv .venv
. .venv/bin/activate
pip install -e ".[dev]"
cp .env.example .env
uvicorn main:app --reload --host 0.0.0.0 --port 8000
```

O usuário administrador inicial é criado automaticamente no primeiro boot usando:

- `MEUNAS_ADMIN_USERNAME`
- `MEUNAS_ADMIN_PASSWORD`

Troque a senha padrão antes de expor a interface na rede.

## Rodando o frontend em desenvolvimento

```bash
cd frontend
npm install
npm run dev -- --host 0.0.0.0
```

## Testes

```bash
cd backend
pytest
```

## Instalação no Raspberry Pi

Os arquivos em `scripts/` mostram uma instalação base em `/opt/meu-nas`, com serviço systemd para o backend e nginx servindo o frontend compilado.

## Imagem de sistema operacional

Para entregar como produto instalável, use a camada `os-image/`. Ela descreve uma imagem appliance baseada em Ubuntu Server ARM64 para Raspberry Pi:

1. grava a imagem no cartão SD ou SSD;
2. faz o primeiro boot;
3. expande o filesystem;
4. cria usuário de sistema dedicado;
5. instala backend, frontend, systemd e nginx;
6. sobe o painel web automaticamente.

O objetivo não é manter um fork completo do Ubuntu. O objetivo é gerar uma imagem reproducível, pequena e atualizável, com o Meu NAS pré-instalado.

Leia `scripts/README_FASE_1.md` antes de executar em um Raspberry Pi real.
