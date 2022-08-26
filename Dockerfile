# syntax=docker/dockerfile:1
##
## Build
##
FROM golang:1.19-alpine AS builder

# set the working directory to the root of the project
WORKDIR /app

# Define build env
ENV GOOS linux
ENV CGO_ENABLED 0
ENV GOPROXY https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download


COPY . .

# Unit Tests
# RUN go test -v ./...

# build the binary: -ldflags="-w -s" for the much smaller binary
RUN go build -ldflags="-w -s" -o ./out/aurora main.go


###
### Deploy
###
FROM scratch

COPY --from=builder /app/out/aurora /aurora

# expose some ports
EXPOSE 6666 8888 12345

CMD ["/aurora"]
