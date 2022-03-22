FROM golang:bullseye

RUN apt-get update -y && apt-get install vim -y

RUN mkdir /go/src/app

WORKDIR /go/src/app

ADD ./src /go/src/app

CMD [ "/bin/bash" ]