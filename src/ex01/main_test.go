package main

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// Проверяем количество задач, номера, границы задержки и порядок сортировки.
func TestRunTasks(t *testing.T) {
	tests := []struct {
		name string
		n    int
		m    int
	}{
		{name: "zero tasks", n: 0, m: 0},
		{name: "one zero delay", n: 1, m: 0},
		{name: "all zero delays", n: 20, m: 0},
		{name: "random delays", n: 100, m: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runTasks(tt.n, tt.m)

			if len(got) != tt.n {
				t.Fatalf("got %d tasks, want %d", len(got), tt.n)
			}

			seen := make([]bool, tt.n)

			for i, result := range got {
				if result.id < 0 || result.id >= tt.n {
					t.Fatalf("invalid goroutine number %d", result.id)
				}

				if seen[result.id] {
					t.Fatalf("duplicate goroutine number %d", result.id)
				}

				seen[result.id] = true

				if result.milliseconds < 0 || result.milliseconds > tt.m {
					t.Errorf("sleep time %d outside [0, %d]", result.milliseconds, tt.m)
				}

				if i > 0 && got[i-1].milliseconds < result.milliseconds {
					t.Errorf("results are not sorted by descending sleep time")
				}

				if i > 0 && got[i-1].milliseconds == result.milliseconds &&
					got[i-1].id > result.id {
					t.Errorf("equal sleep times are not sorted by goroutine number")
				}
			}
		})
	}
}

// Запускаем настоящую CLI-программу, чтобы проверить ввод и печать, а не только runTasks.
func TestCLI(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-n", "8", "-m", "10")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, output)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	if len(lines) != 8 {
		t.Fatalf("got %d output lines, want 8: %q", len(lines), output)
	}

	seen := make([]bool, 8)
	previous := 11

	for _, line := range lines {
		fields := strings.Fields(line)

		if len(fields) != 2 {
			t.Fatalf("expected two numbers, got %q", line)
		}

		id, errID := strconv.Atoi(fields[0])
		ms, errMS := strconv.Atoi(fields[1])

		if errID != nil || errMS != nil ||
			id < 0 || id >= 8 || ms < 0 || ms > 10 {
			t.Fatalf("invalid output line %q", line)
		}

		if seen[id] || ms > previous {
			t.Fatalf("duplicate id or incorrect sorting in %q", line)
		}

		seen[id] = true
		previous = ms
	}
}

// При нуле задач вывода быть не должно: мы действительно запускаем ровно N горутин.
func TestCLIZeroTasks(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-n", "0", "-m", "10")

	output, err := cmd.CombinedOutput()

	if err != nil || len(output) != 0 {
		t.Fatalf("expected empty output and successful exit, got %q, error %v", output, err)
	}
}

// Невалидные параметры должны приводить к ошибке, а не к панике или молчаливому запуску.
func TestCLIInvalidArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing n", args: []string{"-m", "10"}},
		{name: "missing m", args: []string{"-n", "5"}},
		{name: "negative n", args: []string{"-n", "-5", "-m", "10"}},
		{name: "negative m", args: []string{"-n", "5", "-m", "-1"}},
		{name: "non integer", args: []string{"-n", "abc", "-m", "10"}},
		{name: "extra argument", args: []string{"-n", "5", "-m", "10", "extra"}},
		{name: "overflow duration", args: []string{"-n", "1", "-m", "9223372036855"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(
				"go",
				append([]string{"run", "."}, tt.args...)...,
			)

			output, err := cmd.CombinedOutput()

			if err == nil {
				t.Fatalf("expected nonzero exit code, got: %q", output)
			}

			if !strings.Contains(string(output), "invalid arguments") &&
				!strings.Contains(string(output), "invalid value") {
				t.Errorf("expected an argument validation error, got: %q", output)
			}

			if strings.Contains(string(output), "panic:") {
				t.Errorf("invalid arguments caused a panic: %q", output)
			}
		})
	}
}
