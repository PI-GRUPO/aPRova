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

RUN python -m pip install --no-cache-dir pytest pytest-json-report

WORKDIR /workspace

USER aprova

ENTRYPOINT ["/bin/sh", "-c", "exec sleep infinity"]
