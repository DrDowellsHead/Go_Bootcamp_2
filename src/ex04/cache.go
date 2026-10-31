package cache

import (
	"container/list"
	"sync"
)

// cacheItem хранит ключ и значение одного элемента двусвязного списка.
type cacheItem[T any] struct {
	key   string
	value T
}

// Cache хранит не более capacity элементов в порядке последнего использования.
// Все обращения к map и списку защищены одним мьютексом.
type Cache[T any] struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

// NewCache создаёт LRU-кэш для значений типа T.
// При нулевой или отрицательной ёмкости элементы не сохраняются.
func NewCache[T any](capacity int) *Cache[T] {
	if capacity < 0 {
		capacity = 0
	}

	return &Cache[T]{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

// Set добавляет значение или обновляет существующее, помечая его недавно использованным.
func (c *Cache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity == 0 {
		return
	}

	if element, found := c.items[key]; found {
		element.Value.(*cacheItem[T]).value = value
		c.order.MoveToFront(element)
		return
	}

	element := c.order.PushFront(&cacheItem[T]{key: key, value: value})
	c.items[key] = element

	// Удаляем последний элемент списка и соответствующий ключ из map.
	if c.order.Len() > c.capacity {
		oldest := c.order.Back()
		delete(c.items, oldest.Value.(*cacheItem[T]).key)
		c.order.Remove(oldest)
	}
}

// Get возвращает значение и true, одновременно перемещая найденный элемент в начало.
// При отсутствии ключа возвращает нулевое значение типа T и false.
func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, found := c.items[key]
	if !found {
		var zero T
		return zero, false
	}

	c.order.MoveToFront(element)
	return element.Value.(*cacheItem[T]).value, true
}

// Clear очищает map и список, не меняя ёмкость кэша.
func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	clear(c.items)
	c.order.Init()
}
