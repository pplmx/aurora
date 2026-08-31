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


### Deploy
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/out/aurora /aurora

EXPOSE 6666 8888 12345

CMD ["/aurora"]
