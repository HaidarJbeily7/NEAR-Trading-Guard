.PHONY: run test vet contract
run:
	cd backend && go run ./cmd/api
test:
	cd backend && go test ./...
vet:
	cd backend && go vet ./...
contract:
	cd contracts/guard && cargo build --locked --target wasm32-unknown-unknown --release
