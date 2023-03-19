#!/usr/bin/bash
protoc -I=proto \
    -I=./proto/google/api \
    --go_out=pb --go_opt=paths=source_relative \
    --go-grpc_out=pb --go-grpc_opt=paths=source_relative \
    --grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
    proto/service.proto
protoc-go-inject-tag -input="pb/*.pb.go"
