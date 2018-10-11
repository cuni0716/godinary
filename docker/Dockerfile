FROM golang:alpine
WORKDIR ${GOPATH}/src/app

# install system dependencies
COPY ./glide.yaml /glide.yaml
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories \
    && apk add --no-cache --virtual .build-deps \
		g++ gcc make git ca-certificates \
	&& apk add --no-cache glide vips\ 
	&& glide install \
	&& apk del .build-deps

RUN pwd
RUN for i in cmd/*; do go build -o "bin/$(basename $i)" "$i/$(basename $i).go"; done

