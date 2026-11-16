FROM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS compilacao

ENV CGO_ENABLED=0 GOTOOLCHAIN=local

WORKDIR /fonte
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN go build -trimpath -ldflags="-s -w" -o /saida/aprova-api ./cmd/api \
 && go build -trimpath -ldflags="-s -w" -o /saida/aprova-worker ./cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS api

COPY --from=compilacao /saida/aprova-api /usr/local/bin/aprova-api

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/aprova-api"]

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS worker

COPY --from=compilacao /saida/aprova-worker /usr/local/bin/aprova-worker

ENTRYPOINT ["/usr/local/bin/aprova-worker"]
