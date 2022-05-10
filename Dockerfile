FROM golang:latest
LABEL maincontainer=liujianhui<liujh1224@163.com>

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct

COPY . /$GOPATH/src/mini_cmux/
WORKDIR /$GOPATH/src/mini_cmux/example
RUN go build
CMD ["bin/bash","./example"]