package main

import "testing"

type sliceAdvancer struct {
	position int
	values   []int32
}

func (it *sliceAdvancer) Advance() (int32, bool) {
	if it.position < len(it.values) {
		value := it.values[it.position]
		it.position += 1
		return value, true
	} else {
		return 0, false
	}
}

func iter(values... int32) *ItemIterator {
	return NewItemIterator(&sliceAdvancer{values: values})
}

func testIter(t *testing.T, expected *ItemIterator, actual *ItemIterator) {
	for expected.Okay() && actual.Okay() {
		first := expected.Value()
		second := actual.Value()

		if first != second {
			t.Errorf("Iterator produced the value %d, but expected was %d", second, first)
			return
		}

		actual.Advance()
		expected.Advance()
	}

	if expected.Okay() {
		t.Errorf("The iterator produces not enought values. Next expected was %d", expected.Value())
	}

	if actual.Okay() {
		t.Errorf("The iterator produced too many values. Next unexpected was %d", actual.Value())
	}
}

func TestAndIteratorSecondInFirst(t *testing.T) {
	testIter(t, iter(3, 4, 5),
		NewAndIterator(
			iter(1, 2, 3, 4, 5),
			iter(3, 4, 5)))
}

func TestAndIteratorFirstInSecond(t *testing.T) {
	testIter(t, iter(3, 4, 5),
		NewAndIterator(
			iter(3, 4, 5),
			iter(1, 2, 3, 4, 5)))
}

func TestAndIteratorDisjunct(t *testing.T) {
	testIter(t, iter(),
		NewAndIterator(
			iter(3, 4, 8),
			iter(1, 6, 7, 9)))
}

func TestAndIteratorSomeMatching(t *testing.T) {
	testIter(t, iter(6, 7, 9),
		NewAndIterator(
			iter(3, 4, 6, 7, 8, 9, 10),
			iter(1, 6, 7, 9)))
}

func TestAndIteratorSomeMatching2(t *testing.T) {
	testIter(t, iter(6, 7, 9),
		NewAndIterator(
			iter(3, 4, 6, 7, 8, 9),
			iter(1, 6, 7, 9, 10)))
}

func TestAndIteratorSomeMatching3(t *testing.T) {
	testIter(t, iter(6, 7, 11),
		NewAndIterator(
			iter(3, 4, 6, 7, 8, 11),
			iter(1, 6, 7, 9, 10, 11)))
}
