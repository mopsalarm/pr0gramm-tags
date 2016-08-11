package main

import (
	"encoding/binary"
	"errors"
	"io"
)

type ItemIterator interface {
	HasMore() bool
	Next() int32
	Peek() int32
}

type FastItemIterator interface {
	ItemIterator
	SkipUntil(val int32)
}

func IteratorSkipUntil(iter ItemIterator, val int32) {
	if fast, ok := iter.(FastItemIterator); ok {
		fast.SkipUntil(val)
	} else {
		for iter.HasMore() && iter.Peek() < val {
			iter.Next()
		}
	}
}

type varintDeltaIterator struct {
	pos            int
	next, previous int32
	bytes          []byte
	more           bool
}

func NewSequenceIterator(bytes []byte) ItemIterator {
	it := &varintDeltaIterator{bytes: bytes, more: true}
	it.advance()
	return it
}

func (it *varintDeltaIterator) HasMore() bool {
	return it.more
}

func (it *varintDeltaIterator) Peek() int32 {
	return it.next
}

func (it *varintDeltaIterator) Next() int32 {
	current := it.next

	if it.more {
		it.advance()
	}

	return current
}

func (it *varintDeltaIterator) SkipUntil(val int32) {
	// allow inlining of the methods.
	for it.HasMore() && it.Peek() < val {
		it.advance()
	}
}

func (it *varintDeltaIterator) advance() {
	if it.pos < len(it.bytes) {
		next, n := fastVarint32(it.bytes[it.pos:])
		if n <= 0 {
			panic(errors.New("Could not decode varint."))
		}

		it.pos += n
		it.next = next + it.previous
		it.previous = it.next
	} else {
		it.more = false
	}
}

// Varint decodes an int32 from buf and returns that value and the
// number of bytes read (> 0). If an error occurred, the value is 0
// and the number of bytes n is <= 0 with the following meaning:
//
//	n == 0: buf too small
//	n  < 0: value larger than 32 bits (overflow)
//              and -n is the number of bytes read
//
func fastVarint32(buf []byte) (int32, int) {
	var ux uint32
	var s uint
	for i, b := range buf {
		if b < 0x80 {
			if i > 4 || i == 4 && b > 1 {
				return 0, -(i + 1) // overflow
			}

			ux = ux | uint32(b) << s
			x := int32(ux >> 1)
			if ux & 1 != 0 {
				x = ^x
			}

			return x, i + 1
		}

		ux |= uint32(b & 0x7f) << s
		s += 7
	}

	return 0, 0
}

type readerIterator struct {
	more    bool
	current int32
	reader  io.ByteReader
}

func NewIteratorFromReader(reader io.ByteReader) ItemIterator {
	r := &readerIterator{more: true, reader: reader}
	r.buffer()
	return r
}

func (it *readerIterator) buffer() {
	if it.more {
		value, err := binary.ReadVarint(it.reader)
		if err != nil {
			it.more = false
		}

		it.current += int32(value)
	}
}

func (it *readerIterator) HasMore() bool {
	return it.more
}

func (it *readerIterator) Peek() int32 {
	return it.current
}

func (it *readerIterator) Next() int32 {
	value := it.current
	it.buffer()
	return value
}

type negateIterator struct {
	ItemIterator
}

func NewNegateIterator(iter ItemIterator) ItemIterator {
	return &negateIterator{iter}
}

func (it *negateIterator) HasMore() bool {
	return it.ItemIterator.HasMore()
}

func (it *negateIterator) Peek() int32 {
	return -1 * it.ItemIterator.Peek()
}

func (it *negateIterator) Next() int32 {
	return -1 * it.ItemIterator.Next()
}

func (it *negateIterator) SkipUntil(val int32) {
	for it.HasMore() && it.Peek() < val {
		it.Next()
	}
}

type limitIterator struct {
	remaining int
	iter      ItemIterator
}

func NewLimitIterator(limit int, iter ItemIterator) ItemIterator {
	return &limitIterator{remaining: limit, iter: iter}
}

func (it *limitIterator) HasMore() bool {
	return it.remaining > 0 && it.iter.HasMore()
}

func (it *limitIterator) Peek() int32 {
	return it.iter.Peek()
}

func (it *limitIterator) Next() int32 {
	it.remaining -= 1
	return it.iter.Next()
}

func IteratorToList(result []int32, iter ItemIterator) []int32 {
	for iter.HasMore() {
		result = append(result, iter.Next())
	}

	return result
}

type andIterator struct {
	first, second ItemIterator
}

func NewAndIterator(first, second ItemIterator) ItemIterator {
	return &andIterator{first, second}
}

func (it *andIterator) HasMore() bool {
	for it.first.HasMore() && it.second.HasMore() {
		a := it.first.Peek()
		b := it.second.Peek()

		if a == b {
			return true
		}

		if a < b {
			IteratorSkipUntil(it.first, b)
		} else {
			IteratorSkipUntil(it.second, a)
		}
	}

	return false
}

func (it *andIterator) Peek() int32 {
	return it.first.Peek()
}

func (it *andIterator) Next() int32 {
	return it.first.Next()
}

func (it *andIterator) SkipUntil(val int32) {
	// allow inlining of the methods.
	for it.HasMore() && it.Peek() < val {
		it.Next()
	}
}

type orIterator struct {
	first, second ItemIterator
}

func NewOrIterator(first, second ItemIterator) ItemIterator {
	return &orIterator{first, second}
}

func (it *orIterator) HasMore() bool {
	return it.first.HasMore() || it.second.HasMore()
}

func (it *orIterator) Peek() int32 {
	if it.first.HasMore() && it.second.HasMore() {
		a := it.first.Peek()
		b := it.second.Peek()

		if a < b {
			return a
		} else {
			return b
		}
	}

	if it.first.HasMore() {
		return it.first.Peek()
	}

	if it.second.HasMore() {
		return it.second.Peek()
	}

	panic(errors.New("end of iterator"))
}

func (it *orIterator) Next() int32 {
	if it.first.HasMore() && it.second.HasMore() {
		a := it.first.Peek()
		b := it.second.Peek()

		if a == b {
			it.second.Next()
			return it.first.Next()
		}

		if a < b {
			return it.first.Next()
		} else {
			return it.second.Next()
		}
	}

	if it.first.HasMore() {
		return it.first.Next()
	}

	if it.second.HasMore() {
		return it.second.Next()
	}

	panic(errors.New("end of iterator"))
}

func (it *orIterator) SkipUntil(val int32) {
	// allow inlining of the methods.
	for it.HasMore() && it.Peek() < val {
		it.Next()
	}
}

type diffIterator struct {
	first, second ItemIterator
}

func NewDiffIterator(first, second ItemIterator) ItemIterator {
	return &diffIterator{first, second}
}

func (it *diffIterator) HasMore() bool {
	for it.first.HasMore() {
		if ! it.second.HasMore() {
			return it.first.HasMore()
		}

		a := it.first.Peek()
		b := it.second.Peek()

		if a < b {
			return true
		} else if a == b {
			a = it.first.Next()
			IteratorSkipUntil(it.second, a)
		} else {
			IteratorSkipUntil(it.second, a)
		}
	}

	return false
}

func (it *diffIterator) Peek() int32 {
	return it.first.Peek()
}

func (it *diffIterator) Next() int32 {
	return it.first.Next()
}

func (it *diffIterator) SkipUntil(val int32) {
	// allow inlining of the methods.
	for it.HasMore() && it.Peek() < val {
		it.Next()
	}
}

type cachingIterator struct {
	iter ItemIterator
	next int32
	more bool
}

func NewCachingIterator(iter ItemIterator) ItemIterator {
	i := &cachingIterator{iter: iter, more: true}
	i.buffer()
	return i
}

func (it *cachingIterator) buffer() {
	if it.more {
		if it.iter.HasMore() {
			it.next = it.iter.Next()
		} else {
			it.more = false
		}
	}
}

func (it *cachingIterator) HasMore() bool {
	return it.more
}

func (it *cachingIterator) Peek() int32 {
	return it.next
}

func (it *cachingIterator) Next() int32 {
	value := it.next
	it.buffer()
	return value
}
