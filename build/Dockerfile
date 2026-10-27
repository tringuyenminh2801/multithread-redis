# Build from the repo root:  docker build -f build/Dockerfile -t multithreaded-redis .

FROM golang:1.23.2 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# distroless: no shell, no package manager, runs as non-root
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server

EXPOSE 6379 2112

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/server", "healthcheck"]

ENTRYPOINT ["/server"]
