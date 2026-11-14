FROM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS compilacao

ENV CGO_ENABLED=0 GOTOOLCHAIN=local

WORKDIR /fonte
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN go build -trimpath -ldflags="-s -w" -o /saida/aprova-sandbox ./cmd/sandbox

FROM maven:3.9.16-eclipse-temurin-21@sha256:99e61abcff91a9b1333463bd8451fb18495d6eba9250ac66a338b518f8278320

ARG VERSOES_DO_SUREFIRE="3.2.5 3.5.2 3.5.4 3.6.1"
ARG VERSOES_DO_JUNIT5="5.10.2 5.11.4 5.14.4 6.1.3"
ARG VERSOES_DO_JUNIT4="4.13.2"

RUN apt-get update \
    && apt-get install -y --no-install-recommends git procps \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 aprova \
    && useradd --uid 10001 --gid aprova --create-home --shell /usr/sbin/nologin aprova

COPY imagens/java/aquecimento/ /tmp/aquecimento/

RUN set -eu; \
    aquecer() { \
        mvn -B -ntp -q -Dmaven.repo.local=/opt/aprova-cache/m2 \
            -Djunit.version="$2" -Dsurefire.version="$3" -f "/tmp/aquecimento/$1/pom.xml" clean test; \
    }; \
    for surefire in $VERSOES_DO_SUREFIRE; do \
        for junit in $VERSOES_DO_JUNIT5; do aquecer junit5 "$junit" "$surefire"; done; \
        for junit in $VERSOES_DO_JUNIT4; do aquecer junit4 "$junit" "$surefire"; done; \
    done; \
    rm -rf /tmp/aquecimento /root/.m2; \
    chmod -R a+rX /opt/aprova-cache

COPY --from=compilacao /saida/aprova-sandbox /usr/local/bin/aprova-sandbox

WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/aprova-sandbox"]
