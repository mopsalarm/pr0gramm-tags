#include "sequence_c.h"
#include "sequence.hpp"

#include <iostream>
#include <unordered_map>

typedef std::unordered_map<uint32_t, sequence> map_t;

inline map_t& get(store ptr) {
    return *(map_t*) ptr;
}

extern "C" {

    store store_new() {
        return (store) new map_t();
    }

    void store_destroy(store map) {
        delete (map_t*) map;
    }

    uint32_t store_length(store map) {
        return get(map).size();
    }

    void store_seq_push(store map, uint32_t key, uint8_t byte) {
        get(map)[key].push(byte);
    }

    void store_seq_push_n(store st, uint32_t key, uint8_t *bytes, int n) {
        auto&& map = get(st);
        auto&& seq = map[key];

        for(int i = 0; i < n; i++) {
            seq.push(bytes[i]);
        }
    }

    uint32_t store_seq_length(store map, uint32_t key) {
        return get(map)[key].length();
    }

    void store_seq_compact(store map, uint32_t key) {
        get(map)[key].compact();
    }

    void store_remove_key(store map, uint32_t key) {
        get(map).erase(key);
    }

    void store_clear_key(store map, uint32_t key) {
        get(map)[key].clear();
    }

    int store_contains(store map, uint32_t key) {
        return get(map).find(key) != get(map).end();
    }

    uint32_t store_memory_size(store map) {
        uint32_t sum = 0;
        for(auto&& it : get(map)) {
            sum += it.second.memory_size();
        }

        return sum;
    }

    struct byte_view store_get(store st, uint32_t key) {
        auto&& iter = get(st).find(key);
        if (iter != get(st).end()) {
            return get(st)[key].view();
        } else {
            return byte_view{0, nullptr};
        }
    }

    int store_keys(store st, uint32_t *keys, int n) {
        int pos = 0;

        for(auto&& entry : get(st)) {
            if(pos < n) {
                keys[pos] = entry.first;
                pos++;
            }
        }

        return pos;
    }
}
