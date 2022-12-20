all: docker-clean docker-build docker-create

docker-build:
	docker build --tag url-shortener .

docker-create:
	docker create \
		-p 8085:8085 \
		-v ${CURDIR}/src/:/app \
		--name url-shortener \
		url-shortener

docker-start:
	docker start url-shortener

docker-stop:
	docker stop url-shortener

docker-clean:
	-docker rm url-shortener
	-docker rmi url-shortener

