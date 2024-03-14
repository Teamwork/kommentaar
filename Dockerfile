FROM golang:1.22-alpine

VOLUME /code
VOLUME /config
VOLUME /output

COPY . /go/src/github.com/teamwork/kommentaar
COPY config.example /config/kommentaar.conf

CMD ["/go/src/github.com/teamwork/kommentaar/run.sh"]
