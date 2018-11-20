FROM fedora:29

RUN dnf update -y
RUN dnf install -y golang

ADD . /

RUN go build .
