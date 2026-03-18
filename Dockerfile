FROM golang:1.26-alpine AS builder

COPY . /go/src/github.com/teamwork/kommentaar

RUN cd /go/src/github.com/teamwork/kommentaar && \
    GO111MODULE=off go install .

FROM golang:1.26-alpine

VOLUME /code
VOLUME /config
VOLUME /output

COPY --from=builder /go/bin/kommentaar /go/bin/kommentaar
COPY run.sh /run.sh
COPY config.example /config/kommentaar.conf

CMD ["/run.sh"]
