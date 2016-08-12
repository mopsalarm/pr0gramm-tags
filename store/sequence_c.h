#include <stdint.h>
#include "byte_view.h"

#ifdef __cplusplus
extern "C" {
#endif
    typedef void* store;

    store store_new();
    void store_destroy(store map);
    uint32_t store_length(store map);

    void store_seq_push(store map, uint32_t key, uint8_t byte);
    void store_seq_push_n(store map, uint32_t key, uint8_t *bytes, int n);


    uint32_t store_seq_length(store map, uint32_t key);
    void store_seq_compact(store map, uint32_t key);
    void store_remove_key(store map, uint32_t key);
    void store_clear_key(store map, uint32_t key);

    int store_contains(store map, uint32_t key);

    uint32_t store_memory_size(store map);

    struct byte_view store_get(store, uint32_t key);

    int store_keys(store, uint32_t *keys, int n);

#ifdef __cplusplus
}
#endif
