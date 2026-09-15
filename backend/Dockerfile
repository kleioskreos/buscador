# ============================================================
# BTR Backend — Go + MySQL driver
# Multi-stage: compila en golang:alpine, corre en alpine minimo
# ============================================================

# ---------- Stage 1: Build ----------
FROM golang:1.24-alpine AS builder

# Dependencias de compilacion (cgo no es necesario con el driver puro de Go)
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache de dependencias: copiar go.mod/go.sum primero
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || go mod tidy

# Codigo fuente
COPY *.go ./

# Compilacion estatica
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/btr-backend .

# ---------- Stage 2: Runtime ----------
FROM alpine:3.21

# python3 + pyxlsb: el backend convierte .xlsb a TSV al importar datos.
RUN apk add --no-cache ca-certificates tzdata wget python3 py3-pip \
    && python3 -m pip install --break-system-packages pyxlsb \
    && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /app/btr-backend /app/btr-backend
COPY xlsb_to_tsv.py /app/xlsb_to_tsv.py

RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=5 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/btr-backend"]