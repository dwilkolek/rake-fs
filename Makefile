GO ?= go
APP := rakefs
CMD := ./cmd/rakefs

MOUNTPOINT ?= /tmp/rakefs-mnt
COVERAGE_PROFILE ?= coverage.out
COVERAGE_HTML ?= coverage.html
OPEN ?= open

.DEFAULT_GOAL := help

.PHONY: help deps tidy build run test test-race vet fmt check coverage coverage-open clean

help:
	@echo "Available targets:"
	@echo "  make deps           Download Go module dependencies"
	@echo "  make tidy           Update go.mod and go.sum"
	@echo "  make build          Build ./$(APP)"
	@echo "  make run            Mount at $(MOUNTPOINT)"
	@echo "  make test           Run all tests"
	@echo "  make test-race      Run tests with the race detector"
	@echo "  make vet            Run go vet"
	@echo "  make fmt            Format Go source files"
	@echo "  make check          Run tests, race tests, and vet"
	@echo "  make coverage       Generate text and HTML coverage reports"
	@echo "  make coverage-open  Generate and open the HTML coverage report"
	@echo "  make clean          Remove generated build and coverage files"
	@echo
	@echo "Variables:"
	@echo "  MOUNTPOINT=/path    Override the default mountpoint"
	@echo "  OPEN=command        Override the HTML-opening command"

deps:
	$(GO) mod download

tidy:
	$(GO) mod tidy

build:
	$(GO) build -o $(APP) $(CMD)

run:
	mkdir -p "$(MOUNTPOINT)"
	$(GO) run $(CMD) "$(MOUNTPOINT)"

test:
	$(GO) test -count=1 ./...

test-race:
	$(GO) test -race -count=1 ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

check: test test-race vet

coverage:
	$(GO) test -count=1 -covermode=atomic -coverpkg=./... -coverprofile=$(COVERAGE_PROFILE) ./...
	$(GO) tool cover -func=$(COVERAGE_PROFILE)
	$(GO) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@echo "HTML coverage report: $(COVERAGE_HTML)"

coverage-open: coverage
	$(OPEN) "$(COVERAGE_HTML)"

clean:
	$(RM) $(APP) $(COVERAGE_PROFILE) $(COVERAGE_HTML)
