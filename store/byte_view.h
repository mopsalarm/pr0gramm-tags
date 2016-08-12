#include <stdint.h>

#pragma once

struct byte_view {
    uint32_t length;
    const uint8_t * data;
};
