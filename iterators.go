package main

import (
	"errors"
)

type Advancer interface {
	Advance() (int32, bool)
}

type ItemIterator struct {
	advancer Advancer
	value    int32
	okay     bool
}

func NewItemIterator(advancer Advancer) *ItemIterator {
	value, okay := advancer.Advance()
	return &ItemIterator{advancer: advancer, value:value, okay: okay}
}

func (it *ItemIterator) Okay() bool {
	return it.okay
}

func (it *ItemIterator) Value() int32 {
	return it.value
}

func (it *ItemIterator) Advance() (int32, bool) {
	value, okay := it.advancer.Advance()
	it.value = value
	it.okay = okay
	return value, okay
}

func IteratorSkipUntil(iter *ItemIterator, val int32) {
	for iter.Okay() && iter.Value() < val {
		iter.Advance()
	}
}

type deltaAdvancer struct {
	pos      int
	previous int32
	bytes    []byte
}

func NewSequenceIterator(bytes []byte) *ItemIterator {
	return NewItemIterator(&deltaAdvancer{bytes: bytes})
}

func (it *deltaAdvancer) Advance() (int32, bool) {
	if it.pos < len(it.bytes) {
		next, n := fastVarint32(it.bytes[it.pos:])
		if n <= 0 {
			panic(errors.New("Could not decode varint."))
		}

		it.pos += n

		value := next + it.previous
		it.previous = value

		return value, true
	} else {
		return 0, false
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

func IteratorToList(result []int32, iter *ItemIterator, maxValues int) []int32 {
	for i := 0; i < maxValues && iter.Okay(); i++ {
		result = append(result, iter.Value())
	}

	return result
}

type andAdvancer struct {
	first, second *ItemIterator
}

func NewAndIterator(first, second *ItemIterator) *ItemIterator {
	return NewItemIterator(&andAdvancer{first: first, second: second})
}

func (it *andAdvancer) Advance() (int32, bool) {
	for it.first.Okay() && it.second.Okay() {
		a := it.first.Value()
		b := it.second.Value()

		if a == b {
			it.first.Advance()
			it.second.Advance()
			return a, true
		}

		if a < b {
			IteratorSkipUntil(it.first, b)
		} else {
			IteratorSkipUntil(it.second, a)
		}
	}

	return 0, false
}

type orAdvancer struct {
	first, second *ItemIterator
}

func NewOrIterator(first, second *ItemIterator) *ItemIterator {
	return NewItemIterator(&orAdvancer{first: first, second: second})
}

func (it *orAdvancer) Advance() (int32, bool) {
	aOkay := it.first.Okay()
	bOkay := it.second.Okay()

	if aOkay && bOkay {
		a := it.first.Value()
		b := it.second.Value()

		switch {
		case a < b:
			it.first.Advance()
			return a, true

		case a == b:
			it.first.Advance()
			it.second.Advance()
			return a, true

		case a > b:
			it.second.Advance()
			return b, true
		}
	}

	if aOkay {
		value := it.first.Value()
		it.first.Advance()
		return value, true
	}

	if bOkay {
		value := it.second.Value()
		it.second.Advance()
		return value, true
	}

	return 0, false
}

type diffAdvancer struct {
	first, second *ItemIterator
}

func NewDiffIterator(first, second *ItemIterator) *ItemIterator {
	return NewItemIterator(&diffAdvancer{first:first, second:second})
}

func (it *diffAdvancer) Advance() (int32, bool) {
	for it.first.Okay() {
		if !it.second.Okay() {
			value := it.first.Value()
			it.first.Advance()
			return value, true
		}

		a := it.first.Value()
		b := it.second.Value()

		switch {
		case a < b:
			it.first.Advance()
			return a, true

		case a > b:
			IteratorSkipUntil(it.second, a)

		case a == b:
			it.first.Advance()
			IteratorSkipUntil(it.second, a)
		}
	}

	return 0, false
}
