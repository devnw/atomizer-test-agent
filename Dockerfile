FROM golang:latest AS builder

# Set the working directory and copy the code over
WORKDIR /build
COPY . .

# Fetch dependencies.
RUN go get -d -v

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -ldflags='-w -s -extldflags "-static"' -a \
      -o agent .

# Create the atomizer container
FROM scratch

# Copy the atomizer agent to thee new scratch container
COPY --from=builder /build/agent ./agent

# Execute the atomizer agent
ENTRYPOINT ["./agent"]
