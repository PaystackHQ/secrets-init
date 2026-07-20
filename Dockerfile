# syntax=docker/dockerfile:1@sha256:87999aa3d42bdc6bea60565083ee17e86d1f3339802f543c0d03998580f9cb89

FROM --platform=${BUILDPLATFORM} golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=v0
ARG BUILD_DATE=unknown
ARG COMMIT=unknown
ARG BRANCH=unknown

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GO=go TARGETOS=${TARGETOS} TARGETARCH=${TARGETARCH} \
    VERSION=${VERSION} DATE=${BUILD_DATE} COMMIT=${COMMIT} BRANCH=${BRANCH} \
    OUTPUT=/out/secrets-init ./scripts/build.sh

FROM busybox:1.38.0@sha256:fd8d9aa63ba2f0982b5304e1ee8d3b90a210bc1ffb5314d980eb6962f1a9715d
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/secrets-init /secrets-init
RUN adduser -D -u 1000 secrets-init
USER 1000

ENTRYPOINT ["/secrets-init"]
CMD ["--version"]
