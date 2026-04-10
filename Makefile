.PHONY: build clean reset drill

BINARY := hashcards

build:
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY) && rm -f hashcards.db

reset:
	rm -f hashcards.db

drill:
	./hashcards drill example
