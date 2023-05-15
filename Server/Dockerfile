FROM golang:latest
WORKDIR /Server
ADD ./ /Server

RUN go get
RUN go build

EXPOSE 8000

CMD ["./Server"]