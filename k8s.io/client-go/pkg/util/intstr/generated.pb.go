/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by protoc-gen-gogo.
// source: k8s.io/kubernetes/pkg/util/intstr/generated.proto
// DO NOT EDIT!

/*
	Package intstr is a generated protocol buffer package.

	It is generated from these files:
		k8s.io/kubernetes/pkg/util/intstr/generated.proto

	It has these top-level messages:
		IntOrString
*/
package intstr

import proto "github.com/openshift/github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.GoGoProtoPackageIsVersion1

func (m *IntOrString) Reset()                    { *m = IntOrString{} }
func (*IntOrString) ProtoMessage()               {}
func (*IntOrString) Descriptor() ([]byte, []int) { return fileDescriptorGenerated, []int{0} }

func init() {
	proto.RegisterType((*IntOrString)(nil), "k8s.io.client-go.pkg.util.intstr.IntOrString")
}
func (m *IntOrString) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *IntOrString) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0x8
	i++
	i = encodeVarintGenerated(data, i, uint64(m.Type))
	data[i] = 0x10
	i++
	i = encodeVarintGenerated(data, i, uint64(m.IntVal))
	data[i] = 0x1a
	i++
	i = encodeVarintGenerated(data, i, uint64(len(m.StrVal)))
	i += copy(data[i:], m.StrVal)
	return i, nil
}

func encodeFixed64Generated(data []byte, offset int, v uint64) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	data[offset+4] = uint8(v >> 32)
	data[offset+5] = uint8(v >> 40)
	data[offset+6] = uint8(v >> 48)
	data[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Generated(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintGenerated(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func (m *IntOrString) Size() (n int) {
	var l int
	_ = l
	n += 1 + sovGenerated(uint64(m.Type))
	n += 1 + sovGenerated(uint64(m.IntVal))
	l = len(m.StrVal)
	n += 1 + l + sovGenerated(uint64(l))
	return n
}

func sovGenerated(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozGenerated(x uint64) (n int) {
	return sovGenerated(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *IntOrString) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: IntOrString: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: IntOrString: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			m.Type = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.Type |= (Type(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field IntVal", wireType)
			}
			m.IntVal = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.IntVal |= (int32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field StrVal", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.StrVal = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthGenerated
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipGenerated(data []byte) (n int, err error) {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if data[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthGenerated
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowGenerated
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := data[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipGenerated(data[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthGenerated = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGenerated   = fmt.Errorf("proto: integer overflow")
)

var fileDescriptorGenerated = []byte{
	// 269 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x44, 0x8f, 0x31, 0x4e, 0xc3, 0x30,
	0x18, 0x85, 0x6d, 0x5a, 0x2a, 0x08, 0x12, 0x43, 0xc4, 0x50, 0x31, 0x38, 0x81, 0x01, 0x79, 0xc1,
	0x16, 0x1b, 0x62, 0xcc, 0xd6, 0x09, 0x29, 0x45, 0x0c, 0x6c, 0x0d, 0x18, 0x63, 0xa5, 0xd8, 0x96,
	0xf3, 0x67, 0xe8, 0xd6, 0x23, 0xc0, 0xc6, 0xc8, 0x71, 0x32, 0x76, 0x64, 0x40, 0x15, 0x31, 0xb7,
	0x60, 0x42, 0x71, 0x22, 0x75, 0xb2, 0xdf, 0x7b, 0xdf, 0x67, 0xc9, 0xd1, 0x55, 0x79, 0x5d, 0x31,
	0x65, 0x78, 0x59, 0x17, 0xc2, 0x69, 0x01, 0xa2, 0xe2, 0xb6, 0x94, 0xbc, 0x06, 0xb5, 0xe4, 0x4a,
	0x43, 0x05, 0x8e, 0x4b, 0xa1, 0x85, 0x5b, 0x80, 0x78, 0x62, 0xd6, 0x19, 0x30, 0xf1, 0x59, 0xaf,
	0xb0, 0x9d, 0xc2, 0x6c, 0x29, 0x59, 0xa7, 0xb0, 0x5e, 0x39, 0xbd, 0x94, 0x0a, 0x5e, 0xea, 0x82,
	0x3d, 0x9a, 0x57, 0x2e, 0x8d, 0x34, 0x3c, 0x98, 0x45, 0xfd, 0x1c, 0x52, 0x08, 0xe1, 0xd6, 0xbf,
	0x78, 0xfe, 0x8e, 0xa3, 0xa3, 0x99, 0x86, 0x5b, 0x37, 0x07, 0xa7, 0xb4, 0x8c, 0x69, 0x34, 0x86,
	0x95, 0x15, 0x53, 0x9c, 0x62, 0x3a, 0xca, 0x4e, 0x9a, 0x6d, 0x82, 0xfc, 0x36, 0x19, 0xdf, 0xad,
	0xac, 0xf8, 0x1b, 0xce, 0x3c, 0x10, 0xf1, 0x45, 0x34, 0x51, 0x1a, 0xee, 0x17, 0xcb, 0xe9, 0x5e,
	0x8a, 0xe9, 0x7e, 0x76, 0x3c, 0xb0, 0x93, 0x59, 0x68, 0xf3, 0x61, 0xed, 0xb8, 0x0a, 0x5c, 0xc7,
	0x8d, 0x52, 0x4c, 0x0f, 0x77, 0xdc, 0x3c, 0xb4, 0xf9, 0xb0, 0xde, 0x1c, 0x7c, 0x7c, 0x26, 0x68,
	0xfd, 0x9d, 0xa2, 0x8c, 0x36, 0x2d, 0x41, 0x9b, 0x96, 0xa0, 0xaf, 0x96, 0xa0, 0xb5, 0x27, 0xb8,
	0xf1, 0x04, 0x6f, 0x3c, 0xc1, 0x3f, 0x9e, 0xe0, 0xb7, 0x5f, 0x82, 0x1e, 0x26, 0xfd, 0x67, 0xff,
	0x03, 0x00, 0x00, 0xff, 0xff, 0x68, 0x57, 0xfb, 0xfa, 0x43, 0x01, 0x00, 0x00,
}
