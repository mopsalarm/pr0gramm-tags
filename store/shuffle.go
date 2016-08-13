package store

import (
	"math/rand"
	"time"
)

func shuffleOne(values []int32, i int) {
	// choose index uniformly in [i, N-1]
	r := i + rand.Intn(len(values) - i)
	values[r], values[i] = values[i], values[r]
}

type it32shuffleIter struct {
	rng    *rand.Rand
	pos    int
	values []int32
}

func NewShuffledIterator(iter ItemIterator) ItemIterator {
	values := IteratorToList(nil, iter)

	result := &it32shuffleIter{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
		values: values,
	}

	result.advance()
	return result
}

func (it *it32shuffleIter) advance() {
	if it.pos < len(it.values) {
		shuffleOne(it.values, it.pos)
		it.pos += 1
	}
}

func (it *it32shuffleIter) HasMore() bool {
	return it.pos < len(it.values)
}

func (it *it32shuffleIter) Peek() int32 {
	return it.values[it.pos]
}

func (it *it32shuffleIter) Next() int32 {
	// but return the "current" previous
	it.advance()
	return it.values[it.pos - 1]
}

func (it *it32shuffleIter) MaxSize() int {
	return len(it.values)
}
