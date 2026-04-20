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
	rm -fr ./data
	mkdir -p ./data
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

# ─────────────────────────────────────────
.PHONY: rust-upadte
rust-upadte:
	rm -fr rust-hashcards
	mkdir -p tmp
	git clone https://github.com/eudoxia0/hashcards.git tmp/hashcards
	mv tmp/hashcards rust-hashcards
	rm -fr tmp
	rm -fr rust-hashcards/.git
	rm -fr rust-hashcards/.github
	rm -fr rust-hashcards/.gitignore
	rm -fr rust-hashcards/CHANGELOG.xsd
	rm -fr rust-hashcards/codecov.yml
	rm -fr rust-hashcards/deny.toml
	rm -fr rust-hashcards/example/
	rm -fr rust-hashcards/flake.lock
	rm -fr rust-hashcards/flake.nix
	rm -fr rust-hashcards/pre-commit.sh
	rm -fr rust-hashcards/src/cmd/drill/favicon.png
	rm -fr rust-hashcards/test/
	# Not for malicious use, but intended to reduce token usage in Claude. m(_ _)m
	for f in $$(find . -name "*.rs"); do \
		awk 'BEGIN{skip=0} /^\/\/ Copyright/{skip=1} !skip{print} /^\/\/ limitations/{skip=0}' $$f > $$f.tmp && mv $$f.tmp $$f; \
	done
