package store

import (
	"unsafe"
)

type int32byteIterator struct {
	pos      int
	intCount int
	bytes    []byte
	intView  *[1 << 24]int32
}

func NewInt32Iterator(bytes []byte) ItemIterator {
	intCount := len(bytes) / 4
	it := &int32byteIterator{
		bytes:    bytes,
		intCount: intCount,
		intView:  (*[1 << 24]int32)(unsafe.Pointer(&bytes[0])),
	}

	it.Next()
	return it
}

func (it *int32byteIterator) HasMore() bool {
	return it.pos < it.intCount
}

func (it *int32byteIterator) Peek() int32 {
	return it.intView[it.pos]
}

func (it *int32byteIterator) Next() int32 {
	it.pos += 1
	return it.intView[it.pos - 1]
}

func (it *int32byteIterator) SkipUntil(val int32) {
	for it.pos < it.intCount && it.intView[it.pos] < val {
		it.pos += 1
	}
}

func (it *int32byteIterator) MaxSize() int {
	return it.intCount
}

func (it *int32byteIterator) ToSlice() []int32 {
	slice := make([]int32, it.intCount)
	copy(slice, it.intView[:it.intCount])
	return slice
}

type sliceIterator struct {
	values   []int32
	position int
}

func NewSliceIterator(values []int32) ItemIterator {
	return &sliceIterator{values: values}
}

func (it *sliceIterator) HasMore() bool {
	return it.position < len(it.values)
}

func (it *sliceIterator) Peek() int32 {
	return it.values[it.position]
}

func (it *sliceIterator) Next() int32 {
	value := it.values[it.position]
	it.position += 1
	return value
}

func (it *sliceIterator) MaxSize() int {
	return len(it.values)
}

func (it *sliceIterator) SkipUntil(val int32) {
	for it.position < len(it.values) && it.values[it.position] < val {
		it.position += 1
	}
}
