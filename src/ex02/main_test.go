package main

import (
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// collect читает канал до закрытия и сохраняет полученные числа.
func collect(input <-chan int) []int {
	var result []int

	for number := range input {
		result = append(result, number)
	}

	return result
}

// Проверяем включение границ, отрицательные числа и закрытие канала.
func TestGenerate(t *testing.T) {
	maxInt := int(^uint(0) >> 1)

	tests := []struct {
		name string
		k, n int
		want []int
	}{
		{"ascending", 2, 5, []int{2, 3, 4, 5}},
		{"negative and positive", -2, 2, []int{-2, -1, 0, 1, 2}},
		{"same bounds", 3, 3, []int{3}},
		{"reverse bounds", 4, 2, nil},
		{"maximum int", maxInt, maxInt, []int{maxInt}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collect(generate(tt.k, tt.n))

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generate(%d, %d) = %v, want %v",
					tt.k, tt.n, got, tt.want)
			}
		})
	}
}

// Проверяем последовательную обработку входных чисел и завершение по закрытию.
func TestSquare(t *testing.T) {
	input := make(chan int)

	go func() {
		defer close(input)

		for _, number := range []int{-3, -1, 0, 2, 4} {
			input <- number
		}
	}()

	got := collect(square(input))
	want := []int{9, 1, 0, 4, 16}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("square() = %v, want %v", got, want)
	}
}

// Проверяем обе горутины как единый конвейер.
func TestPipeline(t *testing.T) {
	got := collect(square(generate(-3, 3)))
	want := []int{9, 4, 1, 0, 1, 4, 9}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("pipeline = %v, want %v", got, want)
	}
}

// Проверяем аргументы и защиту от переполнения int.
func TestParseArgs(t *testing.T) {
	maxInt := int(^uint(0) >> 1)

	tests := []struct {
		name         string
		args         []string
		wantK, wantN int
		bad          bool
	}{
		{name: "valid", args: []string{"-3", "3"}, wantK: -3, wantN: 3},
		{name: "zero", args: []string{"0", "0"}},
		{name: "missing", args: []string{"1"}, bad: true},
		{name: "extra", args: []string{"1", "2", "3"}, bad: true},
		{name: "noninteger", args: []string{"abc", "2"}, bad: true},
		{name: "reverse", args: []string{"3", "2"}, bad: true},
		{name: "overflow", args: []string{"0", strconv.Itoa(maxInt)}, bad: true},
		{name: "min int", args: []string{strconv.Itoa(-maxInt - 1), "0"}, bad: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, n, err := parseArgs(tt.args)

			if (err != nil) != tt.bad {
				t.Fatalf("parseArgs(%v) error = %v, want bad=%v",
					tt.args, err, tt.bad)
			}

			if !tt.bad && (k != tt.wantK || n != tt.wantN) {
				t.Errorf("parseArgs(%v) = %d, %d; want %d, %d",
					tt.args, k, n, tt.wantK, tt.wantN)
			}
		})
	}
}

// Проверяем реальный запуск программы с позиционными аргументами.
func TestCLI(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-2", "2")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, output)
	}

	want := "4\n1\n0\n1\n4\n"

	if string(output) != want {
		t.Errorf("output = %q, want %q", output, want)
	}
}

// Ошибки ввода должны давать ненулевой код завершения и понятное сообщение.
func TestCLIInvalidArguments(t *testing.T) {
	tests := [][]string{
		nil,
		{"1"},
		{"x", "2"},
		{"3", "2"},
	}

	for _, args := range tests {
		cmd := exec.Command(
			"go",
			append([]string{"run", "."}, args...)...,
		)

		output, err := cmd.CombinedOutput()

		if err == nil || !strings.Contains(string(output), "invalid arguments:") {
			t.Errorf("args %v: expected error, got %q, %v",
				args, output, err)
		}
	}
}
