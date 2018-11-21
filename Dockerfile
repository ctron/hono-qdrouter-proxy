FROM enmasseproject/qdrouterd-base:1.4.1

RUN dnf update -y
RUN dnf install -y golang procps-ng

RUN mkdir -p /root/go/src/github.com/ctron/hono-qdrouter-proxy
ADD . /root/go/src/github.com/ctron/hono-qdrouter-proxy

RUN cd /root/go/src/github.com/ctron/hono-qdrouter-proxy && go build -o /hono-qdrouter-proxy .

ENTRYPOINT /hono-qdrouter-proxy
