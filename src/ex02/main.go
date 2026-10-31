package main

import (
	"fmt"
	"os"
	"strconv"
)

// generate запускает горутину, которая передаёт числа от k до n включительно.
func generate(k, n int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for number := k; number <= n; number++ {
			out <- number

			// Проверяем границу до инкремента, чтобы не переполнить максимальный int.
			if number == n {
				return
			}
		}
	}()

	return out
}

// square последовательно возводит числа из входного канала в квадрат.
func square(input <-chan int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		// range завершается, когда входной канал закрыт и значения прочитаны.
		for number := range input {
			out <- number * number
		}
	}()

	return out
}

// fitsSquare проверяет, поместится ли квадрат числа в тип int без переполнения.
func fitsSquare(number int) bool {
	maxInt := int(^uint(0) >> 1)

	// Модуль минимального int нельзя получить обычным унарным минусом.
	if number < -maxInt {
		return false
	}
	if number < 0 {
		number = -number
	}

	return number == 0 || number <= maxInt/number
}

// parseArgs читает K и N и проверяет допустимость входных данных.
func parseArgs(args []string) (int, int, error) {
	if len(args) != 2 {
		return 0, 0, fmt.Errorf("expected two integers: K N")
	}

	k, err := strconv.Atoi(args[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid K %q: %w", args[0], err)
	}

	n, err := strconv.Atoi(args[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid N %q: %w", args[1], err)
	}

	if k > n {
		return 0, 0, fmt.Errorf("K must not exceed N")
	}
	if !fitsSquare(k) || !fitsSquare(n) {
		return 0, 0, fmt.Errorf("squaring K or N would overflow int")
	}

	return k, n, nil
}

func main() {
	k, n, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid arguments:", err)
		os.Exit(1)
	}

	// Читаем только канал с готовыми квадратами; ожидание происходит через range.
	for result := range square(generate(k, n)) {
		fmt.Println(result)
	}
}
