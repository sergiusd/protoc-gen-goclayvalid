#!/usr/bin/env bash

set -ex

rm -f ./test/*.bin
go build -o ./bin/protoc-gen-prepare ./test/prepare.go
pushd ./test
    for proto in $(ls -1 | grep \.proto); do
        protoc \
            --plugin=protoc-gen-prepare=../bin/protoc-gen-prepare \
            -I./:../example/vendor.pb \
            --prepare_out=original_field_name,pretty:. \
            ${proto};
    done
popd
