# PiNAS backend — multi-stage build, otimizado para ARM64.
# CGO necessário por causa do go-sqlite3.

FROM golang:1.23-alpine AS builder

# Toolchain para CGO em Alpine.
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /src

# Copia TUDO primeiro. Sem isso, `go mod tidy` não vê os imports do código
# e gera um go.sum incompleto/vazio, fazendo o build seguinte falhar.
COPY . .

# Resolve dependências. go mod tidy lê o código, baixa o que falta e
# (re)gera o go.sum. Funciona tanto na 1ª vez (sem go.sum) quanto em rebuilds.
RUN go mod tidy

# Build estático com CGO. -trimpath remove paths absolutos do binário (reproducibility).
# -s -w removem símbolos de debug para reduzir tamanho.
ENV CGO_ENABLED=1
ENV GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/pinas ./cmd/pinas

# ---- runtime ----
FROM alpine:3.20

RUN apk add --no-cache \
        ca-certificates \
        sqlite-libs \
        tzdata \
        wget \
        && addgroup -S pinas && adduser -S -G pinas pinas

# Diretórios persistentes (montados como volumes).
RUN mkdir -p /var/lib/pinas/db \
             /var/lib/pinas/data \
             /var/lib/pinas/thumbs \
             /var/lib/pinas/logs \
             /var/lib/pinas/secrets \
    && chown -R pinas:pinas /var/lib/pinas

COPY --from=builder /out/pinas /usr/local/bin/pinas
COPY migrations /usr/local/share/pinas/migrations

USER pinas
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/pinas"]
