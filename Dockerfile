FROM golang:1.15.2-alpine

RUN mkdir /go/src/app

WORKDIR /go/src/app

# ADD ./src /go/src/app