export HOST_SRC_PATH=$(shell pwd)


up:
	cd components && docker-compose up -d smartbch_genesis

up-multi:
	cd components && docker-compose up -d

down:
	cd components && docker-compose down

clean:
	bash clean.sh

init-rebuild:
	make down && make clean && mkdir data && docker-compose up   --build --force-recreate

init:
	make down && make clean && mkdir data && docker-compose up


reset:
	cd components && down up
