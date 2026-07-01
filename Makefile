BINARY := hashcards

.PHONY: build
build:
	go build -o $(BINARY) ./cmd/$(BINARY)


kill-ports:
	@lsof -ti:3000 | xargs -r kill -9 2>/dev/null || true

.PHONY: server
server: kill-ports build
	./$(BINARY) serve --config=config.toml

init: build kill-ports
	#./hashcards migrate up --dir=pb_data
	./$(BINARY) superuser upsert admin@mail.internal password --dir=pb_data



.PHONY: build-container
build-container:
	docker compose -f compose.yaml up --build --force-recreate


# ─────────────────────────────────────────
#  Docker / deploy
# ─────────────────────────────────────────
.PHONY: build-image
build-image: ## Build Docker image
	docker build -t registry.internal/go-hashcards:latest .

# ─────────────────────────────────────────
.PHONY: build-image-no-cache
build-image-no-cache: ## Build Docker image
	docker build --no-cache -t registry.internal/go-hashcards:latest .

.PHONY: push-image
push-image: ## Push Docker image
	docker push registry.internal/go-hashcards:latest

.PHONY: deploy
deploy: build-image push-image ## (*) Deploy stack via Komodo
	docker exec -it komodo km x -y destroy-stack hashcards
	docker exec -it komodo km x -y pull-stack   hashcards
	docker exec -it komodo km x -y deploy-stack hashcards

