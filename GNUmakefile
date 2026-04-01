.PHONY: help lint testacc

help:
	@echo "Available targets:"
	@echo "  lint - Run MegaLinter checks on project"

# Run MegaLinter for project linting
lint:
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock:rw \
		-v $(PWD):/tmp/lint:rw \
		ghcr.io/oxsecurity/megalinter-go:v9

# Run acceptance tests
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m
