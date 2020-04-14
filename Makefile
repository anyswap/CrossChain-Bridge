.PHONY: all test clean fmt
.PHONY: swapserver swaporacle

GOBIN = ./bin
GOCMD = env GO111MODULE=on GOPROXY=https://goproxy.io go

swapserver:
	$(GOCMD) build -v -o bin/swapserver cmd/swapserver/*.go
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swapserver\" to launch swapserver."

swaporacle:
	$(GOCMD) build -v -o bin/swaporacle cmd/swaporacle/*.go
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swaporacle\" to launch swaporacle."

all:
	$(GOCMD) build -v ./...
	$(GOCMD) build -v -o bin/swapserver cmd/swapserver/*.go
	$(GOCMD) build -v -o bin/swaporacle cmd/swaporacle/*.go
	@echo "Done building."
	@echo "Find binaries in \"$(GOBIN)\" directory."

test:
	$(GOCMD) test ./...

clean:
	$(GOCMD) clean -cache
	rm -fr $(GOBIN)/*

fmt:
	./gofmt.sh
