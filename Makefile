
test:
	go test -v -count=1 ./...

transport/http/%:
	@$(MAKE) --no-print-directory -C transport/http $@

transport/websocket/%:
	@$(MAKE) --no-print-directory -C transport/websocket $@

docs/examples/%:
	@$(MAKE) --no-print-directory -C docs/examples $@

deps:
	go get -u -d ./...

.PHONY: test test/examples t1 deps