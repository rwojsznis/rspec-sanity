test:
	go test ./...

test_coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

dep:
	go mod download

vet:
	go vet

lint:
	golangci-lint run
