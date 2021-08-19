# syntax = docker/dockerfile-upstream:master-labs
#-*-mode:dockerfile;indent-tabs-mode:nil;tab-width:2;coding:utf-8-*-
# vi: ft=dockerfile tabstop=2 shiftwidth=2 softtabstop=2 expandtab:
# ─── PREBUILD IMAGE ALIASES ─────────────────────────────────────────────────────
# hadolint ignore=DL3007
FROM fjolsvin/hashicorp:latest as hashicorp
# hadolint ignore=DL3007
FROM fjolsvin/jen:latest as jen
# hadolint ignore=DL3007
FROM fjolsvin/tojson:latest as tojson
# hadolint ignore=DL3007
FROM fjolsvin/yq:latest as yq
# hadolint ignore=DL3007
FROM fjolsvin/petname:latest as petname
# hadolint ignore=DL3007
FROM fjolsvin/hyperfine:latest as hyperfine
# hadolint ignore=DL3007
FROM fjolsvin/golang-alpine-sandbox:latest
USER root
ARG IMAGE_SPECIFIC_PACKAGES="\
  protobuf=3.15.7-r1 \
  py3-protobuf=3.15.7-r0 \
  moreutils=0.65-r1 \
  parallel=20210722-r0 \
  "
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
# hadolint ignore=DL3018,SC2086
RUN \
  IFS=' ' read -a packages <<< $IMAGE_SPECIFIC_PACKAGES ; \
  apk add --update --no-cache "${packages[@]}" \
  || ( \
  sed -i -e 's/dl-cdn/dl-4/g' /etc/apk/repositories \
  && apk add --update --no-cache "${packages[@]}" \
  ) \
  && touch "/var/run/docker.sock" \
  && chown "$(id -u ${USER}):docker" "/var/run/docker.sock" \
  && chown  "$(id -u ${USER}):$(id -g ${USER})" "${HOME}" -R \
  && chown  "$(id -u ${USER}):$(id -g ${USER})" "${WORKDIR}" -R
COPY --from=yq /workspace/yq /usr/local/bin/yq
COPY --from=hashicorp /usr/local/bin/vault /usr/local/bin/vault
COPY --from=hashicorp /usr/local/bin/consul /usr/local/bin/consul
COPY --from=jen /workspace/jen /usr/local/bin
COPY --from=tojson /workspace/tojson /usr/local/bin/tojson
COPY --from=petname /workspace/petname /usr/local/bin/petname
COPY --from=hyperfine /workspace/hyperfine /usr/local/bin/hyperfine
USER "${USER}"
WORKDIR "${WORKDIR}"
ENTRYPOINT [ "" ]
