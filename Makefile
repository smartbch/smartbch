export HOST_SRC_PATH=/Users/naach/dev/sandbox/sandbox-af/smartbch


up:
	docker-compose up -d smartbch_genesis

up-multi:
	docker-compose up -d

down:
	docker-compose down

clean:
	bash clean.sh

init:
	docker-compose up

init-both:
	bash init-both-node.sh

reset: down up
