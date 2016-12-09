/*
 * Copyright (c) 2016, Shinya Yagyu
 * All rights reserved.
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 * 3. Neither the name of the copyright holder nor the names of its
 *    contributors may be used to endorse or promote products derived from this
 *    software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

package packer

import (
	"bytes"
	"log"
	"testing"
)

type S3 struct {
	B byte
}

type S1 struct {
	I64 uint64
	I32 uint32
	B   byte
	VB1 []byte `len:"5"`
	I16 uint16
	VB2 []byte `len:"lastnum"`
	VI1 VarInt
	VI2 VarInt
	VI3 VarInt
	VI4 VarInt
	VS  string
	ST  S3
	STA []S3 `len:"2"`
	STB []S3 `len:"prefix"`
}

var s1 = S1{
	12345,
	12345,
	byte(0x12),
	[]byte{0x12, 0x34, 0x45, 0x00, 0x00},
	6,
	[]byte{0x67, 0x89, 0xab, 0xcd, 0x00, 0x00},
	VarInt(0x23),
	VarInt(0x1234),
	VarInt(0x12345),
	VarInt(0x123456789),
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
	0x39, 0x30, 0, 0, 0, 0, 0, 0, //8
	0x39, 0x30, 0, 0, //4
	0x12,                         //1
	0x12, 0x34, 0x45, 0x00, 0x00, //5
	0x06, 0x00, //2
	0x67, 0x89, 0xab, 0xcd, 0x00, 0x00, //6
	0x23,             //1
	0xfd, 0x34, 0x12, //3
	0xfe, 0x45, 0x23, 0x01, 0x00, //5
	0xff, 0x89, 0x67, 0x45, 0x23, 0x01, 0x00, 0x00, 0x00, //9
	0x05,                         //1
	0x61, 0x62, 0x63, 0x64, 0x65, //5
	0x12,             //1
	0x34,             //1
	0x35,             //1
	0x02, 0x21, 0x22, //3
}

func TestPack(t *testing.T) {
	var buf bytes.Buffer
	err := Pack(&buf, s1)
	if err != nil {
		t.Fatal(err)
	}
	b1 := buf.Bytes()
	log.Println(b1)
	log.Println(result)
	if !bytes.Equal(b1, result) {
		t.Fatal("not match")
	}
}

func TestUnpack1(t *testing.T) {
	s2 := S1{}
	r := bytes.NewBuffer(result)
	err := Unpack(r, &s2)
	if err != nil {
		t.Fatal(err)
	}
	if s1.I64 != s2.I64 {
		t.Fatal("I64 not match")
	}
	if s1.I32 != s2.I32 {
		t.Fatal("I32 not match")
	}
	if s1.I16 != s2.I16 {
		t.Fatal("I16 not match")
	}
	if s1.B != s2.B {
		t.Fatal("B not match")
	}
	if !bytes.Equal(s1.VB1, s2.VB1) {
		t.Fatal("VB1 not match")
	}
	if !bytes.Equal(s1.VB2, s2.VB2) {
		t.Fatal("VB2 not match")
	}
	if s1.VI1 != s2.VI1 {
		t.Fatal("VI1 not match")
	}
	if s1.VI2 != s2.VI2 {
		t.Fatal("VI2 not match")
	}
	if s1.VI3 != s2.VI3 {
		t.Fatal("VI3 not match")
	}
	if s1.VI4 != s2.VI4 {
		t.Fatal("VI4 not match")
	}
	if s1.VS != s2.VS {
		t.Fatal("VS.Str not match")
	}
	if s1.ST.B != s2.ST.B {
		t.Fatal("ST.B not match")
	}
	if s1.STA[0].B != s2.STA[0].B {
		t.Fatal("STA[0].B not match")
	}
	if s1.STA[1].B != s2.STA[1].B {
		t.Fatal("STA[1].B not match")
	}
	if len(s1.STB) != 2 {
		t.Fatal("len(STB) not match")
	}
	if s1.STB[0].B != s2.STB[0].B {
		t.Fatal("STB[0].B not match")
	}
	if s1.STB[1].B != s2.STB[1].B {
		t.Fatal("STB[1].B not match")
	}
}
