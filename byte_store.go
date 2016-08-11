package main

// #cgo CXXFLAGS: -std=c++11 -O3
// #include "sequence_c.h"
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
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

	MemorySize() ByteSize
}

type cppStore struct {
	p C.store
}

func NewCppStore() *cppStore {
	p := C.store_new()

	store := &cppStore{p: p}
	runtime.SetFinalizer(store, func(st *cppStore) {
		C.store_destroy(st.p)
	})

	return store
}

func (store *cppStore) Push(key uint32, value byte) {
	C.store_seq_push(store.p, C.uint32_t(key), C.uint8_t(value))
}

func (store *cppStore) PushN(key uint32, values []byte) {
	C.store_seq_push_n(store.p, C.uint32_t(key),
		(*C.uint8_t)(&values[0]), C.int(len(values)))
}

func (store *cppStore) KeyCount() uint32 {
	return uint32(C.store_length(store.p))
}

func (store *cppStore) Compact(key uint32) {
	C.store_seq_compact(store.p, C.uint32_t(key))
}

func (store *cppStore) SeqLength(key uint32) uint32 {
	return uint32(C.store_seq_length(store.p, C.uint32_t(key)))
}

func (store *cppStore) Remove(key uint32) {
	C.store_remove_key(store.p, C.uint32_t(key))
}

func (store *cppStore) Clear(key uint32) {
	C.store_clear_key(store.p, C.uint32_t(key))
}

func (store *cppStore) Contains(key uint32) bool {
	return int(C.store_contains(store.p, C.uint32_t(key))) != 0
}

func (store *cppStore) MemorySize() ByteSize {
	return ByteSize(C.store_memory_size(store.p))
}

// You my not hold on to this slice!
func (store *cppStore) Get(key uint32) []byte {
	bv := C.store_get(store.p, C.uint32_t(key))
	length := int(bv.length)

	if length > 1<<24 {
		panic(fmt.Errorf("Sequence is too long (%dbyte).", length))
	}

	if length == 0 {
		return nil
	}

	return (*[1 << 24]byte)(unsafe.Pointer(bv.data))[:length:length]
}

func (store *cppStore) Keys() []uint32 {
	keys := make([]uint32, store.KeyCount())
	n := int(C.store_keys(store.p, (*C.uint32_t)(unsafe.Pointer(&keys[0])), C.int(cap(keys))))
	return keys[:n]
}
