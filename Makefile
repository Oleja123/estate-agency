COVERAGE_FILE ?= coverage.out

TARGET ?= run # CHANGE THIS TO YOUR BINARY NAME
MIGRATIONS_DIR := "db/migrations"
PG_DSN := "postgres://root:root@properties_db:5432/agency?sslmode=disable"
PG_DSN_TEST := "postgres://root:root@properties_db:5432/test?sslmode=disable"

.PHONY: build
build:
	@echo "Выполняется go build для таргета ${TARGET}"
	@mkdir -p .bin
	@go build -o ./bin/${TARGET} ./cmd/${TARGET}

## test: run all tests
.PHONY: test
test:
	@go test -coverpkg='github.com/es-debug/backend-academy-2024-go-template/...' --race -count=1 -coverprofile='$(COVERAGE_FILE)' ./...
	@go tool cover -func='$(COVERAGE_FILE)' | grep ^total | tr -s '\t'

.PHONY: migrate-generate
migrate-generate:
	$(GOPATH)/bin/goose -dir $(MIGRATIONS_DIR) create $(name) sql

.PHONY: migrate-up
migrate-up:
	$(GOPATH)/bin/goose -dir $(MIGRATIONS_DIR) postgres $(PG_DSN) up

.PHONY: migrate-down
migrate-down:
	$(GOPATH)/bin/goose -dir $(MIGRATIONS_DIR) postgres $(PG_DSN) down

.PHONY: migrate-up-test
migrate-up-test:
	$(GOPATH)/bin/goose -dir $(MIGRATIONS_DIR) postgres $(PG_DSN_TEST) up

.PHONY: migrate-down-test
migrate-down-test:
	$(GOPATH)/bin/goose -dir $(MIGRATIONS_DIR) postgres $(PG_DSN_TEST) down