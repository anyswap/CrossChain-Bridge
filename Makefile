.PHONY: all test testv clean fmt
.PHONY: swapserver swaporacle

GOBIN = ./build/bin
GOCMD = env GO111MODULE=on GOPROXY=https://goproxy.io,direct go

swapserver:
	$(GOCMD) run build/ci.go install ./cmd/swapserver
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swapserver\" to launch swapserver."

swaporacle:
	$(GOCMD) run build/ci.go install ./cmd/swaporacle
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swaporacle\" to launch swaporacle."

all:
	$(GOCMD) build -v ./...
	$(GOCMD) run build/ci.go install ./cmd/...
	@echo "Done building."
	@echo "Find binaries in \"$(GOBIN)\" directory."
	@echo ""
	@echo "Copy config-example.toml and config-tokens-example.toml to \"$(GOBIN)\" directory"
	@cp params/config-example.toml $(GOBIN)
	@cp params/config-tokenpair-example.toml $(GOBIN)

test: all
	$(GOCMD) test ./...

testv: all
	$(GOCMD) test -v ./...

clean:
	$(GOCMD) clean -cache
	rm -fr $(GOBIN)/*

fmt:
	./gofmt.sh
