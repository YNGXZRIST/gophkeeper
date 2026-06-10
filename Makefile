.PHONY: proto
proto:
	protoc -I api \
		--go_out=. --go_opt=module=gophkeeper --go_opt=default_api_level=API_OPAQUE \
		--go-grpc_out=. --go-grpc_opt=module=gophkeeper \
		$(shell find api -name '*.proto')
