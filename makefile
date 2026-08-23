all: svelte-deps svelte-build go-build

go-build:
	@echo "=== Building Go Server ==="
	go build -o app

svelte-build:
	@echo "=== Building Svelte Frontend ==="
	cd template && bun run build

svelte-deps:
	@echo "=== Checking for dependecies ==="
	@if [ ! -d "template/node_modules" ]; then \
		echo "Installing dependencies..."; \
		cd template && bun install; \
	fi