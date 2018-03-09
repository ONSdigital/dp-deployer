clean:
	@rm -f latest.tar.gz awdry

test:
	@go test -v -cover -short -race ./...

build: clean test
	@GOOS=linux GOARCH=amd64 go build -o awdry cmd/awdry/main.go

package: build
	@tar czf latest.tar.gz awdry -C ../nomad-glue .

publish: package
	@aws s3 cp latest.tar.gz s3://ons-dp-deployments/awdry/

.PHONY: clean test build package publish
