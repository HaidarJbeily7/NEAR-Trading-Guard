# Go 1.21's internal linker is incompatible with newer macOS loaders.
# Use Apple's linker on macOS; Linux CI uses Go's default.
ifeq ($(shell uname -s),Darwin)
GO_LINK_FLAGS := -ldflags=-linkmode=external
endif

.PHONY: run test vet contract
run:
	cd backend && go run $(GO_LINK_FLAGS) ./cmd/api
test:
	cd backend && go test $(GO_LINK_FLAGS) ./...
vet:
	cd backend && go vet ./...
contract:
	cd contracts/guard && cargo build --locked --target wasm32-unknown-unknown --release

.PHONY: intents
intents:
	cd backend && go run $(GO_LINK_FLAGS) ./cmd/intents $(ARGS)
