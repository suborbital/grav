
test:
	go test -v -count=1 ./...

# test one thing
t1:
	go test -timeout 30s -v -count=1 -run ^$(test)$$ ./grav

deps:
	go get -u -d ./...