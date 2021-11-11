#!/bin/bash

# ==============
# Cleaning
# ==============
echo "Are you sure? your data will be unrecoverable [y/n]"
read ans
if [ "$ans" == "y" ] || [ "$ans" == "yes" ]
then
    docker-compose down
    echo "Cleaning previous node data"
    sudo rm -fr ./data
    echo "Done!"
else
    echo "Cancelled"
    exit
fi
