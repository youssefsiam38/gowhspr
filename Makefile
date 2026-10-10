PREFIX ?= $(HOME)/.local

build:
	go build -trimpath -ldflags="-s -w" -o gowhspr .

install: build
	install -Dm755 gowhspr $(PREFIX)/bin/gowhspr

clean:
	rm -f gowhspr

.PHONY: build install clean
