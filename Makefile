GOBUILD = go build
GOTEST = go test

parse-arin:
	$(GOBUILD) -o ./build/parse-arin .

install:
	@cp ./build/parse-arin /usr/local/bin/

test:
	$(GOTEST) -v ./...
