FROM golang:latest AS builder

# Set the working directory and copy the code over
WORKDIR $GOPATH/src/mypackage/myapp/
COPY . .

# Fetch dependencies.
RUN go get -d -v

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -ldflags='-w -s -extldflags "-static"' -a \
      -o /go/bin/test-agent .

# Create the atomizer container
FROM scratch

# Copy the atomizer agent to thee new scratch container
COPY --from=builder /go/bin/test-agent /go/bin/test-agent

# Execute the atomizer agent
ENTRYPOINT ["/go/bin/test-agent"]
