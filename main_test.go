package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	dir := "./test/"
	files, err := ioutil.ReadDir(dir)
	assert.Nil(t, err)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".proto.bin") {
			continue
		}

		t.Run("Test file "+file.Name(), func(t *testing.T) {
			fr, err := os.Open(dir + file.Name())
			assert.Nil(t, err)
			defer fr.Close()

			var buf []byte
			gotBuffer := bytes.NewBuffer(buf)
			err = processProto(fr, gotBuffer)
			assert.Nil(t, err)

			gotResult := gotBuffer.Bytes()
			for i, b := range gotResult {
				if b == '{' {
					gotResult = gotResult[i:]
					break
				}
			}

			result, err := ioutil.ReadFile(dir + strings.TrimRight(file.Name(), ".proto.bin") + ".json")
			assert.Nil(t, err)

			gotData, err := unmarshall(gotResult)
			assert.Nil(t, err)
			data, err := unmarshall(result)
			assert.Nil(t, err)
			assert.Equal(t, gotData, data)
		})
	}
}

func unmarshall(data []byte) (map[string]interface{}, error) {
	var ret map[string]interface{}
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}
