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
	make down && make clean && docker-compose up   --build --force-recreate

init:
	make down && make clean && mkdir data && docker-compose up

init-both:
	bash init-both-node.sh

reset:
	cd components && down up
