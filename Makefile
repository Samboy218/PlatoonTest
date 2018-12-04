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

install:
	sudo apt -y install docker.io
	sudo apt -y install docker-compose
	sudo usermod -a -G docker $USER
	tar -C /usr/local -xzf install/go1.10.5.linux-amd64.tar.gz
	echo "export GOPATH=$HOME/go" >> ~/.profile
	echo "export PATH=$PATH:$GOPATH/bin" >> ~/.profile
	echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.profile
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	echo "Now you need to reboot and then run \033[31;1;4mmake install-images\033[0m"

install-images:
	curl -sSL http://bit.ly/2ysbOFE | bash -s 1.0.5

