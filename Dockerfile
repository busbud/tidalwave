# Builder
FROM golang:1.12-alpine3.9 as builder

RUN apk add --no-cache git build-base bash

WORKDIR /build
COPY ./go.mod ./go.sum ./
RUN go mod download

COPY . .
RUN make

# App
FROM alpine:3.9

RUN apk add --no-cache bash ca-certificates && rm -rf /usr/share/man /tmp/* /var/tmp/*
COPY --from=builder /build/tidalwave /usr/bin/tidalwave

CMD ["/usr/bin/tidalwave"]
