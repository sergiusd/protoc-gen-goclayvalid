package main

import (
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"io/ioutil"
	"os"
)

func main() {
	req := &plugin.CodeGeneratorRequest{}

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	err = req.Unmarshal(data)
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
