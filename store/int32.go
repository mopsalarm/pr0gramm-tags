package store

import "unsafe"

type int32iterator struct {
	pos      int
	intCount int
	bytes    []byte
	intView  *[1 << 24]int32
}

func NewInt32Iterator(bytes []byte) ItemIterator {
	intCount := len(bytes) / 4
	it := &int32iterator{
		bytes:    bytes,
		intCount: intCount,
		intView:  (*[1 << 24]int32)(unsafe.Pointer(&bytes[0])),
	}

	it.Next()
	return it
}

func (it *int32iterator) HasMore() bool {
	return it.pos < it.intCount
}

func (it *int32iterator) Peek() int32 {
	return it.intView[it.pos]
}

func (it *int32iterator) Next() int32 {
	it.pos += 1
	return it.intView[it.pos-1]
}

func (it *int32iterator) SkipUntil(val int32) {
	for it.pos < it.intCount && it.intView[it.pos] < val {
		it.pos += 1
	}
}
