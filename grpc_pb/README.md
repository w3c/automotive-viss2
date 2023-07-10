## gRPC

The folowwing command builds the VISSv2.proto file:

protoc --go_out=. --go_opt=paths=source_relative     --go-grpc_out=. --go-grpc_opt=paths=source_relative     VISSv2.proto


