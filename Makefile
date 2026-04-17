BINARY := hashcards

.PHONY: build
build:
	go build -o $(BINARY) .

.PHONY: clean
clean:
	rm -f $(BINARY) && rm -f hashcards.db

.PHONY: reset
reset:
	rm -f hashcards.db

.PHONY: serve
serve:
	./hashcards serve --config=config.toml


.PHONY: build-container
build-container:
	docker compose -f compose.yaml up --build --force-recreate


# ─────────────────────────────────────────
#  Docker / deploy
# ─────────────────────────────────────────
.PHONY: build-image
build-image: ## Build Docker image
	docker build -t registry.internal/go-hashcards:latest .

.PHONY: push-image
push-image: ## Push Docker image
	docker push registry.internal/go-hashcards:latest

.PHONY: deploy
deploy: build-image push-image ## (*) Deploy stack via Komodo
	docker exec -it komodo km x -y destroy-stack hashcards
	docker exec -it komodo km x -y pull-stack   hashcards
	docker exec -it komodo km x -y deploy-stack hashcards
