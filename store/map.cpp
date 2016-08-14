#include <cstddef>

#include <vector>
#include <iterator>

template<class K, class V>
class small_map {
private:
    std::vector<K> keys;
    std::vector<V> values;

public:
    inline size_t size() const;
    bool contains(const K& key) const;

    V& get(const K& key);
    void erase(const K& key);

    inline V& operator[](const K& key) {
        return get(key);
    }

    inline const std::vector<K>& get_keys() {
        return keys;
    }

    inline const std::vector<V>& get_values() {
        return values;
    }
};

template<class K, class V>
V& small_map<K, V>::get(const K& key) {
    auto it = std::lower_bound(keys.begin(), keys.end(), key);
    auto valueIter = values.begin() + std::distance(keys.begin(), it);
    if(it == keys.end() || *it != key) {
        it = keys.emplace(it, key);
        valueIter = values.emplace(valueIter);
    }

    return *valueIter;
}

template<class K, class V>
void small_map<K, V>::erase(const K& key) {
    auto it = std::lower_bound(keys.begin(), keys.end(), key);
    if(it != keys.end() && *it == key) {
        keys.erase(it);
        values.erase(values.begin() + std::distance(keys.begin(), it));
    }
}

template<class K, class V>
bool small_map<K, V>::contains(const K& key) const {
    auto it = std::lower_bound(keys.begin(), keys.end(), key);
    return it != keys.end() && *it == key;
}

template<class K, class V>
inline
size_t small_map<K, V>::size() const {
    return keys.size();
}


#ifdef DEBUG_WITH_MAIN
#include <iostream>

int main() {
    small_map<int, long> map;
    map.get(0) = 10;
    std::cout << "size: " << map.size() << std::endl;
    std::cout << "value: " << map.get(0) << std::endl;
    std::cout << "contains 0: " << map.contains(0) << std::endl;
    std::cout << "contains 1: " << map.contains(1) << std::endl;
    std::cout << "contains -1: " << map.contains(-1) << std::endl;

    map.erase(0);
    std::cout << "size: " << map.size() << std::endl;
    return 0;
}
#endif
