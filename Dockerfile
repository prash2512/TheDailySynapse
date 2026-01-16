FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /synapse ./cmd/synapse

FROM alpine:latest

COPY --from=builder /synapse /synapse

EXPOSE 8080

ENTRYPOINT ["/synapse"]