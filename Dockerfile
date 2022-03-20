FROM golang:bullseye

RUN apt-get update && apt-get install vim

RUN mkdir /go/src/app

WORKDIR /go/src/app

ADD ./src /go/src/app

CMD [ "go run main.go" ]