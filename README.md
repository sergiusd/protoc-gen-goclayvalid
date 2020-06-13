# protoc-gen-goclayvalid
protoc plugin for generation of json validation rules when using goclay and validator.v10

## Example
```
make generate
```
See output/example/example.validate.json generated for api/example.proto

## Install to local directory
```
GOBIN=./bin go install github.com/sergiusd/protoc-gen-goclayvalid
```

## Paramsters

* `verbose` - mode verbosity
* `original_field_name` - use original field name instead of json_name

## Usage
Parameters may be set for additional information
```
protoc \
    --plugin=protoc-gen-goclay=bin/protoc-gen-goclay \
    --plugin=protoc-gen-gofast=bin/protoc-gen-gofast \
    --plugin=protoc-gen-goclayvalid=bin/protoc-gen-goclayvalid \
    -I../api/:vendor.pb \
    --gofast_out=plugins=grpc:./output \
    --goclay_out=impl_path=internal/api/example:./output \
    --goclayvalid_out=pretty:./output \
    api/example.proto;
```