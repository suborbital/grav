
test:
	go test -v -count=1 ./...

docs/examples/%:
	@$(MAKE) --no-print-directory -C docs/examples $@

deps:
	go get -u -d ./...

.PHONY: test test/examples deps