package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"sync"
	"time"
)

// Храним номер горутины вместе с назначенной ей длительностью сна.
type taskResult struct {
	id           int
	milliseconds int
}

// Запускаем n задач и возвращаем результаты по убыванию времени сна.
func runTasks(n, maxMilliseconds int) []taskResult {
	results := make([]taskResult, n)
	var wg sync.WaitGroup

	// Добавляем все задачи до запуска: Wait не должен увидеть нулевой счётчик раньше времени.
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(index int) {
			defer wg.Done()

			// Int63n исключает верхнюю границу, поэтому прибавляем 1.
			milliseconds := int(rand.Int63n(int64(maxMilliseconds) + 1))
			time.Sleep(time.Duration(milliseconds) * time.Millisecond)

			// У каждой горутины свой индекс; разные элементы среза не конфликтуют.
			results[index] = taskResult{id: index, milliseconds: milliseconds}
		}(i)
	}

	// После Wait все записи завершены, поэтому сортировка и чтение безопасны.
	wg.Wait()
	sort.Slice(results, func(i, j int) bool {
		if results[i].milliseconds == results[j].milliseconds {
			return results[i].id < results[j].id
		}
		return results[i].milliseconds > results[j].milliseconds
	})
	return results
}

func main() {
	n := flag.Int("n", -1, "number of goroutines")
	m := flag.Int("m", -1, "maximum sleep time in milliseconds")
	flag.Parse()

	if *n < 0 || *m < 0 || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "invalid arguments: specify -n >= 0 and -m >= 0")
		os.Exit(1)
	}

	// time.Duration хранит наносекунды в int64: не даём перевести миллисекунды с переполнением.
	if int64(*m) > math.MaxInt64/int64(time.Millisecond) {
		fmt.Fprintln(os.Stderr, "invalid arguments: -m is too large")
		os.Exit(1)
	}

	for _, result := range runTasks(*n, *m) {
		fmt.Printf("%d %d\n", result.id, result.milliseconds)
	}
}
