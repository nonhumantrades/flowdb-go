proto:
	cd proto && ./build.sh

cli: 
	@go run cmd/flowdb/main.go

.PHONY: proto cli