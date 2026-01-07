# ABOUTME: Makefile for devcontainer management commands.
# ABOUTME: Run from the .devcontainer directory or use make -C .devcontainer <target>.

.PHONY: help build rebuild up run shell stop down status

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the dev container image
	./devcontainer.sh build

rebuild: ## Rebuild the dev container (no cache)
	./devcontainer.sh rebuild

up: ## Start the dev container
	./devcontainer.sh up

run: ## Run a command in the dev container (usage: make run CMD="make test")
	./devcontainer.sh run $(CMD)

shell: ## Open an interactive shell in the dev container
	./devcontainer.sh shell

stop: ## Stop the running dev container
	./devcontainer.sh stop

down: ## Stop and remove the dev container
	./devcontainer.sh down

status: ## Show dev container status
	./devcontainer.sh status
