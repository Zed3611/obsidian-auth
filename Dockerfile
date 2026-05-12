FROM golang:alpine3.23 AS builder

RUN apk update && apk add --no-cache git openssh-client

ENV CGO_ENABLED=0   
ENV GOOS=linux

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/auth ./cmd/auth
COPY pkg ./pkg

RUN go build -ldflags="-s -w" -o /out/auth ./cmd/auth

FROM ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.48 AS probe

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/auth /auth
COPY --from=probe /ko-app/grpc-health-probe /grpc_health_probe

USER nonroot:nonroot
ENTRYPOINT ["/auth"]