# syntax=docker/dockerfile:1

FROM golang:1.17-alpine

WORKDIR /app

COPY src/ ./

RUN go mod download

RUN go build -o /url-shortener

CMD [ "/url-shortener" ]
