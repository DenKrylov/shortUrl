FROM	golang:1.17.2-alpine3.14

WORKDIR /www

COPY	src .

RUN		go mod init shortUrl && go mod tidy && go build main.go && \
		chmod 777 main

CMD		[ "./main" ]