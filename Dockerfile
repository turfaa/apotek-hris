FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:latest AS build

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o /hris

# Generate API documentation using Redocly
FROM --platform=${BUILDPLATFORM:-linux/amd64} node:24-bookworm AS docs

WORKDIR /docs

COPY docs/openapi.yaml ./

RUN npm install -g @redocly/cli && \
  redocly build-docs openapi.yaml --output index.html

FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static-debian12 AS release

WORKDIR /

COPY --from=build /hris /
COPY --from=build /app/migrations /migrations
COPY --from=build /app/docs/openapi.yaml /docs/openapi.yaml
COPY --from=docs /docs/index.html /docs/index.html

USER nonroot:nonroot

EXPOSE 8090

CMD ["/hris", "serve"]
