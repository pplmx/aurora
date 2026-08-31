# syntax=docker/dockerfile:1
### Build
FROM golang:1.26-alpine AS builder

# Version/build-time injected so `aurora version` inside the image reports the
# real source ref instead of the 0.0.1 placeholder (set by `just image`;
# default to dev/unknown so ad-hoc `docker build` still yields a sane value).
ARG VERSION=dev
ARG BUILD_TIME=unknown

WORKDIR /app

ENV GOOS linux
ENV CGO_ENABLED 0
ENV GOPROXY https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# build the binary: -ldflags="-w -s" for the much smaller binary
# -trimpath remove all file system paths from the resulting executable.
# Version/build-time are -X-injected (TASK-206).
RUN go build -trimpath -ldflags="-w -s -X github.com/pplmx/aurora/cmd/aurora/cmd.Version=${VERSION} -X github.com/pplmx/aurora/cmd/aurora/cmd.BuildTime=${BUILD_TIME}" -o ./out/aurora ./cmd/aurora

# Pre-create the runtime state directories so the distroless deploy stage can
# COPY them (gcr.io/distroless has no shell — RUN mkdir/chown would fail).
RUN mkdir -p /out/data /out/logs


### Deploy
FROM gcr.io/distroless/static:nonroot

# WORKDIR /app makes the default `./data/aurora.db` and `./logs` resolve under
# those paths, matching the compose named volumes (aurora-data:/app/data,
# aurora-logs:/app/logs) — without it CWD was / and state landed in the
# ephemeral rootfs, silently vanishing on every `--rm` one-shot run.
WORKDIR /app

# Bake the state dirs in owned by the distroless nonroot uid (65532) so Docker
# named-volume mounts inherit writable ownership for the runtime user.
COPY --from=builder --chown=65532:65532 /out/data /app/data
COPY --from=builder --chown=65532:65532 /out/logs /app/logs

COPY --from=builder /app/out/aurora /aurora

# The image is a CLI toolbox (DEC-009): /aurora is the entrypoint so
# `docker run ... aurora lottery create ...` and compose's one-shot
# `docker compose run --rm aurora lottery create ...` resolve the subcommand
# (without it Docker tried to exec a binary literally named `lottery`).
ENTRYPOINT ["/aurora"]

# No EXPOSE: the image ships only the CLI and has no listener, and the
# Web UI is a separate non-containerised cmd/api binary (DEC-009, ISS-192).
