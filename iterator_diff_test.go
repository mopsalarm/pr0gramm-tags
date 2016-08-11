package main

import "testing"

func TestDiffIteratorSecondInFirst(t *testing.T) {
	testIter(t, iter(1, 2),
		NewDiffIterator(
			iter(1, 2, 3, 4, 5),
			iter(3, 4, 5)))
}

func TestDiffIteratorFirstInSecond(t *testing.T) {
	testIter(t, iter(),
		NewDiffIterator(
			iter(3, 4, 5),
			iter(1, 2, 3, 4, 5)))
}

func TestDiffIteratorDisjunct(t *testing.T) {
	testIter(t, iter(3, 4, 8),
		NewDiffIterator(
			iter(3, 4, 8),
			iter(1, 6, 7, 9)))
}

func TestDiffIteratorSomeMatching(t *testing.T) {
	testIter(t, iter(3, 4, 8, 10),
		NewDiffIterator(
			iter(3, 4, 6, 7, 8, 9, 10),
			iter(1, 6, 7, 9)))
}

func TestDiffIteratorSomeMatching2(t *testing.T) {
	testIter(t, iter(3, 4, 8),
		NewDiffIterator(
			iter(3, 4, 6, 7, 8, 9),
			iter(1, 6, 7, 9, 10)))
}
