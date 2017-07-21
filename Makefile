clean:
	@rm -f latest.tar.gz awdry

test:
	@go test -v -cover $(shell go list ./... | grep -v vendor)

build: clean test
	@GOOS=linux GOARCH=amd64 go build -o awdry cmd/awdry/main.go

package: build
	@tar czfv latest.tar.gz awdry scripts

publish: package
	@aws s3 cp latest.tar.gz s3://ons-dp-deployments/awdry/

.PHONY: clean build package publish
