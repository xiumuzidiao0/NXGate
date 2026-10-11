# Stage 1: Build the binary
FROM golang:alpine AS builder

WORKDIR /src
COPY . .
ENV GOTOOLCHAIN=auto
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/nxgate ./cmd/nxgate

# Stage 2: Minimal runtime image (<25MB)
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /bin/nxgate /app/nxgate

ENV DATA_DIR=/app/data \
    UI_HOST=0.0.0.0 \
    UI_PORT=8787 \
    LOCAL_PROXY_HOST=0.0.0.0 \
    LOCAL_PROXY_PORT=7928

EXPOSE 8787 7928

ENTRYPOINT ["/app/nxgate"]
