FROM fedora:29

RUN dnf update -y
RUN dnf install -y golang

RUN mkdir -p /root/go/src/github.com/ctron/hono-qdrouter-proxy
ADD . /root/go/src/github.com/ctron/hono-qdrouter-proxy

RUN go build -o /hono-qdrouter-proxy /root/go/src/github.com/ctron/hono-qdrouter-proxy

ENTRYPOINT /hono-qdrouter-proxy
