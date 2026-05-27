FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /usr/local/bin/twgts twgts.go

FROM debian:bookworm-slim AS taskwarrior-builder
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wget cmake build-essential && rm -rf /var/lib/apt/lists/* && \
    wget -q https://github.com/GothenburgBitFactory/taskwarrior/releases/download/v3.4.2/task-3.4.2.tar.gz -O - | tar -xz -C /tmp && \
    cd /tmp/task-3.4.2 && cmake -S . -B build -DCMAKE_BUILD_TYPE=Release && cmake --build build && \
    cmake --install build && \
    rm -rf /tmp/task-3.4.2

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends supervisor ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=taskwarrior-builder /usr/local/ /usr/local/
RUN useradd -m -s /bin/bash app && \
    printf '%s\n' '# Taskwarrior config for app user' 'data.location=/data/.task' > /home/app/.taskrc && \
    chown app:app /home/app/.taskrc
COPY --from=builder /usr/local/bin/twgts /usr/local/bin/twgts
COPY supervisord.conf /etc/supervisor/conf.d/twgts.conf
USER app
WORKDIR /data
EXPOSE 9090
ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/twgts.conf"]
