#!/usr/bin/env bash
# hooks/_selftest.sh — verify the hook contract (absolute paths, both argv and env).
echo "argv1 (source): $1"
echo "argv2 (output): $2"
echo "ENV  (source): $HATCHCARDS_SOURCE_DIR"
echo "ENV  (output): $HATCHCARDS_OUTPUT_DIR"
test -d "$1" && echo "source dir exists: OK" || echo "source dir MISSING"
mkdir -p "$2" && echo "output dir writable: OK"
