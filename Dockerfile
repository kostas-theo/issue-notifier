FROM golang:1.13-alpine
WORKDIR /go/src/github.com/kostas-theo/issue-notifier

RUN apk update && apk add --no-cache git

COPY ./src/*.go ./

RUN go mod init issue-notifier
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/issue-notifier


ENTRYPOINT [ "/bin/issue-notifier" ]