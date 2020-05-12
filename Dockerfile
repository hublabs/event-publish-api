FROM pangpanglabs/golang:builder-beta AS builder
WORKDIR /go/src/github.com/hublabs/event-publish-api
COPY . .
# disable cgo
ENV CGO_ENABLED=0
# build steps
RUN echo ">>> 1: go version" && go version
RUN echo ">>> 2: go get" && go get -v -d
RUN echo ">>> 3: go install" && go install

# make application docker image use alpine
FROM pangpanglabs/alpine-ssl
WORKDIR /go/bin/
COPY --from=builder /go/bin/event-publish-api ./
EXPOSE 8000
CMD ["./event-publish-api"]