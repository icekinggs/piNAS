# Contributing

PiNAS is designed to stay private, offline-first, and easy to run on a Raspberry Pi.

## Development checks

Backend:

```bash
cd backend
go mod tidy
go test ./...
```

Frontend:

```bash
cd frontend
npm ci
npm run check
npm run build
```

## Pull request expectations

- Keep the install path simple: `sudo ./bootstrap.sh` should remain the main path.
- Do not add telemetry, cloud dependencies, or mandatory external services.
- Keep runtime resource usage reasonable for Raspberry Pi 4+.
- Add or update tests for auth, filesystem, install, and security-sensitive changes.
- Update docs when endpoints, environment variables, or install behavior changes.
