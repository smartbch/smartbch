FROM ubuntu:20.04

MAINTAINER Napat Charuphant <napat.charuphant1@gmail.com>

RUN apt-get -y update && apt-get -y upgrade
RUN apt-get -y install curl docker.io
RUN curl -L "https://github.com/docker/compose/releases/download/1.27.4/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
RUN chmod +x /usr/local/bin/docker-compose

WORKDIR /usr/src/app
COPY ./init-both-node.sh ./init-both-node.sh
RUN chmod +x /usr/src/app/init-both-node.sh
COPY ./init-mainnet.sh ./init-mainnet.sh
RUN chmod +x /usr/src/app/init-mainnet.sh


ENTRYPOINT ["/usr/src/app/init-both-node.sh"]
