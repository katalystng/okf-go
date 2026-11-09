# okf-go checks its own docs with the tool it builds. The layers order the graph's colors.
LAYERS := intent,spec,product

.PHONY: build test vet docs-viz verify-docs verify

build:
	go build -o bin/okf ./cmd/okf

test:
	go test ./...

vet:
	go vet ./...

docs-viz: build
	bin/okf viz -layers $(LAYERS) docs

verify-docs: build
	bin/okf check -strict docs
	bin/okf viz -check -layers $(LAYERS) docs

verify: vet test verify-docs
