FROM ubuntu:20.04 as builder

MAINTAINER Josh Ellithorpe <quest@mac.com>
MAINTAINER mainnet_pat

ARG TARGETPLATFORM
ARG GO_VERSION=1.16.6
ARG CGO_PATCH_VERSION=0.1.1
ARG SNAPPY_VERSION=1.1.8
ARG ROCKSDB_VERSION=5.18.4
ARG CONFIG_TAG=0.0.3
ARG GIT_TAG=0.3.0

# Update file limit
RUN sed -i -e '$a* soft nofile 65536\n* hard nofile 65536' /etc/security/limits.conf

# Install apt based dependencies
RUN apt-get -y update && apt-get -y upgrade && \
  DEBIAN_FRONTEND="noninteractive" apt-get -y install cmake gcc-8 g++-8 gcc g++ git \
  libgflags-dev zlib1g-dev libbz2-dev liblz4-dev libzstd-dev make vim wget && \
  apt-get clean && rm -rf /var/lib/apt/lists/*

# Setup build directory
WORKDIR /build

# Install Go
RUN export GOPLATFORM=$(echo ${TARGETPLATFORM} | tr "/" "-") && echo ${GOPLATFORM} && \
  wget -q https://dl.google.com/go/go${GO_VERSION}.${GOPLATFORM}.tar.gz && \
  tar zxf go${GO_VERSION}.${GOPLATFORM}.tar.gz && \
  rm -rf go${GO_VERSION}.${GOPLATFORM}.tar.gz && \
  mv go /usr/local && \
  mkdir -p /go/bin

ENV GOROOT=/usr/local/go \
  GOPATH=/go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH

# Patch Go for larger cgo stack size
RUN wget -q https://github.com/smartbch/patch-cgo-for-golang/archive/refs/tags/v${CGO_PATCH_VERSION}.tar.gz && \
  tar zxf v${CGO_PATCH_VERSION}.tar.gz && \
  rm v${CGO_PATCH_VERSION}.tar.gz && \
  cd patch-cgo-for-golang-${CGO_PATCH_VERSION} && cp *.c $GOROOT/src/runtime/cgo/ && \
  cd /build && rm -rf patch-cgo-for-golang-${CGO_PATCH_VERSION}

# Build libsnappy
RUN wget -q https://github.com/google/snappy/archive/refs/tags/${SNAPPY_VERSION}.tar.gz && \
  tar zxf ${SNAPPY_VERSION}.tar.gz && \
  rm ${SNAPPY_VERSION}.tar.gz && \
  cd snappy-${SNAPPY_VERSION} && mkdir build && cd build && cmake -DBUILD_SHARED_LIBS=On ../ && make -j`nproc` && make install && \
  cd /build && rm -rf snappy-${SNAPPY_VERSION}


# Build rocksdb
RUN wget -q https://github.com/facebook/rocksdb/archive/refs/tags/v${ROCKSDB_VERSION}.tar.gz && \
  tar zxf v${ROCKSDB_VERSION}.tar.gz && \
  rm v${ROCKSDB_VERSION}.tar.gz && \
  cd rocksdb-${ROCKSDB_VERSION} && make CC=gcc-8 CXX=g++-8 install-shared -j`nproc` && \
  strip --strip-debug /usr/local/lib/librocksdb.so.${ROCKSDB_VERSION} && \
  cd /build && rm -rf rocksdb-${ROCKSDB_VERSION}

ENV ROCKSDB_PATH="/usr/local" \
 CGO_CFLAGS="-I/$ROCKSDB_PATH/include" \
 CGO_LDFLAGS="-L/$ROCKSDB_PATH -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd" \
 LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH

# Create smartbch directory
WORKDIR /smart_bch

# Build libevmwrap.so
RUN git clone -b v${GIT_TAG} --depth 1 https://github.com/smartbch/moeingevm && \
  cd moeingevm/evmwrap && make -j`nproc`

ENV EVMWRAP=/smart_bch/moeingevm/evmwrap/host_bridge/libevmwrap.so

# Build and setup smartbchd
RUN git clone -b v${GIT_TAG} --depth 1 https://github.com/smartbch/smartbch && \
  cd smartbch && go build -tags cppbtree github.com/smartbch/smartbch/cmd/smartbchd && \
  cp /smart_bch/smartbch/smartbchd /build/smartbchd && \
  rm -rf /smart_bch/smartbch

WORKDIR /root
RUN /build/smartbchd init mynode --chain-id 0x2710 && \
  wget -q https://github.com/smartbch/artifacts/releases/download/v${CONFIG_TAG}/dot.smartbchd.tgz && \
  tar zxf dot.smartbchd.tgz && \
  rm dot.smartbchd.tgz && \
  cp -rf dot.smartbchd/* .smartbchd/

ENV PATH=/build:$PATH

# Go back to main workdir.
WORKDIR /build

VOLUME ["/root/.smartbchd"]

ENTRYPOINT ["smartbchd"]
EXPOSE 8545 8546


# produce clean image
FROM ubuntu:20.04

RUN --mount=type=bind,src=/,dst=/builder,from=builder \
  cd /builder && \
  cp -rn --parents lib/*/libgflags.* / && \
  cp -rn --parents usr/local/lib / && \
  cp -rn --parents root / && \
  cp -rn --parents build / && \
  cp -rn --parents smart_bch/moeingevm/evmwrap/host_bridge/libevmwrap.so /

WORKDIR /root

COPY init_regtest.sh /build/init_regtest.sh

ENV PATH=/build:$PATH\
    LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH \
    EVMWRAP=/smart_bch/moeingevm/evmwrap/host_bridge/libevmwrap.so

VOLUME ["/root/.smartbchd"]

ENTRYPOINT ["smartbchd"]
EXPOSE 8545 8546
