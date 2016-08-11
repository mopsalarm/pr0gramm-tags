package main

import "testing"

func TestOrIteratorSecondInFirst(t *testing.T) {
	testIter(t, iter(1, 2, 3, 4, 5),
		NewOrIterator(
			iter(1, 2, 3, 4, 5),
			iter(3, 4, 5)))
}

func TestOrIteratorFirstInSecond(t *testing.T) {
	testIter(t, iter(1, 2, 3, 4, 5),
		NewOrIterator(
			iter(3, 4, 5),
			iter(1, 2, 3, 4, 5)))
}

func TestOrIteratorDisjunct(t *testing.T) {
	testIter(t, iter(1, 3, 4, 6, 7, 8, 9),
		NewOrIterator(
			iter(3, 4, 8),
			iter(1, 6, 7, 9)))
}

func TestOrIteratorSomeMatching(t *testing.T) {
	testIter(t, iter(1, 3, 4, 6, 7, 8, 9, 10),
		NewOrIterator(
			iter(3, 4, 6, 7, 8, 9, 10),
			iter(1, 6, 7, 9)))
}

func TestOrIteratorSomeMatching2(t *testing.T) {
	testIter(t, iter(1, 3, 4, 6, 7, 8, 9, 10),
		NewOrIterator(
			iter(3, 4, 6, 7, 8, 9),
			iter(1, 6, 7, 9, 10)))
}

