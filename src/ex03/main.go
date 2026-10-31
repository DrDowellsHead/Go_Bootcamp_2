
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// tick содержит порядковый номер срабатывания и прошедшее число секунд.
type tick struct {
	number  uint64
	seconds uint64
}

// parseInterval проверяет единственный аргумент — положительный шаг в секундах.
func parseInterval(args []string) (uint, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("expected one argument: K")
	}

	value, err := strconv.ParseUint(args[0], 10, 0)
	if err != nil || value == 0 {
		return 0, fmt.Errorf("K must be a positive uint")
	}

	return uint(value), nil
}

// startTicker запускает горутину и возвращает канал тиков и сигнал её завершения.
func startTicker(interval uint, stop <-chan struct{}) (<-chan tick, <-chan struct{}) {
	ticks := make(chan tick)
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer close(ticks)

		var seconds uint64

		for {
			// Проверяем остановку каждые 100 мс, чтобы не ждать весь интервал K.
			for i := 0; i < 10; i++ {
				select {
				case <-stop:
					return
				default:
				}

				time.Sleep(100 * time.Millisecond)
			}

			select {
			case <-stop:
				return
			default:
			}

			seconds++

			if seconds%uint64(interval) != 0 {
				continue
			}

			// Отправка может ждать main; остановка доступна и во время ожидания.
			select {
			case ticks <- tick{
				number:  seconds / uint64(interval),
				seconds: seconds,
			}:
			case <-stop:
				return
			}
		}
	}()

	return ticks, done
}

func main() {
	interval, err := parseInterval(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid arguments:", err)
		os.Exit(1)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	stop := make(chan struct{})
	ticks, done := startTicker(interval, stop)

	for {
		select {
		case <-signals:
			// Останавливаем горутину, дожидаемся выхода и только затем завершаемся.
			close(stop)
			<-done

			fmt.Println("Termination")
			return

		case value, ok := <-ticks:
			if !ok {
				return
			}

			fmt.Printf("Tick %d since %d\n", value.number, value.seconds)
		}
	}
}
