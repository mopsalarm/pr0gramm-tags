#include "sequence.hpp"

#include <algorithm>

uint32_t sequence::max_inline_bytes = sizeof(sequence::d.direct) - 1;

uint8_t *allocate_and_copy(uint32_t length, byte_view previous) {
    if (length < previous.length) {
        throw std::runtime_error("can not allocate less bytes than there is data");
    }

    auto data = new uint8_t[length];
    std::copy_n(previous.data, previous.length, data);
    return data;
}

sequence::sequence() : inline_data(true) {
    d.direct[0] = 0;
    d.heap.length = 0;
    d.heap.capacity = 0;
    d.heap.ptr = nullptr;
}

sequence::sequence(sequence&& other) {
    inline_data = other.inline_data;
    if(inline_data) {
        std::copy_n(other.d.direct, sizeof(other.d.direct), d.direct);
    } else {
        d.heap = other.d.heap;
    }

    other.inline_data = true;
    other.d.direct[0] = 0;
}

sequence& sequence::operator=(sequence&& other) {
    if (!inline_data) {
        delete[] d.heap.ptr;
    }

    inline_data = other.inline_data;
    if(inline_data) {
        std::copy_n(other.d.direct, sizeof(other.d.direct), d.direct);
    } else {
        d.heap = other.d.heap;
    }

    other.inline_data = true;
    other.d.direct[0] = 0;
}

sequence::~sequence() {
    if (!inline_data) {
        delete[] d.heap.ptr;
    }
}

void sequence::clear() {
    if (!inline_data) {
        delete[] d.heap.ptr;
    }

    inline_data = true;
    d.direct[0] = 0;
}

void sequence::push(uint8_t value) {
    if (inline_data) {
        const auto move_to_heap = (inline_length() + 1u > max_inline_bytes);
        if(move_to_heap) {
            auto new_length = inline_length();
            auto new_capacity = 8 + 2 * max_inline_bytes;
            auto new_data = allocate_and_copy(new_capacity, view());

            d.heap.length = new_length;
            d.heap.capacity = new_capacity;
            d.heap.ptr = new_data;
            inline_data = false;

            // now push the real value.
            push(value);
            return;
        }

        d.direct[1 + d.direct[0]++] = value;
    } else {
        const auto need_to_grow = d.heap.length == d.heap.capacity;
        if(need_to_grow) {
            auto new_capacity = 8 + 13 * d.heap.capacity / 10;
            auto new_data = allocate_and_copy(new_capacity, view());

            delete[] d.heap.ptr;
            d.heap.ptr = new_data;
            d.heap.capacity = new_capacity;
        }

        d.heap.ptr[d.heap.length++] = value;
    }
}

void sequence::compact() {
    if (!inline_data) {
        auto new_capacity = d.heap.length;
        auto new_data = allocate_and_copy(new_capacity, view());

        delete[] d.heap.ptr;
        d.heap.ptr = new_data;
        d.heap.capacity = new_capacity;
    }
}

uint32_t sequence::memory_size() const {
    auto size = sizeof(*this);
    if(!inline_data) {
        size += d.heap.capacity;
    }

    return size;
}

#ifdef DEBUG_WITH_MAIN

#include <iostream>

int main() {
    sequence seq;
    for(int i = 0; i < 1000000; i++)
        seq.push(i);

    std::cout << seq.length() << std::endl;
    std::cout << seq.view().length << std::endl;
}

#endif
