# SiYuan's official image with its kernel rebuilt from the same release plus patches/.
# The kernel build matches upstream's Dockerfile; the front end is reused unchanged.
ARG SIYUAN_VERSION=v3.8.7

FROM golang:1.26-alpine AS kernel
ARG SIYUAN_VERSION
RUN apk add --no-cache gcc musl-dev git
WORKDIR /src
RUN git clone --quiet --depth 1 --branch "$SIYUAN_VERSION" --filter=blob:none --sparse \
        https://github.com/siyuan-note/siyuan.git . \
    && git sparse-checkout set kernel
COPY patches/ /patches/
RUN git apply /patches/*.patch
WORKDIR /src/kernel
ENV CGO_ENABLED=1
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg \
    go test -tags "fts5 sqlcipher" ./mcp/ \
    && go test -tags "fts5 sqlcipher" -run '^TestAttributeViewPropertyGuard|^TestSystemPageGuard' ./model/ \
    && go build -tags "fts5 sqlcipher" -ldflags "-s -w"

FROM b3log/siyuan:${SIYUAN_VERSION}
COPY --from=kernel --chmod=755 /src/kernel/kernel /opt/siyuan/kernel
