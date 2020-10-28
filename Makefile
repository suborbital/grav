
test:
	go test -v -count=1 ./...

transport/http:
	@$(MAKE) --no-print-directory -C transport/gravhttp $@

transport/http/%:
	@$(MAKE) --no-print-directory -C transport/gravhttp $@

deps:
	go get -u -d ./...

.PHONY: test t1 deps