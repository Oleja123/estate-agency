FROM golang:1.25-alpine

WORKDIR /app

RUN apk add --no-cache make
RUN apk add --no-cache bash

COPY cmd cmd
COPY db db
COPY docs docs
RUN mkdir ./images
COPY internal internal
COPY pkg pkg
COPY go.mod go.mod
COPY go.sum go.sum
COPY boot.sh boot.sh
COPY Makefile Makefile
COPY config.yaml config.yaml


RUN go mod download
RUN go install github.com/pressly/goose/v3/cmd/goose@latest
RUN make build
RUN chmod +x boot.sh

EXPOSE 8080

CMD ["./boot.sh"]