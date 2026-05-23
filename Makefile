.PHONY: proto

PROTO_DIR = api/trend/v1
OUT_DIR = internal/transport/grpc/pb

proto:
	@mkdir -p $(OUT_DIR)
	protoc -I $(PROTO_DIR) \
			--go_out=$(OUT_DIR) --go_opt=paths=source_relative \
			--go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
			$(PROTO_DIR)/trend.proto
