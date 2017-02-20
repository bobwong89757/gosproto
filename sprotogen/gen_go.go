package main

import (
	"bytes"
	"fmt"
	"go/token"
	"strings"

	"github.com/davyxu/gosproto/meta"
)

const goCodeTemplate = `// Generated by github.com/davyxu/gosproto/sprotogen
// DO NOT EDIT!

package {{.PackageName}}

import (
	"reflect"
	"github.com/davyxu/gosproto"
	"github.com/davyxu/goobjfmt"
	{{if .CellnetReg}}"github.com/davyxu/cellnet/codec/sproto"{{end}}
)

{{range $a, $enumobj := .Enums}}
type {{.Name}} int32
const (	{{range .GoFields}}
	{{$enumobj.Name}}_{{.Name}} {{$enumobj.Name}} = {{.Tag}} {{end}}
)

var {{$enumobj.Name}}_ValueByName = map[string]int32{ {{range .GoFields}}
	"{{.Name}}": {{.Tag}}, {{end}}
}

var {{$enumobj.Name}}_NameByValue = map[int32]string{ {{range .GoFields}}
	{{.Tag}}: "{{.Name}}" , {{end}}
}

func (self {{$enumobj.Name}}) String() string {
	return sproto.EnumName({{$enumobj.Name}}_NameByValue, int32(self))
}
{{end}}

{{range .Structs}}
type {{.Name}} struct{
	{{range .GoFields}}
		{{.FieldName}} {{.GoTypeName}} {{.GoTags}} 
	{{end}}
}

func (self *{{.Name}}) String() string { return goobjfmt.CompactTextString(self) }

{{end}}

var SProtoStructs = []reflect.Type{
{{range .Structs}}
	reflect.TypeOf((*{{.Name}})(nil)).Elem(), // {{.MsgID}} {{end}}
}

{{if .CellnetReg}}
func init() {
	sprotocodec.AutoRegisterMessageMeta(SProtoStructs)
}
{{end}}

`

// 字段首字母大写
func publicFieldName(name string) string {
	return strings.ToUpper(string(name[0])) + name[1:]
}

type goFieldModel struct {
	*meta.FieldDescriptor
}

func (self *goFieldModel) FieldName() string {
	pname := publicFieldName(self.Name)

	// 碰到关键字在尾部加_
	if token.Lookup(pname).IsKeyword() {
		return pname + "_"
	}

	return pname
}

func (self *goFieldModel) GoTypeName() string {

	var b bytes.Buffer
	if self.Repeatd {
		b.WriteString("[]")
	}

	if self.Type == meta.FieldType_Struct {
		b.WriteString("*")
	}

	// 字段类型映射go的类型
	switch self.Type {
	case meta.FieldType_Integer:
		b.WriteString("int")
	case meta.FieldType_Bool:
		b.WriteString("bool")
	case meta.FieldType_Struct,
		meta.FieldType_Enum:
		b.WriteString(self.Complex.Name)
	default:
		b.WriteString(self.Type.String())
	}

	return b.String()
}

func (self *goFieldModel) GoTags() string {

	var b bytes.Buffer

	b.WriteString("`sproto:\"")

	// 整形类型对解码层都视为整形
	switch self.Type {
	case meta.FieldType_Int32,
		meta.FieldType_Int64,
		meta.FieldType_UInt32,
		meta.FieldType_UInt64,
		meta.FieldType_Enum:
		b.WriteString("integer")
	default:
		b.WriteString(self.Kind())
	}

	b.WriteString(",")

	b.WriteString(fmt.Sprintf("%d", self.Tag))
	b.WriteString(",")

	if self.Repeatd {
		b.WriteString("array,")
	}

	b.WriteString(fmt.Sprintf("name=%s", self.FieldName()))

	b.WriteString("\"`")

	return b.String()
}

type goStructModel struct {
	*meta.Descriptor

	GoFields []goFieldModel

	f *goFileModel
}

func (self *goStructModel) MsgID() uint32 {
	return StringHash(self.MsgFullName())
}

func (self *goStructModel) MsgFullName() string {
	return self.f.PackageName + "." + self.Name
}

type goFileModel struct {
	*meta.FileDescriptor

	Structs []*goStructModel

	Enums []*goStructModel

	PackageName string

	CellnetReg bool
}

func addGoStruct(descs []*meta.Descriptor, callback func(*goStructModel)) {

	for _, st := range descs {

		stModel := &goStructModel{
			Descriptor: st,
		}

		for _, fd := range st.Fields {

			fdModel := goFieldModel{
				FieldDescriptor: fd,
			}

			stModel.GoFields = append(stModel.GoFields, fdModel)

		}

		callback(stModel)

	}
}

func gen_go(fileD *meta.FileDescriptor, packageName, filename string, cellnetReg bool) {

	fm := &goFileModel{
		FileDescriptor: fileD,
		PackageName:    packageName,
		CellnetReg:     cellnetReg,
	}

	addGoStruct(fileD.Structs, func(stModel *goStructModel) {
		stModel.f = fm
		fm.Structs = append(fm.Structs, stModel)
	})

	addGoStruct(fileD.Enums, func(stModel *goStructModel) {
		fm.Enums = append(fm.Enums, stModel)
	})

	generateCode("sp->go", goCodeTemplate, filename, fm, &generateOption{
		formatGoCode: true,
	})

}
