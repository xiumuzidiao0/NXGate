FROM golang:1.25.13-alpine@sha256:1e0126852075c9c60731c8ba49088448b91f63e2aed97ca9d1a9791622a05946 AS builder

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/nxgate ./cmd/nxgate

FROM alpine:3.20@sha256:d9e853e87e55526f6b2917df91a2115c36dd7c696a35be12163d44e6e2a4b6bc

RUN apk add --no-cache openvpn ca-certificates tzdata iptables

WORKDIR /app
COPY --from=builder /bin/nxgate /app/nxgate

ENV DATA_DIR=/app/data \
    UI_HOST=:: \
    UI_PORT=8787 \
    LOCAL_PROXY_HOST=:: \
    LOCAL_PROXY_PORT=7928

EXPOSE 8787 7928

ENTRYPOINT ["/app/nxgate"]
