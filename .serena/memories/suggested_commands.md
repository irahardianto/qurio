# Suggested Commands

## Deployment
- **Start System:** `docker-compose up -d`
- **Stop System:** `docker-compose down`
- **View Logs:** `docker-compose logs -f`

## Development
- **Go (API):** `go run main.go` (or `go run ./cmd/server`)
- **Frontend (Vue):** `npm run dev` (likely in frontend dir)
- **Python (Worker):** `python main.py` (or `python -m worker`)

## Verification
- **Health Check:** `curl http://localhost:8081/health`
- **MCP Endpoint:** `http://localhost:8081/mcp`
- **Admin UI:** `http://localhost:3000`
- **E2E Tests:** `cd apps/e2e && npx playwright test`

## Utilities
- **Linting (Go):** `golangci-lint run`
- **Linting (TS):** `npm run lint`
- **Formatting:** `go fmt ./...`, `prettier --write .`
