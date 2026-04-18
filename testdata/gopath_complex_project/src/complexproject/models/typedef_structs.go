package models

import "complexproject/types"

// ComplexTypeStruct 包含多种重定义类型的复杂结构体
type ComplexTypeStruct struct {
	ID        types.IDType
	Status    types.ByteFlag
	Port      types.PortNumber
	Timeout   types.TimeoutMs
	Timestamp types.Timestamp
	Count     types.Counter64
	Name      types.NameString
	URL       types.URLString
	Data      types.ByteSlice
	Tags      types.StringSlice
	Metadata  types.StringMap
	Enabled   types.BoolFlag
}

// AnotherTypeStruct 另一个包含重定义类型的结构体
type AnotherTypeStruct struct {
	Code   types.WordSize
	Flag   types.FlagWord
	Value  types.Float32Value
	Price  types.Float64Value
	Hash   types.HashString
	Key    types.KeyString
	IDs    types.Int64Slice
	Config types.InterfaceMap
}
