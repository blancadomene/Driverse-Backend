FROM golang:latest

ARG MICROSERVICE
ENV MICROSERVICE=${MICROSERVICE}

ENV GO111MODULE=off

RUN go get github.com/go-sql-driver/mysql
RUN go get github.com/gorilla/mux

COPY ${MICROSERVICE}/main.go /tmp/main.go
RUN go build -o /main /tmp/main.go

ENTRYPOINT ["/main"]