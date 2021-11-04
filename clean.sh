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
    sudo rm -fr ./smartbch_genesis_data
    sudo rm -fr ./smartbch_node_data
    sudo rm -fr ./keys 
    echo "Done!"
else
    echo "Cancelled"
    exit 
fi