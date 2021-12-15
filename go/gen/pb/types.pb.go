// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.6.1
// source: types.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Snippet
// Snippet represents a piece of text found in a HTML or text document.
// This is the type used while text is being processed. As snippets are read from the source document by the HTMLReader
// or TextReader, they are sent to the Tokeniser. This creates tokens based on the snippet's text and sends them to
// recognisers to find entities.
//
// swagger:model Snippet
type Snippet struct {
	//swagger:ignore
	state         protoimpl.MessageState
	//swagger:ignore
	sizeCache     protoimpl.SizeCache
	//swagger:ignore
	unknownFields protoimpl.UnknownFields

	// The text of the snippet. This is usually either the contents of a HTML text node or a line of a text document.
	Text           string `protobuf:"bytes,1,opt,name=text,proto3" json:"text,omitempty"`
	// The text after being normalised
	NormalisedText string `protobuf:"bytes,2,opt,name=normalisedText,proto3" json:"normalisedText,omitempty"`
	// Offset indicates the number of chars into the document the snippet appears
	Offset         uint32 `protobuf:"varint,3,opt,name=offset,proto3" json:"offset,omitempty"`
	// Xpath refers to the HTML element containing the snippet's text, if applicable.
	Xpath          string `protobuf:"bytes,4,opt,name=xpath,proto3" json:"xpath,omitempty"`
}

func (x *Snippet) Reset() {
	*x = Snippet{}
	if protoimpl.UnsafeEnabled {
		mi := &file_types_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Snippet) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Snippet) ProtoMessage() {}

func (x *Snippet) ProtoReflect() protoreflect.Message {
	mi := &file_types_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Snippet.ProtoReflect.Descriptor instead.
func (*Snippet) Descriptor() ([]byte, []int) {
	return file_types_proto_rawDescGZIP(), []int{0}
}

func (x *Snippet) GetText() string {
	if x != nil {
		return x.Text
	}
	return ""
}

func (x *Snippet) GetNormalisedText() string {
	if x != nil {
		return x.NormalisedText
	}
	return ""
}

func (x *Snippet) GetOffset() uint32 {
	if x != nil {
		return x.Offset
	}
	return 0
}

func (x *Snippet) GetXpath() string {
	if x != nil {
		return x.Xpath
	}
	return ""
}

// Entity
// An entity recognised in a piece of text.
// swagger:model Entity
type Entity struct {
	//swagger:ignore
	state         protoimpl.MessageState
	//swagger:ignore
	sizeCache     protoimpl.SizeCache
	// swagger:ignore
	unknownFields protoimpl.UnknownFields

	// The entity's text
	Name        string            `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Position within the enclosing text
	Position    uint32            `protobuf:"varint,2,opt,name=position,proto3" json:"position,omitempty"`
	// Xpath of the HTML element in which the text was found
	Xpath       string            `protobuf:"bytes,3,opt,name=xpath,proto3" json:"xpath,omitempty"`
	// Which recogniser was used to find the entity
	Recogniser  string            `protobuf:"bytes,4,opt,name=recogniser,proto3" json:"recogniser,omitempty"`
	// swagger:ignore
	// ?? what is this ??
	Identifiers map[string]string `protobuf:"bytes,5,rep,name=identifiers,proto3" json:"identifiers,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Metadata    []byte            `protobuf:"bytes,6,opt,name=metadata,proto3" json:"metadata,omitempty"`
}

func (x *Entity) Reset() {
	*x = Entity{}
	if protoimpl.UnsafeEnabled {
		mi := &file_types_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Entity) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Entity) ProtoMessage() {}

func (x *Entity) ProtoReflect() protoreflect.Message {
	mi := &file_types_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Entity.ProtoReflect.Descriptor instead.
func (*Entity) Descriptor() ([]byte, []int) {
	return file_types_proto_rawDescGZIP(), []int{1}
}

func (x *Entity) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Entity) GetPosition() uint32 {
	if x != nil {
		return x.Position
	}
	return 0
}

func (x *Entity) GetXpath() string {
	if x != nil {
		return x.Xpath
	}
	return ""
}

func (x *Entity) GetRecogniser() string {
	if x != nil {
		return x.Recogniser
	}
	return ""
}

func (x *Entity) GetIdentifiers() map[string]string {
	if x != nil {
		return x.Identifiers
	}
	return nil
}

func (x *Entity) GetMetadata() []byte {
	if x != nil {
		return x.Metadata
	}
	return nil
}

var File_types_proto protoreflect.FileDescriptor

var file_types_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x73, 0x0a,
	0x07, 0x53, 0x6e, 0x69, 0x70, 0x70, 0x65, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x78, 0x74,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x65, 0x78, 0x74, 0x12, 0x26, 0x0a, 0x0e,
	0x6e, 0x6f, 0x72, 0x6d, 0x61, 0x6c, 0x69, 0x73, 0x65, 0x64, 0x54, 0x65, 0x78, 0x74, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x6e, 0x6f, 0x72, 0x6d, 0x61, 0x6c, 0x69, 0x73, 0x65, 0x64,
	0x54, 0x65, 0x78, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0d, 0x52, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x12, 0x14, 0x0a, 0x05,
	0x78, 0x70, 0x61, 0x74, 0x68, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x78, 0x70, 0x61,
	0x74, 0x68, 0x22, 0x86, 0x02, 0x0a, 0x06, 0x45, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x08, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x14, 0x0a,
	0x05, 0x78, 0x70, 0x61, 0x74, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x78, 0x70,
	0x61, 0x74, 0x68, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x63, 0x6f, 0x67, 0x6e, 0x69, 0x73, 0x65,
	0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x63, 0x6f, 0x67, 0x6e, 0x69,
	0x73, 0x65, 0x72, 0x12, 0x3a, 0x0a, 0x0b, 0x69, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x65,
	0x72, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x45, 0x6e, 0x74, 0x69, 0x74,
	0x79, 0x2e, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x52, 0x0b, 0x69, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x65, 0x72, 0x73, 0x12,
	0x1a, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x06, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x1a, 0x3e, 0x0a, 0x10, 0x49,
	0x64, 0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0x45, 0x5a, 0x43, 0x67,
	0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x6d, 0x64, 0x63, 0x61, 0x74, 0x61, 0x70, 0x75, 0x6c, 0x74,
	0x2e, 0x69, 0x6f, 0x2f, 0x73, 0x6f, 0x66, 0x74, 0x77, 0x61, 0x72, 0x65, 0x2d, 0x65, 0x6e, 0x67,
	0x69, 0x6e, 0x65, 0x65, 0x72, 0x69, 0x6e, 0x67, 0x2f, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x2d,
	0x72, 0x65, 0x63, 0x6f, 0x67, 0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x67, 0x65, 0x6e, 0x2f,
	0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_types_proto_rawDescOnce sync.Once
	file_types_proto_rawDescData = file_types_proto_rawDesc
)

func file_types_proto_rawDescGZIP() []byte {
	file_types_proto_rawDescOnce.Do(func() {
		file_types_proto_rawDescData = protoimpl.X.CompressGZIP(file_types_proto_rawDescData)
	})
	return file_types_proto_rawDescData
}

var file_types_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_types_proto_goTypes = []interface{}{
	(*Snippet)(nil), // 0: Snippet
	(*Entity)(nil),  // 1: Entity
	nil,             // 2: Entity.IdentifiersEntry
}
var file_types_proto_depIdxs = []int32{
	2, // 0: Entity.identifiers:type_name -> Entity.IdentifiersEntry
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_types_proto_init() }
func file_types_proto_init() {
	if File_types_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_types_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Snippet); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_types_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Entity); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_types_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_types_proto_goTypes,
		DependencyIndexes: file_types_proto_depIdxs,
		MessageInfos:      file_types_proto_msgTypes,
	}.Build()
	File_types_proto = out.File
	file_types_proto_rawDesc = nil
	file_types_proto_goTypes = nil
	file_types_proto_depIdxs = nil
}
