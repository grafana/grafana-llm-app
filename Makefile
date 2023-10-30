SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

frontend:
	cd llm-frontend-libs
	yarn install --immutable
	yarn run build-dev
	cd ../llmexamples-app
	npm install
	npm run build-dev
	cd ..
	npm install
	npm run build-dev
.PHONY: frontend

backend:
	mage build:backend && cd ./llmexamples-app && mage build:backend
.PHONY: backend

build: frontend backend
.PHONY: build

docker: build
	docker compose up
.PHONY: docker

clean:
	rm -r dist node_modules ./llm-frontend-libs/node_modules ./llm-frontend-libs/compiled ./llm-frontend-libs/dist \
	./llmexamples-app/dist ./llmexamples-app/node_modules
.PHONY: clean