module github.com/hyperledger/fabric

go 1.15

require (
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/miekg/pkcs11 v1.1.1
	github.com/pkg/errors v0.9.1
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.68.0
	google.golang.org/protobuf v1.35.0 // indirect
)

require (
	github.com/golang/protobuf v1.5.2
	github.com/hyperledger/fabric-amcl v0.0.0-20210603140002-2670f91851c8
	github.com/hyperledger/fabric-protos-go v0.0.0-20201028172056-a3136dde2354
	github.com/sykesm/zap-logfmt v0.0.4
	golang.org/x/crypto v0.28.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.28.0 // indirect
)

replace google.golang.org/grpc => ./third/grpc
