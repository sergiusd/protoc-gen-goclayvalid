package main

import (
	"io/ioutil"
	"os"

	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func main() {
	req := &plugin.CodeGeneratorRequest{}

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	err = proto.Unmarshal(data, req)
	if err != nil {
		panic(err)
	}

	var path string
	for _, fn := range req.FileToGenerate {
		path = fn + ".bin"
		break
	}

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		panic(err)
	}
}
