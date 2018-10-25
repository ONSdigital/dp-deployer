clean:
	@rm -f latest.tar.gz dp-deployer

test:
	@go test -cover -short -race ./...

build: clean test
	@GOOS=linux GOARCH=amd64 go build -o dp-deployer cmd/dp-deployer/main.go

package: build
	@tar czf latest.tar.gz dp-deployer -C ../nomad-glue .

publish: package
	@aws s3 cp latest.tar.gz s3://ons-dp-deployments/dp-deployer/

.PHONY: clean test build package publish
