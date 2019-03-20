FROM golang:1.11 as builder
ENV GO111MODULE=on
ARG PKG=/go/src/github.com/mountkin/go-salad/webhook-relay

ADD go.mod go.sum $PKG/
WORKDIR $PKG
RUN go mod download

ADD . $PKG
RUN go build -tags "netgo osusergo" -ldflags="-s -w" -o /tmp/webhook

FROM alpine
RUN apk add --update --no-cache ca-certificates
COPY --from=builder /tmp/webhook /bin/
CMD ["/bin/webhook", "-mode", "client"]
