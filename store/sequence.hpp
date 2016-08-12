#include <cstdint>
#include <stdexcept>

#include "byte_view.h"

#pragma once

class sequence {
private:
    union {
        struct {
            uint32_t length;
            uint32_t capacity;
            uint8_t *ptr;
        } heap;

        uint8_t direct[sizeof(heap)];
    } d;

    bool inline_data;

    inline uint32_t inline_length() const;


public:
    sequence();
    sequence(const sequence& other) = delete;

    ~sequence();

    inline uint32_t length() const;
    inline byte_view view() const;

    void push(uint8_t value);
    void compact();
    void clear();

    uint32_t memory_size() const;

    static uint32_t max_inline_bytes;
} __attribute__((packed));

uint32_t sequence::length() const {
    if (inline_data) {
        return d.direct[0];
    } else {
        return d.heap.length;
    }
}

 byte_view sequence::view() const {
    if (inline_data) {
        return byte_view{inline_length(), &d.direct[1]};
    } else {
        return byte_view{d.heap.length, d.heap.ptr};
    }
}

uint32_t sequence::inline_length() const {
    if (!inline_data)
        throw std::runtime_error("asking for inline length when using heap data");

    return d.direct[0];
}
