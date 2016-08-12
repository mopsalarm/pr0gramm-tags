package store

import (
	"encoding/binary"
	"errors"
	"unsafe"
)

type SequenceCodec interface {
	Id() byte
	CanAppend() bool
	Decode(bytes []byte) ItemIterator
	Encode(values []int32) []byte
}

type int24Codec struct{}

func (*int24Codec) Id() byte {
	return 1
}

func (*int24Codec) CanAppend() bool {
	return true
}

func (*int24Codec) Decode(bytes []byte) ItemIterator {
	return NewInt24Iterator(bytes)
}

func (*int24Codec) Encode(values []int32) []byte {
	scratch := make([]byte, len(values)*3)
	for idx, value := range values {
		bytes := int24ToBytes(value)

		o := 3 * idx
		scratch[o] = bytes[0]
		scratch[o+1] = bytes[1]
		scratch[o+2] = bytes[2]
	}

	return scratch
}

type varintCodec struct{}

func (*varintCodec) Id() byte {
	return 2
}

func (*varintCodec) CanAppend() bool {
	return false
}

func (*varintCodec) Decode(bytes []byte) ItemIterator {
	return NewDecompressingIterator(bytes)
}

func (*varintCodec) Encode(values []int32) []byte {
	scratch := make([]byte, len(values)*binary.MaxVarintLen32)

	var written int
	previous := int32(0)
	for _, value := range values {
		n := binary.PutVarint(scratch[written:], int64(value-previous))
		written += n
		previous = value
	}

	return scratch[:written]
}

type int32Codec struct{}

func (*int32Codec) Id() byte {
	return 4
}

func (*int32Codec) CanAppend() bool {
	return false
}

func (*int32Codec) Decode(bytes []byte) ItemIterator {
	return NewInt32Iterator(bytes)
}

func (*int32Codec) Encode(values []int32) []byte {
	byteCount := 4 * len(values)
	return (*[1 << 24]byte)(unsafe.Pointer(&values[0]))[:byteCount:byteCount]
}

var int24CodecInstance = &int24Codec{}
var varintCodecInstance = &varintCodec{}
var int32CodecInstance = &int32Codec{}

func SequenceCodecById(id byte) SequenceCodec {
	switch {
	case id == 1:
		return int24CodecInstance

	case id == 2:
		return varintCodecInstance

	case id == 4:
		return int32CodecInstance

	default:
		panic(errors.New("unknown codec"))
	}
}

func OptimalCodec(itemCount int) SequenceCodec {
	if itemCount > 100000 {
		return int32CodecInstance
	} else {
		return varintCodecInstance
	}
}
