FROM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS compilacao

ENV CGO_ENABLED=0 GOTOOLCHAIN=local

WORKDIR /fonte
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN go build -trimpath -ldflags="-s -w" -o /saida/aprova-sandbox ./cmd/sandbox

FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_INPUT=1

RUN apt-get update \
    && apt-get install -y --no-install-recommends git \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 aprova \
    && useradd --uid 10001 --gid aprova --create-home --shell /usr/sbin/nologin aprova

RUN python -m pip install --no-cache-dir pytest pytest-json-report pytest-timeout

COPY --from=compilacao /saida/aprova-sandbox /usr/local/bin/aprova-sandbox

WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/aprova-sandbox"]
