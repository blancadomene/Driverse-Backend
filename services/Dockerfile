FROM golang:latest

ARG MICROSERVICE
ENV MICROSERVICE=${MICROSERVICE}

ENV GO111MODULE=off

RUN go get github.com/Sirupsen/logrus
RUN go get github.com/go-sql-driver/mysql
RUN go get github.com/gorilla/mux
RUN go get github.com/prometheus/client_golang/prometheus
RUN go get github.com/prometheus/client_golang/prometheus/promauto
RUN go get github.com/prometheus/client_golang/prometheus/promhttp

COPY ${MICROSERVICE}/main.go /tmp/main.go
RUN go build -o /main /tmp/main.go

ENTRYPOINT ["/main"]