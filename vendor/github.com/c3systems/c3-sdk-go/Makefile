all: test

.PHONY: deps
deps:
	@echo "running dep ensure..." && \
	dep ensure -v && \
	$(MAKE) gxundo

.PHONY: gxundo
gxundo:
	@bash scripts/gxundo.sh vendor/

.PHONY: test
test:
	@go test -v *.go
