package cache_test

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"testing"

	cache "go_bootcamp_2/ex04"
)

// Новый ключ вытесняет наименее недавно использованный элемент при заполнении.
func TestEvictionWhenFull(t *testing.T) {
	c := cache.NewCache[int](2)
	c.Set("A", 10)
	c.Set("B", 20)
	c.Set("C", 30)

	if value, ok := c.Get("A"); ok || value != 0 {
		t.Errorf("A = (%d, %v), want (0, false)", value, ok)
	}
	if value, ok := c.Get("B"); !ok || value != 20 {
		t.Errorf("B = (%d, %v), want (20, true)", value, ok)
	}
	if value, ok := c.Get("C"); !ok || value != 30 {
		t.Errorf("C = (%d, %v), want (30, true)", value, ok)
	}
}

// Чтение обновляет приоритет: ранее прочитанный элемент не должен вытесняться первым.
func TestGetRefreshesRecency(t *testing.T) {
	c := cache.NewCache[int](2)
	c.Set("A", 10)
	c.Set("B", 20)
	if value, ok := c.Get("A"); !ok || value != 10 {
		t.Fatalf("A = (%d, %v), want (10, true)", value, ok)
	}
	c.Set("C", 30)

	if _, ok := c.Get("B"); ok {
		t.Error("B should be evicted after Get(A)")
	}
	if value, ok := c.Get("A"); !ok || value != 10 {
		t.Errorf("A = (%d, %v), want (10, true)", value, ok)
	}
}

// Повторный Set обновляет значение без увеличения количества элементов.
func TestSetUpdatesValueAndRecency(t *testing.T) {
	c := cache.NewCache[int](2)
	c.Set("A", 1)
	c.Set("B", 2)
	c.Set("A", 3)
	c.Set("C", 4)

	if value, ok := c.Get("A"); !ok || value != 3 {
		t.Errorf("A = (%d, %v), want (3, true)", value, ok)
	}
	if _, ok := c.Get("B"); ok {
		t.Error("B should be evicted after Set(A)")
	}
	if value, ok := c.Get("C"); !ok || value != 4 {
		t.Errorf("C = (%d, %v), want (4, true)", value, ok)
	}
}

// Clear удаляет старые записи и позволяет снова использовать ту же ёмкость.
func TestClear(t *testing.T) {
	c := cache.NewCache[string](2)
	c.Set("A", "hello")
	c.Set("B", "world")
	c.Clear()
	c.Clear()

	for _, key := range []string{"A", "B"} {
		if value, ok := c.Get(key); ok || value != "" {
			t.Errorf("Get(%q) = (%q, %v), want empty string and false", key, value, ok)
		}
	}

	c.Set("C", "new")
	if value, ok := c.Get("C"); !ok || value != "new" {
		t.Errorf("Get(C) = (%q, %v), want (new, true)", value, ok)
	}
}

// При промахе возвращается zero-value именно типа T.
func TestMissingReturnsZeroValue(t *testing.T) {
	c := cache.NewCache[*int](1)
	if value, ok := c.Get("missing"); ok || value != nil {
		t.Errorf("Get(missing) = (%v, %v), want (nil, false)", value, ok)
	}
}

// Нулевая и отрицательная ёмкость трактуются как пустой кэш.
func TestNonPositiveCapacity(t *testing.T) {
	for _, capacity := range []int{0, -1} {
		t.Run(strconv.Itoa(capacity), func(t *testing.T) {
			c := cache.NewCache[int](capacity)
			c.Set("A", 1)
			if value, ok := c.Get("A"); ok || value != 0 {
				t.Errorf("capacity %d: got (%d, %v), want (0, false)", capacity, value, ok)
			}
			c.Clear()
		})
	}
}

// Generic T может быть и несравнимым типом: например, срезом.
func TestGenericSliceValues(t *testing.T) {
	c := cache.NewCache[[]int](2)
	c.Set("numbers", []int{1, 2, 3})

	value, ok := c.Get("numbers")
	if !ok || !reflect.DeepEqual(value, []int{1, 2, 3}) {
		t.Errorf("Get(numbers) = (%v, %v)", value, ok)
	}
}

// Под нагрузкой несколько горутин одновременно выполняют Set, Get и Clear.
// Детектор гонок дополняет проверку согласованности после завершения работы.
func TestConcurrentAccess(t *testing.T) {
	const capacity = 16
	c := cache.NewCache[int](capacity)
	var wg sync.WaitGroup

	for worker := 0; worker < 12; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()

			for i := 0; i < 300; i++ {
				key := strconv.Itoa((worker + i) % 32)
				c.Set(key, i)
				c.Get(key)
				if i%70 == 0 {
					c.Clear()
				}
			}
		}(worker)
	}
	wg.Wait()

	// По завершении конкурентных операций размер не превышает ёмкость.
	count := 0
	for i := 0; i < 32; i++ {
		if _, ok := c.Get(strconv.Itoa(i)); ok {
			count++
		}
	}
	if count > capacity {
		t.Errorf("found %d entries, capacity is %d", count, capacity)
	}

	c.Clear()
	c.Set("final", 42)
	if value, ok := c.Get("final"); !ok || value != 42 {
		t.Errorf("cache unusable after concurrent operations: (%d, %v)", value, ok)
	}
}

// Пример показывает изменение приоритета после чтения.
func ExampleCache() {
	c := cache.NewCache[int](2)
	c.Set("A", 10)
	c.Set("B", 20)

	value, ok := c.Get("A")
	fmt.Println(value, ok)

	c.Set("C", 30)
	value, ok = c.Get("B")
	fmt.Println(value, ok)

	// Output:
	// 10 true
	// 0 false
}
