
SRC = ${wildcard *.go}

all: run

caddy: $(SRC) go.mod
	GONOSUMDB=github.com/smallstep xcaddy build --with github.com/johnweldon/unifi-api=.


.PHONY: validate
validate: caddy
	./caddy validate --config=deploy/Caddyfile

.PHONY: run
run: validate
	./caddy run --config=deploy/Caddyfile --watch

.PHONY: fmt
fmt: validate
	./caddy fmt deploy/Caddyfile --overwrite
