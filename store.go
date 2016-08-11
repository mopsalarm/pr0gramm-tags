package main

import (
	"encoding/binary"
)

type ByteStore interface {
	Push(key uint32, value byte)
	PushN(key uint32, value []byte)

	Contains(key uint32) bool
	Get(key uint32) []byte

	Remove(key uint32)
	Compact(key uint32)
	Clear(key uint32)

	KeyCount() uint32
	Keys() []uint32

	MemorySize() uint32
}

type IterStore interface {
	ByteStore

	GetIterator(key uint32) *ItemIterator
	PushInt(key uint32, values int32)
	PushInts(key uint32, values []int32)
}

type VarintStore struct {
	ByteStore
}

func (store *VarintStore) PushInt(key uint32, value int32) {
	var scratch [binary.MaxVarintLen32]byte
	n := binary.PutVarint(scratch[:], int64(value))
	store.PushN(key, scratch[:n])
}

func (store *VarintStore) PushInts(key uint32, values []int32) {
	written := 0
	scratch := make([]byte, len(values) * binary.MaxVarintLen32)

	previous := int32(0)
	for _, value := range values {
		n := binary.PutVarint(scratch[written:], int64(value - previous))
		written += n
		previous = value
	}

	if written > 0 {
		store.PushN(key, scratch[:written])
	}
}

func (store *VarintStore) GetIterator(key uint32) *ItemIterator {
	return NewSequenceIterator(store.Get(key))
}

func int24ToBytes(value int32) [3]byte {
	var bytes [3]byte
	bytes[0] = byte((value >> 16) & 0xff)
	bytes[1] = byte((value >> 8) & 0xff)
	bytes[2] = byte(value & 0xff)
	return bytes
}

func bytesToInt24(bytes [3]byte) int32 {
	val := (int32(bytes[0]) << 16) | (int32(bytes[1]) << 8) | int32(bytes[2])
	if bytes[0] & 0x80 != 0 {
		return -1 << 24 | val
	}
	return val
}
//
//type UncompressedStore struct {
//	ByteStore
//}
//
//func (store *UncompressedStore) PushInt(key uint32, value int32) {
//	bytes := int24ToBytes(value)
//	store.PushN(key, bytes[:])
//}
//
//func (store *UncompressedStore) PushInts(key uint32, values []int32) {
//	scratch := make([]byte, len(values) * 3)
//	for idx, value := range values {
//		bytes := int24ToBytes(value)
//		copy(scratch[3 * idx:3 * idx + 3], bytes[:])
//	}
//
//	store.PushN(key, scratch[:])
//}
//
//func (store *UncompressedStore) GetIterator(key uint32) *ItemIterator {
//	bytes := store.Get(key)
//	val := &int24iterator{bytes: bytes, pos: -3}
//	val.Next()
//	return val
//}
//
//type int24iterator struct {
//	pos   int
//	bytes []byte
//	next  int32
//}
//
//func (it *int24iterator) HasMore() bool {
//	return it.pos < len(it.bytes)
//}
//
//func (it *int24iterator) Peek() int32 {
//	return it.next
//}
//
//func (it *int24iterator) Next() int32 {
//	value := it.next
//
//	it.pos += 3
//	if it.HasMore() {
//		scratch := (*[3]byte)(unsafe.Pointer(&it.bytes[it.pos]))
//		it.next = bytesToInt24(*scratch)
//	}
//
//	return value
//}
//
//func (it *int24iterator) SkipUntil(val int32) {
//	for it.HasMore() && it.Peek() < val {
//		it.Next()
//	}
//}

