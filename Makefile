.PHONY: default build 

default: build

build:
	GOOS=linux go build -o contact
	zip deployment.zip contact

deploy:
	aws lambda update-function-code --function-name contact --zip-file fileb://deployment.zip
