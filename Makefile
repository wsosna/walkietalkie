.PHONY: dev
dev:
	@set -a && source .dev/.env && set +a && go run main.go
