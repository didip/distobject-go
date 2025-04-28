.PHONY: test bench docker-test

test:
	go test ./distobject/...

bench:
	go test -bench=. ./distobject/

docker-test:
	docker compose up --build --abort-on-container-exit --exit-code-from test-runner
