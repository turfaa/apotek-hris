FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:latest AS build

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

ENV GOMODCACHE=/go/pkg/mod

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o /hris

FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static-debian12 AS release

WORKDIR /

COPY --from=build /hris /app/migrations /

USER nonroot:nonroot

EXPOSE 8090

CMD ["/hris", "serve"]
