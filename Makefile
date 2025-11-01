BIN=.bin/templr

.PHONY: build test e2e golden clean

build:
	go build -o $(BIN) .

test: build
	go test ./e2e -v || true

e2e: build
	chmod +x tests/run_examples.sh
	tests/run_examples.sh

golden: build
	UPDATE_GOLDEN=1 tests/run_examples.sh

clean:
	rm -rf .bin .out
