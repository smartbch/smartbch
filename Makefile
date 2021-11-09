export HOST_SRC_PATH=/Users/naach/dev/sandbox/sandbox-af/smartbch


up:
	cd component && docker-compose up -d smartbch_genesis

up-multi:
	cd component && docker-compose up -d

down:
	cd component && docker-compose down

clean:
	bash clean.sh

init:
	docker-compose up

init-both:
	bash init-both-node.sh

reset:
	cd component && down up
