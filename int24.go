package main

import "unsafe"

func int24ToBytes(value int32) [3]byte {
	var bytes [3]byte
	bytes[0] = byte((value >> 16) & 0xff)
	bytes[1] = byte((value >> 8) & 0xff)
	bytes[2] = byte(value & 0xff)
	return bytes
}

func bytesToInt24(bytes [3]byte) int32 {
	val := (int32(bytes[0]) << 16) | (int32(bytes[1]) << 8) | int32(bytes[2])
	if bytes[0]&0x80 != 0 {
		return -1<<24 | val
	}
	return val
}

type int24iterator struct {
	pos      int
	length   int
	bytesPtr uintptr
	next     int32
}

func NewInt24Iterator(bytes []byte) ItemIterator {
	it := &int24iterator{
		length:   len(bytes),
		pos:      -3,
		bytesPtr: uintptr(unsafe.Pointer(&bytes[0])),
	}

	it.Next()
	return it
}

func (it *int24iterator) HasMore() bool {
	return it.pos < it.length
}

func (it *int24iterator) Peek() int32 {
	return it.next
}

func (it *int24iterator) Next() int32 {
	value := it.next

	it.pos += 3
	if it.pos < it.length {
		scratch := (*[3]byte)(unsafe.Pointer(it.bytesPtr + uintptr(it.pos)))
		it.next = bytesToInt24(*scratch)
	}

	return value
}

func (it *int24iterator) SkipUntil(val int32) {
	for it.HasMore() && it.Peek() < val {
		it.Next()
	}
}
