
test:
	go test -v -count=1 ./...

transport/http:
	@$(MAKE) --no-print-directory -C transport/gravhttp $@

transport/http/%:
	@$(MAKE) --no-print-directory -C transport/gravhttp $@

transport/websocket:
	@$(MAKE) --no-print-directory -C transport/websocket $@

transport/websocket/%:
	@$(MAKE) --no-print-directory -C transport/websocket $@

deps:
	go get -u -d ./...

.PHONY: test t1 deps