# colima start
# docker build -t paulfdunn/go-quantstudio .
## run interactive
# docker run -it --name go-quantstudio -p 8080:8080 go-quantstudio:go-quantstudio /bin/bash
## run container
# docker run --name go-quantstudio -p 8080:8080 go-quantstudio:go-quantstudio
# docker container rm go-quantstudio
FROM golang:1.22-bullseye AS builder
COPY ./ /go/src/go-quantstudio/
WORKDIR /go/src/go-quantstudio/
# The tests cannot be run when building for Artifact Repository, as the platform is different.
# RUN go test -v ./... >test.log 2>&1
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

# Use a second build stage to reduce the image from > 900MB to < 200MB
FROM ubuntu:22.04
COPY --from=builder /go/src/go-quantstudio/go-quantstudio /go/src/go-quantstudio/
RUN apt-get update -y
RUN apt-get install vim sqlite3 -y
EXPOSE 8080
ENTRYPOINT ["/go/src/go-quantstudio/go-quantstudio", "-logfile=log.txt"]
