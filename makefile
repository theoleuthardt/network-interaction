.PHONY: all run clean

GO := $(HOME)/.local/go/bin/go

all:
	$(GO) build -o network-interaction

run:
	$(GO) run main.go

clean:
	rm -f network-interaction
