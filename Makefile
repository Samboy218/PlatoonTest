SHELL=/bin/bash
.PHONY: all dev clean build env-up env-down run

all: clean build env-up run

dev: build run

build:
	@echo "Building..."
	@dep ensure
	@go build
	@echo "Build done."

env-up:
	@echo "Starting env..."
	@docker-compose up --force-recreate -d
	@for i in $$(seq 1 15); do let "sec = 15-$$i"; echo -ne "\033[2Ksleeping $$sec to finish setup"\\r; sleep 1; done
	@echo -e "\033[2Kenv up."

env-down:
	@echo "Tearing down env..."
	@docker-compose down
	@echo "env down."
run:
	@echo "Running application.."
	@./PlatoonTest
	@echo "run done."
clean: env-down
