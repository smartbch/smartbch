FROM ubuntu:20.04

MAINTAINER Josh Ellithorpe <quest@mac.com>

ARG GOLANG_VERSION="1.18"

ENV DEBIAN_FRONTEND="noninteractive"
RUN apt-get -y update && apt-get -y upgrade
RUN apt-get -y install gcc-8 g++-8 gcc g++ libgflags-dev libsnappy-dev zlib1g-dev libbz2-dev liblz4-dev libzstd-dev wget make git

RUN mkdir /build
WORKDIR /build
RUN wget https://dl.google.com/go/go${GOLANG_VERSION}.linux-amd64.tar.gz
RUN tar zxvf go${GOLANG_VERSION}.linux-amd64.tar.gz
RUN mv go /usr/local
RUN mkdir -p /go/bin
RUN wget https://github.com/facebook/rocksdb/archive/refs/tags/v5.18.4.tar.gz
RUN tar zxvf v5.18.4.tar.gz

ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH

WORKDIR /build/rocksdb-5.18.4
RUN make CC=gcc-8 CXX=g++-8 shared_lib

ENV ROCKSDB_PATH="/build/rocksdb-5.18.4"
ENV CGO_CFLAGS="-I/$ROCKSDB_PATH/include"
ENV CGO_LDFLAGS="-L/$ROCKSDB_PATH -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"
ENV LD_LIBRARY_PATH=$ROCKSDB_PATH

RUN mkdir /smart_bch
WORKDIR /smart_bch
RUN git clone https://github.com/smartbch/moeingevm.git
RUN git clone https://github.com/smartbch/smartbch.git

WORKDIR /smart_bch/moeingevm/evmwrap
RUN make

ENV EVMWRAP=/smart_bch/moeingevm/evmwrap/host_bridge/libevmwrap.so

WORKDIR /smart_bch/smartbch
RUN go install -tags cppbtree github.com/smartbch/smartbch/cmd/smartbchd

VOLUME ["/root/.smartbchd"]

ENTRYPOINT ["smartbchd"]
EXPOSE 8545 8546
