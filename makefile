all: go-build svelte-build

go-build:
	@echo "=== Building Go Server ==="
	go build -o app

svelte-build:
	@echo "=== Building Svelte Frontend ==="
	cd template && bun run build