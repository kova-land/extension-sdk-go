.PHONY: generate check-schemas test vet lint

generate: ## Regenerate JSON schemas from Go protocol types
	go run ./cmd/generate-schemas/

check-schemas: ## Validate committed schemas match Go types (CI use)
	go run ./cmd/generate-schemas/ --check

test: ## Run all tests
	go test -v ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint
	golangci-lint run ./...
