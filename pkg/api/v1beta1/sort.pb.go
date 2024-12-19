// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: sqs/query/v1beta1/sort.proto

package v1beta1

import (
	fmt "fmt"
	proto "github.com/cosmos/gogoproto/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// SortDirection represents the direction for sorting.
type SortDirection int32

const (
	SortDirection_ASCENDING  SortDirection = 0
	SortDirection_DESCENDING SortDirection = 1
)

var SortDirection_name = map[int32]string{
	0: "ASCENDING",
	1: "DESCENDING",
}

var SortDirection_value = map[string]int32{
	"ASCENDING":  0,
	"DESCENDING": 1,
}

func (x SortDirection) String() string {
	return proto.EnumName(SortDirection_name, int32(x))
}

func (SortDirection) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_2084d9d808812ac7, []int{0}
}

// SortField represents a field to sort by, including its direction.
type SortField struct {
	// field is the name of the field to sort by.
	Field string `protobuf:"bytes,1,opt,name=field,proto3" json:"field,omitempty"`
	// direction is a sorting direction: ASCENDING or DESCENDING.
	Direction SortDirection `protobuf:"varint,2,opt,name=direction,proto3,enum=sqs.query.v1beta1.SortDirection" json:"direction,omitempty"`
}

func (m *SortField) Reset()         { *m = SortField{} }
func (m *SortField) String() string { return proto.CompactTextString(m) }
func (*SortField) ProtoMessage()    {}
func (*SortField) Descriptor() ([]byte, []int) {
	return fileDescriptor_2084d9d808812ac7, []int{0}
}
func (m *SortField) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *SortField) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_SortField.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *SortField) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SortField.Merge(m, src)
}
func (m *SortField) XXX_Size() int {
	return m.Size()
}
func (m *SortField) XXX_DiscardUnknown() {
	xxx_messageInfo_SortField.DiscardUnknown(m)
}

var xxx_messageInfo_SortField proto.InternalMessageInfo

func (m *SortField) GetField() string {
	if m != nil {
		return m.Field
	}
	return ""
}

func (m *SortField) GetDirection() SortDirection {
	if m != nil {
		return m.Direction
	}
	return SortDirection_ASCENDING
}

// SortRequest allows sorting by multiple fields with specified precedence.
// The sort is applied in the order fields are specified - the first field
// is the primary sort key, the second field is used for ties, and so on.
// Example:
//
//	{
//	  "fields": [
//	    {"field": "liquidity", "direction": "DESCENDING"},
//	    {"field": "volume_24h", "direction": "ASCENDING"}
//	  ]
//	}
type SortRequest struct {
	// fields represents list of fields to sort by.
	// The order of fields determines the sort precedence.
	Fields []*SortField `protobuf:"bytes,1,rep,name=fields,proto3" json:"fields,omitempty"`
}

func (m *SortRequest) Reset()         { *m = SortRequest{} }
func (m *SortRequest) String() string { return proto.CompactTextString(m) }
func (*SortRequest) ProtoMessage()    {}
func (*SortRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2084d9d808812ac7, []int{1}
}
func (m *SortRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *SortRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_SortRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *SortRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SortRequest.Merge(m, src)
}
func (m *SortRequest) XXX_Size() int {
	return m.Size()
}
func (m *SortRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_SortRequest.DiscardUnknown(m)
}

var xxx_messageInfo_SortRequest proto.InternalMessageInfo

func (m *SortRequest) GetFields() []*SortField {
	if m != nil {
		return m.Fields
	}
	return nil
}

func init() {
	proto.RegisterEnum("sqs.query.v1beta1.SortDirection", SortDirection_name, SortDirection_value)
	proto.RegisterType((*SortField)(nil), "sqs.query.v1beta1.SortField")
	proto.RegisterType((*SortRequest)(nil), "sqs.query.v1beta1.SortRequest")
}

func init() { proto.RegisterFile("sqs/query/v1beta1/sort.proto", fileDescriptor_2084d9d808812ac7) }

var fileDescriptor_2084d9d808812ac7 = []byte{
	// 263 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x29, 0x2e, 0x2c, 0xd6,
	0x2f, 0x2c, 0x4d, 0x2d, 0xaa, 0xd4, 0x2f, 0x33, 0x4c, 0x4a, 0x2d, 0x49, 0x34, 0xd4, 0x2f, 0xce,
	0x2f, 0x2a, 0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12, 0x2c, 0x2e, 0x2c, 0xd6, 0x03, 0xcb,
	0xea, 0x41, 0x65, 0x95, 0x12, 0xb9, 0x38, 0x83, 0xf3, 0x8b, 0x4a, 0xdc, 0x32, 0x53, 0x73, 0x52,
	0x84, 0x44, 0xb8, 0x58, 0xd3, 0x40, 0x0c, 0x09, 0x46, 0x05, 0x46, 0x0d, 0xce, 0x20, 0x08, 0x47,
	0xc8, 0x8e, 0x8b, 0x33, 0x25, 0xb3, 0x28, 0x35, 0xb9, 0x24, 0x33, 0x3f, 0x4f, 0x82, 0x49, 0x81,
	0x51, 0x83, 0xcf, 0x48, 0x41, 0x0f, 0xc3, 0x24, 0x3d, 0x90, 0x31, 0x2e, 0x30, 0x75, 0x41, 0x08,
	0x2d, 0x4a, 0xce, 0x5c, 0xdc, 0x20, 0xb9, 0xa0, 0xd4, 0xc2, 0xd2, 0xd4, 0xe2, 0x12, 0x21, 0x13,
	0x2e, 0x36, 0xb0, 0xb9, 0xc5, 0x12, 0x8c, 0x0a, 0xcc, 0x1a, 0xdc, 0x46, 0x32, 0x38, 0xcc, 0x02,
	0x3b, 0x29, 0x08, 0xaa, 0x56, 0x4b, 0x8f, 0x8b, 0x17, 0xc5, 0x02, 0x21, 0x5e, 0x2e, 0x4e, 0xc7,
	0x60, 0x67, 0x57, 0x3f, 0x17, 0x4f, 0x3f, 0x77, 0x01, 0x06, 0x21, 0x3e, 0x2e, 0x2e, 0x17, 0x57,
	0x38, 0x9f, 0xd1, 0xc9, 0xf5, 0xc4, 0x23, 0x39, 0xc6, 0x0b, 0x8f, 0xe4, 0x18, 0x1f, 0x3c, 0x92,
	0x63, 0x9c, 0xf0, 0x58, 0x8e, 0xe1, 0xc2, 0x63, 0x39, 0x86, 0x1b, 0x8f, 0xe5, 0x18, 0xa2, 0xb4,
	0xd3, 0x33, 0x4b, 0x32, 0x4a, 0x93, 0xf4, 0x92, 0xf3, 0x73, 0xf5, 0xf3, 0x8b, 0x73, 0xf3, 0x8b,
	0x33, 0x8b, 0x75, 0x73, 0x12, 0x93, 0x8a, 0xf5, 0x41, 0x41, 0x57, 0x90, 0x9d, 0xae, 0x9f, 0x58,
	0x90, 0x09, 0x0b, 0xbc, 0x24, 0x36, 0x70, 0xc0, 0x19, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0xbc,
	0xcc, 0x10, 0x4f, 0x58, 0x01, 0x00, 0x00,
}

func (m *SortField) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *SortField) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SortField) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Direction != 0 {
		i = encodeVarintSort(dAtA, i, uint64(m.Direction))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Field) > 0 {
		i -= len(m.Field)
		copy(dAtA[i:], m.Field)
		i = encodeVarintSort(dAtA, i, uint64(len(m.Field)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *SortRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *SortRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SortRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Fields) > 0 {
		for iNdEx := len(m.Fields) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Fields[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintSort(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintSort(dAtA []byte, offset int, v uint64) int {
	offset -= sovSort(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *SortField) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Field)
	if l > 0 {
		n += 1 + l + sovSort(uint64(l))
	}
	if m.Direction != 0 {
		n += 1 + sovSort(uint64(m.Direction))
	}
	return n
}

func (m *SortRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Fields) > 0 {
		for _, e := range m.Fields {
			l = e.Size()
			n += 1 + l + sovSort(uint64(l))
		}
	}
	return n
}

func sovSort(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozSort(x uint64) (n int) {
	return sovSort(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *SortField) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSort
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: SortField: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: SortField: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Field", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSort
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthSort
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSort
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Field = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Direction", wireType)
			}
			m.Direction = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSort
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Direction |= SortDirection(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipSort(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSort
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
func (m *SortRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSort
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: SortRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: SortRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Fields", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSort
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSort
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSort
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Fields = append(m.Fields, &SortField{})
			if err := m.Fields[len(m.Fields)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSort(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSort
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
func skipSort(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowSort
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
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
					return 0, ErrIntOverflowSort
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSort
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthSort
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupSort
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthSort
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthSort        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowSort          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupSort = fmt.Errorf("proto: unexpected end of group")
)
