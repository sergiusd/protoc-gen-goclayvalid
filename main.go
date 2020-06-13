package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	//gogo_proto "github.com/gogo/protobuf/proto"
	//"github.com/gogo/protobuf/gogoproto"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"google.golang.org/genproto/googleapis/api/annotations"
)

func main() {
	err := func() error {
		builder, err := NewBuilder(os.Stdin)
		if err != nil {
			return err
		}

		if err := builder.parseParameters(); err != nil {
			return err
		}

		err = builder.generateCode()
		if err != nil {
			return err
		}

		return builder.write(os.Stdout)
	}()
	if err != nil {
		panic(fmt.Sprintf("Error protoc-gen-goclayvalid: %+v", err))
	}
}

type Builder struct {
	request              *plugin.CodeGeneratorRequest
	response             *plugin.CodeGeneratorResponse
	urlMessageMap        map[string]string
	messageValidateMap   map[string]interface{}
	verbose              bool
	useOriginalFieldName bool
	pretty               bool
}

func NewBuilder(input io.Reader) (*Builder, error) {
	req := &plugin.CodeGeneratorRequest{}
	resp := &plugin.CodeGeneratorResponse{}

	data, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	err = req.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	return &Builder{
		request:              req,
		response:             resp,
		urlMessageMap:        make(map[string]string),
		messageValidateMap:   make(map[string]interface{}),
		verbose:              false,
		useOriginalFieldName: false,
		pretty:               false,
	}, nil
}

func (b *Builder) parseParameters() error {
	parameters := b.request.GetParameter()
	for _, element := range strings.Split(parameters, ",") {
		kv := strings.Split(element, "=")
		switch kv[0] {
		case "verbose":
			b.verbose = true
		case "pretty":
			b.pretty = true
		case "original_field_name":
			b.useOriginalFieldName = true
		default:
			return errors.New("Unknown parameter '" + kv[0] + "'")
		}
	}
	return nil
}

func (b *Builder) protoByName(protoName string) *descriptor.FileDescriptorProto {
	for _, p := range b.request.ProtoFile {
		if p.GetName() == protoName {
			return p
		}
	}
	return nil
}

func (b *Builder) addHttpRule(ruleList []*annotations.HttpRule, message string) {
	for _, rule := range ruleList {
		if r := rule.GetGet(); r != "" {
			b.urlMessageMap["GET "+r] = message
			b.log("    GET ", r)
		}
		if r := rule.GetPost(); r != "" {
			b.urlMessageMap["POST "+r] = message
			b.log("    POST ", r)
		}
		if r := rule.GetPut(); r != "" {
			b.urlMessageMap["PUT "+r] = message
			b.log("    PUT ", r)
		}
		if r := rule.GetDelete(); r != "" {
			b.urlMessageMap["DELETE "+r] = message
			b.log("    DELETE ", r)
		}
		if r := rule.GetPatch(); r != "" {
			b.urlMessageMap["PATCH "+r] = message
			b.log("    PATCH ", r)
		}
	}

}

func (b *Builder) collectUrlMessage() {
	for _, filename := range b.request.FileToGenerate {
		b.log("services: " + filename)
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
				if method.Options != nil && proto.HasExtension(method.Options, annotations.E_Http) {
					extension, _ := proto.GetExtension(method.Options, annotations.E_Http)
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
	for _, filename := range b.request.FileToGenerate {
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
				fieldName := b.getFieldName(field)
				b.log("    " + fieldName + ", " + strings.Join(b.getValidateOption(field.GetOptions()), ", "))
				nestedData[fieldName] = b.getValidateOption(field.GetOptions())
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
			fieldName := b.getFieldName(field)
			if field.TypeName != nil {
				b.log("  " + fieldName + ", " + *field.TypeName)
				data[fieldName] = b.messageValidateMap[*field.TypeName]
			} else {
				b.log("  " + fieldName + ", " + strings.Join(b.getValidateOption(field.GetOptions()), ", "))
				data[fieldName] = b.getValidateOption(field.GetOptions())
			}
		}
		b.messageValidateMap[path] = data
	}
}

func (b *Builder) getFieldName(field *descriptor.FieldDescriptorProto) string {
	if b.useOriginalFieldName {
		return field.GetName()
	}
	return field.GetJsonName()
}

func (b *Builder) getValidateOption(data *descriptor.FieldOptions) []string {
	// TODO get validate data from struct
	//extension, err := gogo_proto.ExtensionDescs(data)
	//b.log("    > ", extension, err.Error())
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
	for _, filename := range b.request.FileToGenerate {
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
		var err error
		if b.pretty {
			jsonData, err = json.MarshalIndent(data, "", "  ")
		} else {
			jsonData, err = json.Marshal(data)
		}
		if err != nil {
			return err
		}
		content := string(jsonData)
		mdFile.Content = &content
		b.response.File = append(b.response.File, &mdFile)
		println(fmt.Sprintf("Created validate file: %s -> %s", filename, outfileName))
		println()
	}
	return nil
}

func (b *Builder) generateCode() error {
	files := make([]*plugin.CodeGeneratorResponse_File, 0)
	b.response.File = files

	b.collectUrlMessage()
	b.collectMessage()

	return b.createFile()
}

func (b *Builder) write(output io.Writer) error {
	marshalled, err := proto.Marshal(b.response)
	if err != nil {
		return err
	}
	_, err = output.Write(marshalled)
	return err
}

func (b *Builder) log(msg ...interface{}) {
	if b.verbose {
		for i, m := range msg {
			if i == len(msg)-1 {
				println(fmt.Sprintf("%v", m))
			} else {
				print(fmt.Sprintf("%v", m))
			}
		}
	}
}
