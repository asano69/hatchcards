BINARY := hashcards

.PHONY: frontend-deps
frontend-deps:
	cd frontend && pnpm install

.PHONY: build-frontend
build-frontend: frontend-deps
	cd frontend && pnpm run build

.PHONY: build
build: build-frontend
	go build -o $(BINARY) ./cmd/$(BINARY)

.PHONY: kill-ports
kill-ports:
	@lsof -ti:3000 | xargs -r kill -9 2>/dev/null || true
	@lsof -ti:3001 | xargs -r kill -9 2>/dev/null || true


.PHONY: server
server: kill-ports build
	#./hashcards migrate up --dir=pb_data
	./$(BINARY) superuser upsert admin@mail.internal password --dir=pb_data
	./$(BINARY) serve

# --------------
.PHONY: clean
	rm -fr ./tmp/ # air

# port: 3001
.PHONY: dev-front
dev-front: clean kill-ports
	npx concurrently -n "frontend,backend" -c "blue,green" "cd frontend && pnpm dev" "air"

# port: 3000
.PHONY: dev-back
dev-back: clean kill-ports
	npx concurrently -n "frontend,backend" -c "blue,green" "cd frontend && pnpm watch" "air"


.PHONY: test
test:
	go test ./...


migrate-collections:
	go run ./cmd/hashcards migrate collections


.PHONY: card
card:
	python3 ./scripts/question_to_json.py ~/Documents/obsidian/computer cards

# -----------


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

