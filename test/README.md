# Tests

Tests are kept outside `internal/` so production packages remain separated from test fixtures and integration dependencies.

Run unit tests:

```sh
go test ./test/unit/...
```

Integration tests require Redis or Valkey. Start the project dependencies first, then run:

```sh
go test -tags=integration ./test/integration/...
```

Set `REDIS_URL` when the integration service is not available at `redis://localhost:6379`.
