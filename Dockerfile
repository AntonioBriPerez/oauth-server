# --- ETAPA 1: BUILD ---
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copiamos dependencias primero (caché de Docker)
COPY go.mod go.sum ./
RUN go mod download

# Copiamos el código
COPY . .

# Compilamos el binario estático
# CGO_ENABLED=0 es vital para Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o oauth-server ./cmd/api/main.go

# --- ETAPA 2: RUNTIME ---
FROM alpine:latest

WORKDIR /root/

# Copiamos solo el binario desde la etapa anterior
COPY --from=builder /app/oauth-server .

EXPOSE 8080

CMD ["./oauth-server"]