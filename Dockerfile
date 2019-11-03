FROM golang:1.13.4

WORKDIR /go/src/github.com/benjivesterby/atomizer-agent
COPY . .

RUN go install -v ./...

CMD atomizer-agent -e