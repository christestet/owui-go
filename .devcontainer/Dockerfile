FROM golang:1.26.0

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    curl \
    git \
    shellcheck \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*
