#!/bin/bash

# ==============
# Cleaning 
# ==============
echo "Cleaning previous node data"
sudo rm -fr ./smartbch_genesis_data
sudo rm -fr ./smartbch_node_data
sudo rm -fr ./keys 
echo "Done!"