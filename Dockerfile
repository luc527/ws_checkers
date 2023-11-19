FROM golang:1.21-alpine3.18
WORKDIR /go/src
COPY *.go go.mod go.sum ./
RUN CGO_ENABLED=0 GOOS=linux go install
CMD ["/go/bin/ws_checkers"]
