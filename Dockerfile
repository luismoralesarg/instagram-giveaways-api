# syntax=docker/dockerfile:1

# Misma versión de Go que go.mod y que .github/workflows/release.yml.
FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -o /out/scheduler ./cmd/scheduler
RUN CGO_ENABLED=0 go build -o /out/seedadmin ./cmd/seedadmin
RUN CGO_ENABLED=0 go build -o /out/seedinstagramtoken ./cmd/seedinstagramtoken

# Misma versión que instala .github/workflows/release.yml para correr las migraciones en CI —
# si se cambia una, cambiar la otra.
FROM golang:1.25-alpine AS migrate-build
RUN CGO_ENABLED=0 go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /out/api /out/scheduler /out/seedadmin /out/seedinstagramtoken ./
COPY --from=migrate-build /go/bin/migrate ./
COPY internal/infrastructure/persistence/postgres/migrations ./migrations
COPY docker/entrypoint.sh ./entrypoint.sh
RUN chmod +x ./entrypoint.sh ./api ./scheduler ./seedadmin ./seedinstagramtoken

# El entrypoint aplica las migraciones pendientes y después ejecuta lo que venga en CMD (o lo
# que docker-compose.prod.yml pise vía `command:`/`entrypoint:` por servicio). Ver docker/entrypoint.sh.
ENTRYPOINT ["./entrypoint.sh"]
CMD ["./api"]
