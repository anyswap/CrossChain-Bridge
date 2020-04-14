#!/bin/sh

find_files() {
  find . ! \( \
      \( \
        -path '.github' \
        -o -path '.git' \
        -o -path './bin' \
        -o -path '*/vendor/*' \
      \) -prune \
    \) -name '*.go'
}

GOFMT="gofmt -s -w"
GOIMPORTS="goimports -w"
find_files | xargs $GOFMT
find_files | xargs $GOIMPORTS
