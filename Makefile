.PHONY: build clean reset

BINARY := hashcards

build:
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY) && rm -f hashcards.db

reset:
	rm -f hashcards.db

.PHONY: serve
serve:
	./hashcards serve --config=hashcards.toml
