package types

// 1字节重定义类型
type ByteFlag uint8
type BoolFlag bool
type StatusByte uint8

// 2字节重定义类型
type WordSize uint16
type PortNumber uint16
type FlagWord int16

// 4字节重定义类型
type DWordSize uint32
type TimeoutMs uint32
type Counter32 int32
type Float32Value float32

// 8字节重定义类型
type QWordSize uint64
type Timestamp int64
type Counter64 int64
type IDType uint64
type Float64Value float64

// 16字节重定义类型
type NameString string
type URLString string
type KeyString string
type HashString string

// 复杂重定义类型
type ByteSlice []byte
type StringSlice []string
type Int64Slice []int64
type StringMap map[string]string
type InterfaceMap map[string]interface{}
