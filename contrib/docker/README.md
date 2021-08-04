# Docker

This directory contains the necessary files and scripts to produce and run smartbch docker images.

`Dockerfile` describes the building of the docker image itself. It can be used both to produce debug and minified release (default) images ready to be used for smartbch mainnet. If you need to build and experiment with debug image, comment everything after `# produce clean image` comment.

`docker-compose.yml` contains two configurations:
  * `smartbch` to join mainnet as validator
  * `smartbch-regtest` to start smartbch in a regtest mode (single-node testnet) for local development

`test-keys.txt` the file containing private keys to be mounted into regtest container. If file does not exist ot is empty, it will be populated. These private keys persist after `docker-compose down -v`.

`build.sh` contains a script to build the docker images for most common `amd64` and `arm64` architectures. After building, images will be uploaded to dockerhub.

Also the repository contains a GitHub actions workflow `.github/workflows/release.yml` which is triggered upon creation of a new release tag. It requires your repository to contain a secret named `DOCKERHUB_PASSWORD` containing the password to login to dockerhub under the account name corresponding to the repository name.


To start the mainnet validator node simply invoke `docker-compose up smartbch`.

To start the regtest local development node use `docker-compose up smartbch-regtest`.


## Manual setup

To run smartBCH via `docker-compose` you can execute the commands below! Note, the first time you run docker-compose it will take a while, as it will need to build the docker image.

```
# Generate a set of 10 test keys.
docker-compose run smartbch gen-test-keys -n 10 > test-keys.txt

# Init the node, include the keys from the last step as a comma separated list.
docker-compose run smartbch init mynode --chain-id 0x2711 \
    --init-balance=10000000000000000000 \
    --test-keys=`paste -d, -s test-keys.txt` \
    --home=/root/.smartbchd --overwrite

# Generate consensus key info
CPK=$(docker-compose run -w /root/.smartbchd/ smartbch generate-consensus-key-info)
docker-compose run --entrypoint mv smartbch /root/.smartbchd/priv_validator_key.json /root/.smartbchd/config

# Generate genesis validator
K1=$(head -1 test-keys.txt)
VAL=$(docker-compose run smartbch generate-genesis-validator $K1 \
  --consensus-pubkey $CPK \
  --staking-coin 10000000000000000000000 \
  --voting-power 1 \
  --introduction "tester" \
  --home /root/.smartbchd
  )
docker-compose run smartbch add-genesis-validator --home=/root/.smartbchd $VAL

# Start it up, you are all set!
# Note that the above generated 10 accounts are not unlocked, you have to operate them through private keys
docker-compose up
```
