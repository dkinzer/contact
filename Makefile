.PHONY: default build 

default: build

build:
	GOOS=linux go build -o contact
	zip deployment.zip contact
