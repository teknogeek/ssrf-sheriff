FROM golang:1.21-alpine AS build-env

WORKDIR /build
RUN go mod init ssrf-sheriff
COPY . .
RUN go get -d -v ./...
RUN go build -o ssrf-sheriff .

FROM alpine:3.19

WORKDIR /app
COPY --from=build-env /build/ssrf-sheriff /usr/local/bin/ssrf-sheriff
COPY config/base.example.yaml config/base.yaml

ENTRYPOINT ["ssrf-sheriff"]
