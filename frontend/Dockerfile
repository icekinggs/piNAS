# PiNAS frontend — build estático, servido pelo Caddy.
# Usar como ferramenta CI/CD opcional. Em dev local, basta `npm run build`.

FROM node:22-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci || npm install

COPY . .
RUN npm run build

# Stage final só com o build/ — útil para `docker cp` para o volume.
FROM alpine:3.20
COPY --from=builder /app/build /srv/frontend
CMD ["sh", "-c", "echo 'frontend build pronto em /srv/frontend' && tail -f /dev/null"]
