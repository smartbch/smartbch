#!/bin/bash

# check for args
if [ "$#" -ne 1 ]
then
  echo "Please specify node name such as happynode"
  exit 1
fi

docker-compose run smartbch init $1 --chain-id 0x2711
