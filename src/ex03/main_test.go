package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Проверяем отсутствие аргумента, ноль, отрицательное и нечисловое значения.
func TestParseInterval(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    uint
		invalid bool
	}{
		{name: "one second", args: []string{"1"}, want: 1},
		{name: "large value", args: []string{"300"}, want: 300},
		{name: "missing", invalid: true},
		{name: "extra", args: []string{"1", "2"}, invalid: true},
		{name: "zero", args: []string{"0"}, invalid: true},
		{name: "negative", args: []string{"-1"}, invalid: true},
		{name: "not a number", args: []string{"abc"}, invalid: true},
		{name: "overflow uint", args: []string{"18446744073709551616"}, invalid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInterval(tt.args)

			if (err != nil) != tt.invalid {
				t.Fatalf(
					"parseInterval(%v) error = %v, invalid = %v",
					tt.args, err, tt.invalid,
				)
			}

			if !tt.invalid && got != tt.want {
				t.Errorf("got interval %d, want %d", got, tt.want)
			}
		})
	}
}

// Убеждаемся, что тикер создаёт тики с нужными номерами и завершается после stop.
func TestStartTicker(t *testing.T) {
	stop := make(chan struct{})
	ticks, done := startTicker(1, stop)

	t.Cleanup(func() {
		close(stop)
		<-done
	})

	deadline := make(chan struct{})
	go func() {
		time.Sleep(4 * time.Second)
		close(deadline)
	}()

	for expected := uint64(1); expected <= 2; expected++ {
		select {
		case got, ok := <-ticks:
			if !ok {
				t.Fatal("ticker closed before the expected tick")
			}

			if got.number != expected || got.seconds != expected {
				t.Errorf(
					"got tick %+v, want number %d and %d seconds",
					got, expected, expected,
				)
			}

		case <-deadline:
			t.Fatal("ticker did not deliver two ticks within four seconds")
		}
	}
}

// Если прервать тикер до первого срабатывания, он должен закрыть свои каналы.
func TestStopTicker(t *testing.T) {
	stop := make(chan struct{})
	ticks, done := startTicker(100, stop)
	close(stop)

	deadline := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)
		close(deadline)
	}()

	select {
	case <-done:
	case <-deadline:
		t.Fatal("ticker did not stop")
	}

	if _, ok := <-ticks; ok {
		t.Fatal("ticker sent a value after stop")
	}
}

// Сигнал посылаем непосредственно бинарнику: go run запускает промежуточный процесс.
func TestTerminationSignals(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.Process.Signal with SIGINT/SIGTERM is not supported on Windows")
	}

	binary := filepath.Join(t.TempDir(), "ticker")

	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, output)
	}

	for _, test := range []struct {
		name   string
		signal os.Signal
	}{
		{name: "SIGINT", signal: os.Interrupt},
		{name: "SIGTERM", signal: syscall.SIGTERM},
	} {
		t.Run(test.name, func(t *testing.T) {
			cmd := exec.Command(binary, "1")

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}

			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				if cmd.ProcessState == nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			})

			lines := make(chan string, 4)

			go func() {
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					lines <- scanner.Text()
				}
				close(lines)
			}()

			deadline := make(chan struct{})

			go func() {
				time.Sleep(5 * time.Second)
				close(deadline)
			}()

			select {
			case line := <-lines:
				if line != "Tick 1 since 1" {
					t.Fatalf(
						"first line = %q, want Tick 1 since 1",
						line,
					)
				}

			case <-deadline:
				t.Fatal("no first tick before timeout")
			}

			if err := cmd.Process.Signal(test.signal); err != nil {
				t.Fatalf("failed to send %s: %v", test.name, err)
			}

			select {
			case line := <-lines:
				if line != "Termination" {
					t.Fatalf(
						"line after signal = %q, want Termination",
						line,
					)
				}

			case <-deadline:
				t.Fatal("process did not handle termination signal")
			}

			if err := cmd.Wait(); err != nil {
				t.Errorf("process terminated with an error: %v", err)
			}
		})
	}
}

// Некорректные аргументы должны печатать ошибку в stderr и возвращать ненулевой код.
func TestCLIInvalidArguments(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"0"},
		{"-1"},
		{"x"},
		{"1", "2"},
	} {
		cmd := exec.Command(
			"go",
			append([]string{"run", "."}, args...)...,
		)

		output, err := cmd.CombinedOutput()

		if err == nil || !strings.Contains(string(output), "invalid arguments:") {
			t.Errorf(
				"args %v: got %q and error %v",
				args, output, err,
			)
		}
	}
}
