.PHONY: generate test vet

generate: ## Regenerate JSON schemas from Go protocol types
	go run ./cmd/generate-schemas/

test: ## Run all tests
	go test -v ./...

vet: ## Run go vet
	go vet ./...
