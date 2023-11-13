.PHONY: all
all: ;

.PHONY: run
run: build
	./cmd/gophkeeper/gophkeeper -r :8080

.PHONY: build-client
build-client:
	go build -o ./cmd/client/bin/client_darwin64 ./cmd/client

.PHONY: build-client-manyplatform
build-client-manyplatform:
	GOOS=windows GOARCH=amd64 go build -o ./cmd/client/bin/client_win64 ./cmd/client
	GOOS=linux GOARCH=amd64 go build -o ./cmd/client/bin/client_linux64 ./cmd/client 
	GOOS=darwin GOARCH=amd64 go build -o ./cmd/client/bin/client_darwin64 ./cmd/client

.PHONY: run-client
run-client:
	./cmd/client/bin/client

.PHONY: build
build:
	go build -o ./cmd/gophkeeper/gophkeeper ./cmd/gophkeeper

.PHONY: restart-pg
restart-pg: stop-pg clean-data pg

.PHONY: pg
pg:
	docker run --rm \
		--name=gophkeeper-db \
		-v $(abspath ./db/init/):/docker-entrypoint-initdb.d \
		-v $(abspath ./db/data/):/var/lib/postgresql/data \
		-e POSTGRES_PASSWORD="P@ssw0rd" \
		-d \
		-p 5432:5432 \
		postgres:15.3

.PHONY: stop-pg
stop-pg:
	docker stop gophkeeper-db

.PHONY: clean-data
clean-data:
	sudo rm -rf ./db/data/

.PHONY: golangci-lint-run
golangci-lint-run: _golangci-lint-rm-unformatted-report

.PHONY: _golangci-lint-reports-mkdir
_golangci-lint-reports-mkdir:
	mkdir -p ./golangci-lint

.PHONY: _golangci-lint-run
_golangci-lint-run: _golangci-lint-reports-mkdir
	-docker run --rm \
    -v $(shell pwd):/app \
    -v $(shell pwd)/golangci-lint/cache:/root/.cache \
    -w /app \
    golangci/golangci-lint:v1.53.3 \
        golangci-lint run \
            -c .golangci-lint.yml \
	> ./golangci-lint/report-unformatted.json

.PHONY: _golangci-lint-format-report
_golangci-lint-format-report: _golangci-lint-run
	cat ./golangci-lint/report-unformatted.json | jq > ./golangci-lint/report.json

.PHONY: _golangci-lint-rm-unformatted-report
_golangci-lint-rm-unformatted-report: _golangci-lint-format-report
	rm ./golangci-lint/report-unformatted.json

.PHONY: golangci-lint-clean
golangci-lint-clean:
	sudo rm -rf ./golangci-lint 