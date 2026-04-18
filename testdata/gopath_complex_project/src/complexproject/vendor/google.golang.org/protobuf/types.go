package protobuf

// ProtoMessage 协议消息
type ProtoMessage struct {
	Data     []byte
	Size     int64
	Version  int32
	Enabled  bool
}

// ProtoField 协议字段
type ProtoField struct {
	Name     string
	Type     int32
	Number   int32
	Repeated bool
	Optional bool
}
