build:
	@go mod download
	@go build -o protoc-gen-restapi
	