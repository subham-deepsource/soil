# syntax = docker/dockerfile-upstream:master-labs
#-*-mode:dockerfile;indent-tabs-mode:nil;tab-width:2;coding:utf-8-*-
# vi: ft=dockerfile tabstop=2 shiftwidth=2 softtabstop=2 expandtab:

# ─── PREBUILD IMAGE ALIASES ─────────────────────────────────────────────────────
# hadolint ignore=DL3007
FROM fjolsvin/upx:latest as upx
# ─── BUILD STAGE ────────────────────────────────────────────────────────────────
FROM golang:alpine AS build-stage
SHELL ["/bin/ash", "-o", "pipefail", "-c"]
# hadolint ignore=DL3018
RUN \
  set -ex && \
  apk add --no-cache build-base make git bash ncurses
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
# ─── INSTALL MAGE ───────────────────────────────────────────────────────────────
RUN \
  set -ex && \
  git clone https://github.com/magefile/mage /tmp/mage
WORKDIR /tmp/mage
RUN \
  --mount=type=cache,target="/root/.cache/go-build" \
  --mount=type=cache,target="/go/pkg/mod" \
  --mount=type=tmpfs,target="/go/src/" \
  set -xeu ; \
  go run bootstrap.go
# ─── BUILD ──────────────────────────────────────────────────────────────────────
WORKDIR /workspace
COPY . .
# hadolint ignore=SC2239
RUN \
  --mount=type=cache,target="/root/.cache/go-build" \
  --mount=type=cache,target="/go/pkg/mod" \
  --mount=type=tmpfs,target="/go/src/" \
  set -xeu ; \
  mage -v build \
  && echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd


# ─── COMPRESSION STAGE ──────────────────────────────────────────────────────────
FROM upx AS compression-stage
COPY --from=build-stage "/workspace/bin" "/workspace"
RUN \
  compress ;
# ─── MAIN LAYER ─────────────────────────────────────────────────────────────────
FROM alpine:3.14
SHELL ["/bin/ash", "-o", "pipefail", "-c"]
# hadolint ignore=DL3018
RUN apk add --no-cache curl
COPY --from=build-stage /etc_passwd /etc/passwd
USER nobody
WORKDIR /workspace
COPY --from=build-stage /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=compression-stage --chown=65534:0 "/workspace/soil" /entrypoint
ENTRYPOINT [ "/entrypoint" ]
CMD [ "help" ]
