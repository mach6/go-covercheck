FROM golang:1.25 AS builder

RUN true \
  && apt-get update && apt-get install -y ca-certificates git binutils make \
  && update-ca-certificates

ADD . /tmp/build
WORKDIR /tmp/build

ENV CGO_ENABLED=0 \
    GO111MODULE=on

RUN go get ./... \
  && make build \
  && strip go-covercheck

# ---

FROM scratch

COPY --from=builder /tmp/build/go-covercheck /
CMD ["/go-covercheck"]
