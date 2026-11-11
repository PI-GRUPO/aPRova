FROM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS compilacao

ENV CGO_ENABLED=0 GOTOOLCHAIN=local

WORKDIR /fonte
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN go build -trimpath -ldflags="-s -w" -o /saida/aprova-sandbox ./cmd/sandbox

FROM maven:3.9-eclipse-temurin-21

RUN apt-get update \
    && apt-get install -y --no-install-recommends git procps \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 aprova \
    && useradd --uid 10001 --gid aprova --create-home --shell /usr/sbin/nologin aprova

COPY tests/integration/testdata/java/ok /tmp/aprova-java-cache

RUN mkdir -p /opt/aprova-cache/m2 /opt/aprova-cache/home \
    && chown -R aprova:aprova /opt/aprova-cache /tmp/aprova-java-cache \
    && su -s /bin/sh -c 'HOME=/opt/aprova-cache/home mvn -B -Dmaven.repo.local=/opt/aprova-cache/m2 -f /tmp/aprova-java-cache/pom.xml test' aprova \
    && rm -rf /tmp/aprova-java-cache \
    && chmod -R a+rX /opt/aprova-cache

COPY --from=compilacao /saida/aprova-sandbox /usr/local/bin/aprova-sandbox

WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/aprova-sandbox"]
