package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

var byteOrder = binary.BigEndian

type StoreState struct {
	LastTagId  int
	LastItemId int
}

func WriteCheckpoint(writer io.Writer, state StoreState, store IterStore) error {
	{
		bytes, err := json.Marshal(state)
		if err != nil {
			return err
		}

		if err := binary.Write(writer, byteOrder, uint32(len(bytes))); err != nil {
			return err
		}

		if _, err := writer.Write(bytes); err != nil {
			return err
		}
	}

	keyCount := uint32(store.KeyCount())
	if err := binary.Write(writer, byteOrder, keyCount); err != nil {
		return err
	}

	var values []int32
	for _, key := range store.Keys() {
		values = IteratorToList(values[:0], store.GetIterator(key))

		if err := binary.Write(writer, byteOrder, uint32(key)); err != nil {
			return err
		}

		if err := binary.Write(writer, byteOrder, uint32(len(values))); err != nil {
			return err
		}

		if err := binary.Write(writer, byteOrder, values); err != nil {
			return err
		}
	}

	return nil
}

func WriteCheckpointFile(filename string, state StoreState, store IterStore) error {
	tempname := fmt.Sprintf("%s.%d", filename, time.Now().UnixNano())
	fp, err := os.Create(tempname)
	if err != nil {
		return err
	}

	defer fp.Close()

	// wrap into a buffered writer
	writer := bufio.NewWriterSize(fp, 16*1024)

	// write the store now.
	WriteCheckpoint(writer, state, store)
	if err := writer.Flush(); err != nil {
		return err
	}

	// close file
	if err := fp.Close(); err != nil {
		return err
	}

	// rename to the target name
	return os.Rename(tempname, filename)
}

func ReadCheckpoint(reader io.Reader, state *StoreState, store IterStore) error {
	{
		var jsonLength uint32
		if err := binary.Read(reader, byteOrder, &jsonLength); err != nil {
			return err
		}

		bytes := make([]byte, jsonLength)
		if _, err := io.ReadFull(reader, bytes); err != nil {
			return err
		}

		if err := json.Unmarshal(bytes, state); err != nil {
			return err
		}
	}

	var keyCount uint32
	if err := binary.Read(reader, byteOrder, &keyCount); err != nil {
		return err
	}

	for idx := uint32(0); idx < keyCount; idx++ {
		var key, valueCount uint32
		if err := binary.Read(reader, byteOrder, &key); err != nil {
			return err
		}

		if err := binary.Read(reader, byteOrder, &valueCount); err != nil {
			return err
		}

		values := make([]int32, valueCount)
		if err := binary.Read(reader, byteOrder, values); err != nil {
			return err
		}

		store.Replace(key, values)
	}

	return nil
}

func ReadCheckpointFile(filename string, state *StoreState, store IterStore) error {
	fp, err := os.Open(filename)
	if err != nil {
		return err
	}

	defer fp.Close()

	return ReadCheckpoint(bufio.NewReaderSize(fp, 16*1024), state, store)
}
