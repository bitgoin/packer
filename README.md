[![GoDoc](https://godoc.org/github.com/utamaro/packer?status.svg)](https://godoc.org/github.com/utamaro/packer)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/utamaro/packer/LICENSE)


# packer 

## Overview

This  library is for packing/unpacking structs(i.e. serialize struct to binary) in classical protocol style mostly used in C
 (e.g. bitcoiin protocol).  

## Requirements

This requires

* git
* go 1.3+

# Supported types in Struct

* uint16/32/64
* byte
* byte slice
* [VarInt](https://en.bitcoin.it/wiki/Protocol_documentation#Variable_length_integer)
* string(automatically prefixed by length)
* struct slice

You must specify how size of slice is serialized to binary by struct tag named `len`.
* `len:"<number>"` : fixed size (e.g. `len:"5"`)
* `len:"lastnum"` : The size of slice is specified at last field in struct.
* `len:"prefix` : The size of slice is prefixed.

When you pack slices in struct and use "lastnum" , pack will NOT automatically adds the sizes of slices. You must
specify the size manually. If you use "prefix", the size is automatically prefixed. The length of fixed size
is not packed.

Length of non-slice types(e.g. size 8 of uint64 etc) is not packed. 

All integers are packed in little endian.

## Installation

     $ go get github.com/utamaro/packer


## Example
(This example omits error handlings for simplicity.)

```go

import packer

type S3 struct {
	B byte
}

type S1 struct {
	I64 uint64
	VB1 []byte `len:"5"`
	VI1 packer.VarInt
	VS  string
	ST  S3
	STA []S3 `len:"2"`
	STB []S3 `len:"prefix"`
}

var s1 = S1{
	12345,
	[]byte{0x12, 0x34, 0x45, 0x00, 0x00},
	packer.VarInt(6),
	"abcde",
	S3{
		0x12,
	},
	[]S3{
		{0x34}, {0x35},
	},
	[]S3{
		{0x21}, {0x22},
	},
}
var result = []byte{
	0x39, 0x30, 0, 0, 0, 0, 0, 0, //uint64(12345)
	0x12, 0x34, 0x45, 0x00, 0x00, //fixed size of slice, no length is packed.
	0x06, 0x00, //VarInt(6)
	0x05,                         //size of string(abcde)
	0x61, 0x62, 0x63, 0x64, 0x65, //string("abcde")
	0x12,             //struct S3{byte(0x12)}
	0x34,0x35         //fixed size of []S3
	0x02,  //size of []S3
	0x21, 0x22, //struct []S3
}

func main(){
	var buf bytes.Buffer
	err := Pack(&buf, s1)

	s2 := S1{}
	r := bytes.NewBuffer(result)
	err := Unpack(r, &s2)

}
```


# Contribution
Improvements to the codebase and pull requests are encouraged.


