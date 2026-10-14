package protocol

//go:generate sh -c "trap 'rm -f .protoc-gen-go.tmp' EXIT && go build -o .protoc-gen-go.tmp google.golang.org/protobuf/cmd/protoc-gen-go && protoc --plugin=protoc-gen-go=.protoc-gen-go.tmp --go_out=. --go_opt=paths=source_relative message.proto rendezvous.proto"
