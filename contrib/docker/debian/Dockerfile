FROM golang:1.9 as builder

ENV REPO github.com/da-moon/soil
ENV BIN soil

ENV CGO_ENABLED 0

WORKDIR /go/src/${REPO}

COPY . ./

RUN go build -installsuffix cgo -ldflags "-s -w -X ${REPO}/cmd.V=$(git describe --always --tags --dirty)" -o /${BIN} ${REPO}/cmd/${BIN} \
  && go build -installsuffix cgo -ldflags "-s -w -X ${REPO}/cmd.V=$(git describe --always --tags --dirty)" -tags debug -o /${BIN}-debug ${REPO}/cmd/${BIN}

FROM alpine:3.6

COPY --from=builder /soil /soil-debug /usr/bin/

ENTRYPOINT ["/usr/bin/soil"]
