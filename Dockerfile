# Sets linux/amd64 in case it's not injected by older Docker versions
ARG BUILDPLATFORM=linux/amd64
ARG ALPINE_VERSION=3.18
ARG GO_VERSION=1.20
ARG XCPUTRANSLATE_VERSION=v0.6.0
ARG GOLANGCI_LINT_VERSION=v1.41.1

FROM --platform=${BUILDPLATFORM} qmcgaw/xcputranslate:${XCPUTRANSLATE_VERSION} AS xcputranslate
FROM --platform=${BUILDPLATFORM} qmcgaw/binpot:golangci-lint-${GOLANGCI_LINT_VERSION} AS golangci-lint

FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS base
ENV CGO_ENABLED=0
WORKDIR /tmp/gobuild
RUN apk --update add git g++
COPY --from=xcputranslate /xcputranslate /usr/local/bin/xcputranslate
COPY --from=golangci-lint /bin /go/bin/golangci-lint
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/

FROM base AS test
# Note on the go race detector:
# - we set CGO_ENABLED=1 to have it enabled
# - we installed g++ in the base stage to support the race detector
ENV CGO_ENABLED=1
ENTRYPOINT go test -race -coverpkg=./... -coverprofile=coverage.txt -covermode=atomic ./...

FROM base AS lint
COPY .golangci.yml ./
RUN golangci-lint run --timeout=10m

FROM base AS tidy
RUN git init && \
  git config user.email ci@localhost && \
  git config user.name ci && \
  git add -A && git commit -m ci && \
  sed -i '/\/\/ indirect/d' go.mod && \
  go mod tidy && \
  git diff --exit-code -- go.mod

FROM base AS build
ARG TARGETPLATFORM
ARG VERSION=unknown
ARG CREATED="an unknown date"
ARG COMMIT=unknown
RUN GOARCH="$(xcputranslate translate -targetplatform=${TARGETPLATFORM} -field arch)" \
  GOARM="$(xcputranslate translate -targetplatform=${TARGETPLATFORM} -field arm)" \
  go build -trimpath -ldflags="-s -w \
  -X 'main.version=$VERSION' \
  -X 'main.created=$CREATED' \
  -X 'main.commit=$COMMIT' \
  " -o entrypoint cmd/devtainr/main.go

FROM scratch
ENTRYPOINT ["/devtainr"]
ARG VERSION=unknown
ARG BUILD_DATE="an unknown date"
ARG COMMIT=unknown
LABEL \
  org.opencontainers.image.authors="quentin.mcgaw@gmail.com" \
  org.opencontainers.image.created=$BUILD_DATE \
  org.opencontainers.image.version=$VERSION \
  org.opencontainers.image.revision=$COMMIT \
  org.opencontainers.image.url="https://github.com/qdm12/devtainr" \
  org.opencontainers.image.documentation="https://github.com/qdm12/devtainr" \
  org.opencontainers.image.source="https://github.com/qdm12/devtainr" \
  org.opencontainers.image.title="Devtainr" \
  org.opencontainers.image.description="Setup your development container with style"
COPY --from=build /tmp/gobuild/entrypoint /devtainr
