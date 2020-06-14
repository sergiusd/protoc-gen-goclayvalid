# protoc-gen-goclayvalid

Plugin for [protocol buffer compiler](https://github.com/protocolbuffers/protobuf).

If you use a [goclay](https://github.com/utrack/clay) to generate HTTP handlers, and a  [validator.v10](https://github.com/go-playground/validator) for field validation, then this plugin is for you

The plugin is useful when you want to do validation on the front. To use the validation rules from your proto file, generate information about them in the life format and use this data on the front. Then there is no need for manual synchronization of validation rules.

## Example

By running the command you can see the result of the plugin:
```bash
make generate
```
See `output/example.validate.json` generated from `example/example.proto`.

## How to install in local directory

```bash
GOBIN=./bin go install github.com/sergiusd/protoc-gen-goclayvalid
```

## Parameters

* `verbose` - mode verbosity
* `original_field_name` - use original field name instead of json_name
* `pretty` - for pretty json

## Work result example

Source proto file

```proto
syntax = "proto3";

package v1;

import "github.com/gogo/protobuf/gogoproto/gogo.proto";
import "google/api/annotations.proto";

service ExampleService {
    rpc ExampleCall(ExampleMessage) returns(ReturnType) {
        option (google.api.http) = {
            get: "/example/1"
            additional_bindings {
                post: "/example/1/additional"
                body: "*"
            }
        };
    }
}

message ExampleMessage {
    string my_string = 1 [(gogoproto.moretags) = "validate:\"required,len=15\""];
}

message ReturnType {}
```

Generated JSON file

```json
{
  "GET /example/1": {
    "my_string": ["required", "len=15"]
  },
  "POST /example/1/additional": {
    "my_string": ["required", "len=15"]
  }
}
```

## Usage

An example of using a set of plugins

* `protoc-gen-gofast` for gRPC handlers
* `protoc-gen-goclay` for HTTP handlers
* `protoc-gen-goclayvalid` for generate JSON with validation rules

```bash
protoc \
    --plugin=protoc-gen-gofast=bin/protoc-gen-gofast \
    --plugin=protoc-gen-goclay=bin/protoc-gen-goclay \
    --plugin=protoc-gen-goclayvalid=bin/protoc-gen-goclayvalid \
    -I. \
    --gofast_out=plugins=grpc:./output \
    --goclay_out=impl_path=internal/api/example:./output \
    --goclayvalid_out=pretty:./output \
    example.proto;
```
