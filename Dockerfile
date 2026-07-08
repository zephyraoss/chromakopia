FROM docker.io/library/golang:1.25-bookworm AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 \
    go build -ldflags="-s -w" -o /out/chromakopia ./cmd/chromakopia

FROM docker.io/library/debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends libchromaprint-tools ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /out/chromakopia /usr/local/bin/chromakopia
ENTRYPOINT ["chromakopia"]
