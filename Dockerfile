# syntax=docker/dockerfile:1

# Build stage: compile a fully static, reproducible binary (pure stdlib, no CGO).
# Toolchain + flags mirror the Makefile so `make build` and the image produce
# byte-identical output. SOURCE_DATE_EPOCH is fixed for reproducibility.
FROM --platform=linux/amd64 golang:1.24.4-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 \
    GOTOOLCHAIN=go1.24.4 SOURCE_DATE_EPOCH=0 \
    go build -trimpath -buildvcs=false -ldflags='-s -w' -o /out/meow-server .

# Runtime stage: scratch — the server only listens (no outbound calls),
# so it needs no CA certs, shell, or DNS resolver.
FROM scratch
COPY --from=builder /out/meow-server /meow-server
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/meow-server"]
