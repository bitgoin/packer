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
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
)

//VarInt is variable integer in bitcoin protocol.
type VarInt uint64

const (
	prefixTag  = "prefix"
	lastnumTag = "lastnum"
)

//Pack converts struct to []byte.
func Pack(buf io.Writer, t interface{}) error {
	v := reflect.ValueOf(t)
	v = reflect.Indirect(v)
	ty := v.Type()
	if ty.Kind() != reflect.Struct {
		return errors.New("must be struct")
	}
	var val uint64
	for i := 0; i < ty.NumField(); i++ {
		f := v.Field(i)
		intf := f.Interface()
		var result []byte
		switch dat := intf.(type) {
		case uint64:
			result = make([]byte, 8)
			binary.LittleEndian.PutUint64(result, dat)
			val = dat
		case uint32:
			result = make([]byte, 4)
			binary.LittleEndian.PutUint32(result, dat)
			val = uint64(dat)
		case uint16:
			result = make([]byte, 2)
			binary.LittleEndian.PutUint16(result, dat)
			val = uint64(dat)
		case byte:
			result = []byte{dat}
			val = uint64(dat)
		case bool:
			if dat {
				result = []byte{1}
			} else {
				result = []byte{0}
			}
		case []byte:
			var err error
			tag := ty.Field(i).Tag
			result, err = makeBytesFromTagForPack(tag, val, dat)
			if err != nil {
				return err
			}
		case VarInt:
			result = int2varint(uint64(dat))
			val = uint64(dat)
		case string:
			if _, err := buf.Write(int2varint(uint64(len(dat)))); err != nil {
				return err
			}
			result = []byte(dat)
		default:
			switch f.Kind() {
			case reflect.Struct:
				if err := Pack(buf, dat); err != nil {
					return err
				}
			case reflect.Slice:
				if ty.Field(i).Tag.Get("len") == prefixTag {
					if _, err := buf.Write(int2varint(uint64(f.Len()))); err != nil {
						return err
					}
				}
				for ii := 0; ii < f.Len(); ii++ {
					if err := Pack(buf, f.Index(ii).Interface()); err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("cannot conver field no %d", i)
			}
		}
		if result == nil {
			continue
		}
		if _, err := buf.Write(result); err != nil {
			return err
		}
	}
	return nil
}

//Unpack converts []byte to struct.
func Unpack(buf io.Reader, t interface{}) error {
	v := reflect.ValueOf(t).Elem()
	ty := v.Type()
	if ty.Kind() != reflect.Struct {
		return errors.New("must be struct")
	}
	var val uint64
	for i := 0; i < ty.NumField(); i++ {
		vi := v.Field(i)
		if !(vi.IsValid() && vi.CanSet()) {
			return fmt.Errorf("not valid or cannot set at field no %d", i)
		}
		intf := vi.Interface()
		var result interface{}
		switch intf.(type) {
		case uint64:
			b := make([]byte, 8)
			if _, err := io.ReadFull(buf, b); err != nil {
				return err
			}
			dat := binary.LittleEndian.Uint64(b)
			val = dat
			result = dat
		case uint32:
			b := make([]byte, 4)
			if _, err := io.ReadFull(buf, b); err != nil {
				return err
			}
			dat := binary.LittleEndian.Uint32(b)
			result = dat
			val = uint64(dat)
		case uint16:
			b := make([]byte, 2)
			if _, err := io.ReadFull(buf, b); err != nil {
				return err
			}
			dat := binary.LittleEndian.Uint16(b)
			result = dat
			val = uint64(dat)
		case bool:
			bs := make([]byte, 1)
			if _, err := io.ReadFull(buf, bs); err != nil {
				return err
			}
			if bs[0] == 0 {
				result = false
			} else {
				result = true
			}
		case byte:
			bs := make([]byte, 1)
			if _, err := io.ReadFull(buf, bs); err != nil {
				return err
			}
			result = bs[0]
			val = uint64(bs[0])
		case VarInt:
			dat, err := byte2varint(buf)
			if err != nil {
				return err
			}
			result = VarInt(dat)
			val = dat
		case string:
			siz, err := byte2varint(buf)
			if err != nil {
				return err
			}
			b := make([]byte, siz)
			if _, err := io.ReadFull(buf, b); err != nil {
				return err
			}
			result = string(b)
		case []byte:
			var err error
			tag := ty.Field(i).Tag
			dat, err := makeBytesFromTag(tag, val, buf)
			if err != nil {
				return err
			}
			if _, err := io.ReadFull(buf, dat); err != nil {
				return err
			}
			result = dat
		default:
			switch vi.Kind() {
			case reflect.Struct:
				if err := Unpack(buf, vi.Addr().Interface()); err != nil {
					return err
				}
			case reflect.Slice:
				tag := ty.Field(i).Tag
				newv, err := makeValueFromTag(tag, val, vi, buf)
				if err != nil {
					return err
				}
				for ii := 0; ii < newv.Len(); ii++ {
					if err := Unpack(buf, newv.Index(ii).Addr().Interface()); err != nil {
						return err
					}
				}
				vi.Set(*newv)
			default:
				return fmt.Errorf("cannot conver field no %d", i)
			}
		}
		if result != nil {
			vi.Set(reflect.ValueOf(result))
		}
	}
	return nil
}

func getTagValue(tag reflect.StructTag, val uint64, buf io.Reader) (int, error) {
	le := tag.Get("len")
	var n int
	var err error
	switch le {
	case lastnumTag:
		n = int(val)
	case prefixTag:
		val, err = byte2varint(buf)
		if err != nil {
			return 0, err
		}
		n = int(val)
	case "":
		err = fmt.Errorf("no tag for slice")
	default:
		n, err = strconv.Atoi(le)
	}
	return n, err
}

func makeBytesFromTag(tag reflect.StructTag, val uint64, buf io.Reader) ([]byte, error) {
	n, err := getTagValue(tag, val, buf)
	if err != nil {
		return nil, err
	}
	d := make([]byte, n)
	return d, nil
}
func makeBytesFromTagForPack(tag reflect.StructTag, val uint64, dat []byte) ([]byte, error) {
	le := tag.Get("len")
	switch le {
	case lastnumTag:
		b := make([]byte, val)
		copy(b, dat)
		return b, nil
	case prefixTag:
		ll := int2varint(uint64(len(dat)))
		b := make([]byte, len(dat)+len(ll))
		copy(b, ll)
		copy(b[len(ll):], dat)
		return b, nil
	case "":
		return nil, fmt.Errorf("no tag for slice")
	default:
		n, err := strconv.Atoi(le)
		if err != nil {
			return nil, err
		}
		b := make([]byte, n)
		copy(b, dat)
		return b, nil
	}
}

func makeValueFromTag(tag reflect.StructTag, val uint64,
	vi reflect.Value, buf io.Reader) (*reflect.Value, error) {
	n, err := getTagValue(tag, val, buf)
	if err != nil {
		return nil, err
	}
	newv := reflect.MakeSlice(vi.Type(), n, n)
	return &newv, nil
}

//byte2varint converts []byte to varsint uint64.
func byte2varint(dat io.Reader) (uint64, error) {
	bb := make([]byte, 1)
	if _, err := io.ReadFull(dat, bb); err != nil {
		return 0, err
	}
	switch bb[0] {
	case 0xfd:
		bs := make([]byte, 2)
		_, err := io.ReadFull(dat, bs)
		if err != nil {
			return 0, err
		}
		v := binary.LittleEndian.Uint16(bs)
		return uint64(v), nil
	case 0xfe:
		bs := make([]byte, 4)
		_, err := io.ReadFull(dat, bs)
		if err != nil {
			return 0, err
		}
		v := binary.LittleEndian.Uint32(bs)
		return uint64(v), nil
	case 0xff:
		bs := make([]byte, 8)
		_, err := io.ReadFull(dat, bs)
		if err != nil {
			return 0, err
		}
		v := binary.LittleEndian.Uint64(bs)
		return v, nil

	default:
		return uint64(bb[0]), nil
	}
}

//int2varint converts varsint uint64 to []byte.
func int2varint(dat uint64) []byte {
	var b []byte
	switch {
	case dat < uint64(0xfd):
		b = make([]byte, 1)
		b[0] = byte(dat & 0xff)
	case dat <= uint64(0xffff):
		b = make([]byte, 3)
		b[0] = 0xfd
		binary.LittleEndian.PutUint16(b[1:], uint16(dat))
	case dat <= uint64(0xffffffff):
		b = make([]byte, 5)
		b[0] = 0xfe
		binary.LittleEndian.PutUint32(b[1:], uint32(dat))
	default:
		b = make([]byte, 9)
		b[0] = 0xff
		binary.LittleEndian.PutUint64(b[1:], dat)
	}
	return b
}
