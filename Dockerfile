FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /usr/local/bin/twgts twgts.go

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends taskwarrior supervisor ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /usr/local/bin/twgts /usr/local/bin/twgts
COPY supervisord.conf /etc/supervisor/conf.d/twgts.conf
WORKDIR /data
EXPOSE 9090
ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/twgts.conf"]
