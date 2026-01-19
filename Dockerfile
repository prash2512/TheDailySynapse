FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /synapse ./cmd/synapse

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /synapse /synapse
COPY backend/scripts/ /app/scripts/

ENV DATABASE_URL=/app/data/synapse.db
ENV PORT=8080

VOLUME /app/data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -q --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/synapse"]
