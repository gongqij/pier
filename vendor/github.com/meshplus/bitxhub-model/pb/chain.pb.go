// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: chain.proto

package pb

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	github_com_meshplus_bitxhub_kit_types "github.com/meshplus/bitxhub-kit/types"
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

type ChainMeta struct {
	Height            uint64                                      `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	BlockHash         *github_com_meshplus_bitxhub_kit_types.Hash `protobuf:"bytes,2,opt,name=block_hash,json=blockHash,proto3,customtype=github.com/meshplus/bitxhub-kit/types.Hash" json:"block_hash,omitempty"`
	InterchainTxCount uint64                                      `protobuf:"varint,3,opt,name=interchain_tx_count,json=interchainTxCount,proto3" json:"interchain_tx_count,omitempty"`
}

func (m *ChainMeta) Reset()         { *m = ChainMeta{} }
func (m *ChainMeta) String() string { return proto.CompactTextString(m) }
func (*ChainMeta) ProtoMessage()    {}
func (*ChainMeta) Descriptor() ([]byte, []int) {
	return fileDescriptor_d4d91b2d037e7a44, []int{0}
}
func (m *ChainMeta) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ChainMeta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ChainMeta.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ChainMeta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChainMeta.Merge(m, src)
}
func (m *ChainMeta) XXX_Size() int {
	return m.Size()
}
func (m *ChainMeta) XXX_DiscardUnknown() {
	xxx_messageInfo_ChainMeta.DiscardUnknown(m)
}

var xxx_messageInfo_ChainMeta proto.InternalMessageInfo

func (m *ChainMeta) GetHeight() uint64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *ChainMeta) GetInterchainTxCount() uint64 {
	if m != nil {
		return m.InterchainTxCount
	}
	return 0
}

func init() {
	proto.RegisterType((*ChainMeta)(nil), "pb.ChainMeta")
}

func init() { proto.RegisterFile("chain.proto", fileDescriptor_d4d91b2d037e7a44) }

var fileDescriptor_d4d91b2d037e7a44 = []byte{
	// 232 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x4e, 0xce, 0x48, 0xcc,
	0xcc, 0xd3, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x2a, 0x48, 0x92, 0xd2, 0x4d, 0xcf, 0x2c,
	0xc9, 0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0x4f, 0xcf, 0x4f, 0xcf, 0xd7, 0x07, 0x4b, 0x25,
	0x95, 0xa6, 0x81, 0x79, 0x60, 0x0e, 0x98, 0x05, 0xd1, 0xa2, 0xb4, 0x88, 0x91, 0x8b, 0xd3, 0x19,
	0x64, 0x84, 0x6f, 0x6a, 0x49, 0xa2, 0x90, 0x18, 0x17, 0x5b, 0x46, 0x6a, 0x66, 0x7a, 0x46, 0x89,
	0x04, 0xa3, 0x02, 0xa3, 0x06, 0x4b, 0x10, 0x94, 0x27, 0xe4, 0xcb, 0xc5, 0x95, 0x94, 0x93, 0x9f,
	0x9c, 0x1d, 0x9f, 0x91, 0x58, 0x9c, 0x21, 0xc1, 0xa4, 0xc0, 0xa8, 0xc1, 0xe3, 0xa4, 0x77, 0xeb,
	0x9e, 0xbc, 0x16, 0x92, 0x65, 0xb9, 0xa9, 0xc5, 0x19, 0x05, 0x39, 0xa5, 0xc5, 0xfa, 0x49, 0x99,
	0x25, 0x15, 0x19, 0xa5, 0x49, 0xba, 0xd9, 0x99, 0x25, 0xfa, 0x25, 0x95, 0x05, 0xa9, 0xc5, 0x7a,
	0x1e, 0x89, 0xc5, 0x19, 0x41, 0x9c, 0x60, 0x13, 0x40, 0x4c, 0x21, 0x3d, 0x2e, 0xe1, 0xcc, 0xbc,
	0x92, 0xd4, 0x22, 0xb0, 0xdb, 0xe3, 0x4b, 0x2a, 0xe2, 0x93, 0xf3, 0x4b, 0xf3, 0x4a, 0x24, 0x98,
	0xc1, 0x76, 0x0a, 0x22, 0xa4, 0x42, 0x2a, 0x9c, 0x41, 0x12, 0x4e, 0x12, 0x27, 0x1e, 0xc9, 0x31,
	0x5e, 0x78, 0x24, 0xc7, 0xf8, 0xe0, 0x91, 0x1c, 0xe3, 0x84, 0xc7, 0x72, 0x0c, 0x17, 0x1e, 0xcb,
	0x31, 0xdc, 0x78, 0x2c, 0xc7, 0x90, 0xc4, 0x06, 0xf6, 0x85, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff,
	0x6a, 0xd9, 0x1d, 0xc9, 0x07, 0x01, 0x00, 0x00,
}

func (m *ChainMeta) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ChainMeta) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ChainMeta) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.InterchainTxCount != 0 {
		i = encodeVarintChain(dAtA, i, uint64(m.InterchainTxCount))
		i--
		dAtA[i] = 0x18
	}
	if m.BlockHash != nil {
		{
			size := m.BlockHash.Size()
			i -= size
			if _, err := m.BlockHash.MarshalTo(dAtA[i:]); err != nil {
				return 0, err
			}
			i = encodeVarintChain(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if m.Height != 0 {
		i = encodeVarintChain(dAtA, i, uint64(m.Height))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintChain(dAtA []byte, offset int, v uint64) int {
	offset -= sovChain(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *ChainMeta) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Height != 0 {
		n += 1 + sovChain(uint64(m.Height))
	}
	if m.BlockHash != nil {
		l = m.BlockHash.Size()
		n += 1 + l + sovChain(uint64(l))
	}
	if m.InterchainTxCount != 0 {
		n += 1 + sovChain(uint64(m.InterchainTxCount))
	}
	return n
}

func sovChain(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozChain(x uint64) (n int) {
	return sovChain(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *ChainMeta) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowChain
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
			return fmt.Errorf("proto: ChainMeta: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ChainMeta: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Height", wireType)
			}
			m.Height = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowChain
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Height |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BlockHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowChain
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthChain
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthChain
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var v github_com_meshplus_bitxhub_kit_types.Hash
			m.BlockHash = &v
			if err := m.BlockHash.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field InterchainTxCount", wireType)
			}
			m.InterchainTxCount = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowChain
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.InterchainTxCount |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipChain(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthChain
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthChain
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
func skipChain(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowChain
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
					return 0, ErrIntOverflowChain
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
					return 0, ErrIntOverflowChain
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
				return 0, ErrInvalidLengthChain
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupChain
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthChain
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthChain        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowChain          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupChain = fmt.Errorf("proto: unexpected end of group")
)