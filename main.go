package main

import (
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	go_proto "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"google.golang.org/genproto/googleapis/api/annotations"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

type Builder struct {
	Request    *plugin.CodeGeneratorRequest
	Response   *plugin.CodeGeneratorResponse
	urlMessageMap map[string]string
	messageValidateMap map[string]interface{}
	verbose bool
}

func (b *Builder) protoByName(protoName string) *descriptor.FileDescriptorProto {
	for _, p := range b.Request.ProtoFile {
		if p.GetName() == protoName {
			return p
		}
	}
	return nil
}

func (b *Builder) addHttpRule(ruleList []*annotations.HttpRule, message string) {
	for _, rule := range ruleList {
		if r := rule.GetGet(); r != "" {
			b.urlMessageMap["GET " + r] = message
			b.log("    GET ", r)
		}
		if r := rule.GetPost(); r != "" {
			b.urlMessageMap["POST " + r] = message
			b.log("    POST ", r)
		}
		if r := rule.GetPut(); r != "" {
			b.urlMessageMap["PUT " + r] = message
			b.log("    PUT ", r)
		}
		if r := rule.GetDelete(); r != "" {
			b.urlMessageMap["DELETE " + r] = message
			b.log("    DELETE ", r)
		}
		if r := rule.GetPatch(); r != "" {
			b.urlMessageMap["PATCH " + r] = message
			b.log("    PATCH ", r)
		}
	}

}

func (b *Builder) collectUrlMessage()  {
	for _, filename := range b.Request.FileToGenerate {
		b.log("services: " +  filename)
		protoDesc := b.protoByName(filename)
		if protoDesc == nil {
			return
		}
		for _, service := range protoDesc.GetService() {
			b.log(service.GetName())
			for _, method := range service.Method {
				if method.Name != nil {
					b.log("  method: ", *method.Name, *method.InputType)
				}
				if method.Options != nil && go_proto.HasExtension(method.Options, annotations.E_Http) {
					extension, _ := go_proto.GetExtension(method.Options, annotations.E_Http)
					httpRule, ok := extension.(*annotations.HttpRule)
					if ok {
						b.addHttpRule([]*annotations.HttpRule{httpRule}, *method.InputType)
						b.addHttpRule(httpRule.AdditionalBindings, *method.InputType)
					}
				}
			}
		}
	}
}

func (b *Builder) collectMessage() {
	for _, filename := range b.Request.FileToGenerate {
		b.log("locations: ", filename)
		protoDesc := b.protoByName(filename)
		desc := protoDesc.GetSourceCodeInfo()
		locations := desc.GetLocation()
		for _, location := range locations {
			if len(location.GetPath()) > 2 {
				continue
			}

			if len(location.GetPath()) > 1 && location.GetPath()[0] == int32(4) {
				message := protoDesc.GetMessageType()[location.GetPath()[1]]
				path := "." + *protoDesc.Package + "." + message.GetName()
				b.log(path)

				b.collectMessageDeep(message, path)
			}
		}
	}
}

func (b *Builder) collectMessageDeep(message *descriptor.DescriptorProto, path string) {
	if len(message.NestedType) > 0 {
		for _, nestedMessage := range message.NestedType {
			nestedPath := path + "." + nestedMessage.GetName()
			b.log("  " + nestedPath)
			nestedData := make(map[string]interface{}, len(nestedMessage.Field))
			for _, field := range nestedMessage.Field {
				b.log("    " + field.GetJsonName() + ", " + strings.Join(b.getValidateOption(field.GetOptions()), ", "))
				nestedData[field.GetJsonName()] = b.getValidateOption(field.GetOptions())
			}
			b.messageValidateMap[nestedPath] = nestedData
			if len(nestedMessage.NestedType) > 0 {
				b.collectMessageDeep(nestedMessage, nestedPath)
			}
		}
	}

	if len(message.Field) > 0 {
		data := make(map[string]interface{}, len(message.Field))
		for _, field := range message.Field {
			if field.TypeName != nil {
				b.log("  " + field.GetJsonName() + ", " + *field.TypeName)
				data[field.GetJsonName()] = b.messageValidateMap[*field.TypeName]
			} else {
				b.log("  " + field.GetJsonName() + ", " + strings.Join(b.getValidateOption(field.GetOptions()), ", "))
				data[field.GetJsonName()] = b.getValidateOption(field.GetOptions())
			}
		}
		b.messageValidateMap[path] = data
	}
}

func (b *Builder) getValidateOption(data *descriptor.FieldOptions) []string {
	//extension, err := proto.GetExtension(data, gogoproto.E_Moretags)
	//b.log("  > ", extension, err.Error())
	//moreTags, ok := extension.(*proto.ExtensionDesc)
	//if !ok {
	//	return nil
	//}
	//b.log("  >> ", moreTags.Name, moreTags.ExtendedType.String())
	validateRE := regexp.MustCompile(`validate:\\"(.*)\\"`)
	if data == nil {
		return nil
	}
	d := validateRE.FindAllStringSubmatch(data.String(), 1)
	if len(d) != 0 && len(d[0]) > 1 {
		return strings.Split(d[0][1], ",")
	}
	return nil
}

func (b *Builder) createFile() error {
	for _, filename := range b.Request.FileToGenerate {
		data := make(map[string]interface{})
		for url, messageName := range b.urlMessageMap {
			if fields, ok := b.messageValidateMap[messageName]; ok {
				data[url] = fields
			}
		}

		var outfileName string
		outfileName = strings.Replace(filename, ".proto", ".validate.json", -1)
		var mdFile plugin.CodeGeneratorResponse_File
		mdFile.Name = &outfileName
		var jsonData []byte
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		content := string(jsonData)
		mdFile.Content = &content
		b.Response.File = append(b.Response.File, &mdFile)
		println(fmt.Sprintf("Created validate file: '%s'", filename))
		println()
	}
	return nil
}

func (b *Builder) generateCode() error {
	files := make([]*plugin.CodeGeneratorResponse_File, 0)
	b.Response.File = files

	b.collectUrlMessage()
	b.collectMessage()

	return b.createFile()
}

func (b *Builder) log(msg ...interface{}) {
	if b.verbose {
		fmt.Println(msg...)
	}
}

func main() {
	req := &plugin.CodeGeneratorRequest{}
	resp := &plugin.CodeGeneratorResponse{}

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	err = req.Unmarshal(data)
	if err != nil {
		panic(err)
	}

	builder := &Builder{
		Request:    req,
		Response:   resp,
		urlMessageMap: make(map[string]string),
		messageValidateMap: make(map[string]interface{}),
		verbose: false,
	}

	parameters := req.GetParameter()
	for _, element := range strings.Split(parameters, ",") {
		kv := strings.Split(element, "=")
		if kv[0] == "verbose" {
			builder.verbose = true
		}
	}

	err = builder.generateCode()
	if err != nil {
		panic(err)
	}

	marshalled, err := proto.Marshal(resp)
	if err != nil {
		panic(err)
	}
	_, _ = os.Stdout.Write(marshalled)
}
