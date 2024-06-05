.PHONY: info fmt goimports gofumpt lint tidy go_fix go_vet golangci test coverage

info:
	go version

fmt: goimports gofumpt
	$(info === format done)

goimports:
	goimports -e -l -w -local github.com/peczenyj/ttempdir .

gofumpt:
	gofumpt -l -w -extra .

lint: tidy go_fix go_vet golangci
	$(info === lint done)

tidy:
	go mod tidy

go_fix:
	go fix ./...

go_vet:
	go vet -all ./...

golangci:
	golangci-lint run ./...

test:
	go test -v ./...

coverage:
	export GOEXPERIMENT="nocoverageredesign"
	go test -v -race -cover -covermode=atomic -coverprofile coverage.out ./...
