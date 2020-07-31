FLAGS := -tags gm

test:
	go test -v $(FLAGS) img utils

run:
	go run $(FLAGS) uploader.go

runrace:
	go run -race $(FLAGS) uploader.go

.PHONY: test
