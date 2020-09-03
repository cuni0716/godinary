get-deps:
	glide install

build-test:
	docker build --build-arg RUNTESTS=1 -t godinary:latest .

build:
	docker build -t godinary:latest .

build-dev:
	docker build -t godinary:dev -f Dockerfile.dev .

test:
	go test --cover godinary/http godinary/storage godinary/image

local-certs:
	openssl genrsa -out server.key 2048 && openssl ecparam -genkey -name secp384r1 -out server.key && openssl req -new -x509 -sha256 -key server.key -out server.pem -days 3650 -subj /C=US/ST=City/L=City/O=company/OU=SSLServers/CN=localhost/emailAddress=me@example.com

test-docker-image:
	docker run -p 3002:3002 --env-file .env --entrypoint sh -ti godinary:latest

run:
	docker run --rm -p 3000:3000 --env-file .env \
	       -v $$PWD/:cd ay/ \
		   -ti godinary:dev

sh-dev:
	docker run --rm -p 3000:3000 --env-file .env \
	       -v $$PWD/:/go/src/godinary/ \
		   -ti godinary:dev bash

up-dev:
	docker-compose -f docker-compose.yml -f docker-compose.override.yml up --build

up-godinary:
	docker-compose -f docker-compose.yml up godinary

down-dev:
	docker-compose -f docker-compose.yml -f docker-compose.override.yml down

run-rabbit-consumer-dev:
	docker-compose -f docker-compose.override.yml exec godinary sh -c "go run /go/src/godinary/cmd/rabbit/rabbit.go"

run-rabbit-publisher-dev:
	docker-compose -f docker-compose.override.yml exec godinary sh -c "go run /go/src/godinary/cmd/rabbit/rabbit_producer.go --image_url=$(image_url)"
