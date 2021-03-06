package store

import (
	"fmt"
)

type IterStore interface {
	KeyCount() uint32
	Keys() []uint32

	GetIterator(key uint32) ItemIterator

	Replace(key uint32, values []int32)
	MemorySize() ByteSize
}

type iterStore struct {
	ByteStore
}

func NewIterStore(store ByteStore) IterStore {
	if store == nil {
		store = NewByteStore()
	}

	return &iterStore{store}
}

func (store *iterStore) PushInt(key uint32, value int32) {
	bytes := store.Get(key)
	if len(bytes) == 0 {
		codec := int24CodecInstance
		store.Push(key, codec.Id())
		store.PushN(key, codec.Encode([]int32{value}))
	} else {
		codec := SequenceCodecById(bytes[0])
		if !codec.CanAppend() {
			panic(fmt.Errorf("Can not append to codec %s", codec))
		}

		store.PushN(key, codec.Encode([]int32{value}))
	}
}

func (store *iterStore) Replace(key uint32, values []int32) {
	if len(values) > 0 {
		store.Clear(key)

		codec := OptimalCodec(len(values))
		store.Push(key, codec.Id())
		store.PushN(key, codec.Encode(values))
		store.Compact(key)

	} else {
		store.Remove(key)
	}
}

func (store *iterStore) GetIterator(key uint32) ItemIterator {
	bytes := store.Get(key)
	if len(bytes) == 0 {
		return NewEmptyIterator()
	} else {
		codec := SequenceCodecById(bytes[0])
		return codec.Decode(bytes[1:])
	}
}

func MergeIterStores(target, other IterStore) {
	for _, key := range other.Keys() {
		values := IteratorToList(nil, NewOrIterator(target.GetIterator(key), other.GetIterator(key)))
		target.Replace(key, values)
	}
}
