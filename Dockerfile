FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o aegis ./cmd/api

FROM alpine:3.22
RUN apk --no-cache add ca-certificates
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /build/aegis .
USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/aegis"]
