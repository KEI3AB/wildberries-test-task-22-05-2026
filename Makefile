.PHONY: proto

PROTO_DIR = api
OUT_DIR = internal/transport/grpc/pb
MODULE_NAME = github.com/wildberries-test-task-22-05-2026

PROTO_FILES = $(shell find $(PROTO_DIR) -name "*.proto")

proto:
	@mkdir -p $(OUT_DIR)
	protoc -I $(PROTO_DIR) \
		--go_out=. --go_opt=module=$(MODULE_NAME) \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE_NAME) \
		$(PROTO_FILES)
