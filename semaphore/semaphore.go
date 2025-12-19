package semaphore

import (
	"fmt"
	"sync"
	"time"
)

// CountingSemaphore — структура семафора подсчёта
type CountingSemaphore struct {
	// Канал для управления доступом к ресурсу
	sem chan struct{}
	// Mutex для безопасного доступа к счетчику разрешений
	mutex sync.Mutex
	// Общее количество разрешений
	totalPermits int
	// Время ожидания основных операций с семафором, чтобы не 
	// блокировать операции с ним навечно (необязательное требование, зависит от 
	// нужд программы)
	timeout time.Duration
}

// Acquire — метод захвата семафора (уменьшает счётчик)
func (cs *CountingSemaphore) Acquire() error {
	select {
	case cs.sem <- struct{}{}:
		return nil
	case <-time.After(cs.timeout):
		return fmt.Errorf("Не удалось захватить семафор: таймаут")
	}
}

// Release — метод освобождения семафора (увеличивает счётчик)
func (cs *CountingSemaphore) Release() error {
	select {
	case <-cs.sem:
		return nil
	case <-time.After(cs.timeout):
		return fmt.Errorf("Не удалось освободить семафор: таймаут")
	}
}

// TryAcquire — метод попытки немедленного захвата семафора без ожидания
func (cs *CountingSemaphore) TryAcquire() bool {
	select {
	case cs.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

// TryRelease — метод попытки немедленного освобождения семафора без ожидания
func (cs *CountingSemaphore) TryRelease() bool {
	select {
	case <-cs.sem:
		return true
	default:
		return false
	}
}

// AvailablePermits — метод получения количества доступных разрешений
func (cs *CountingSemaphore) AvailablePermits() int {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	return cap(cs.sem) - len(cs.sem)
}

// TotalPermits — метод получения общего количества разрешений
func (cs *CountingSemaphore) TotalPermits() int {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	return cs.totalPermits
}

// AcquireN — метод захвата N разрешений у семафора
func (cs *CountingSemaphore) AcquireN(n int) error {
	if n <= 0 {
		return fmt.Errorf("нельзя захватить %d разрешений", n)
	}

	for i := 0; i < n; i++ {
		select {
		case cs.sem <- struct{}{}:
			continue
		case <-time.After(cs.timeout):
			// Если не удалось захватить все разрешения, освобождаем уже захваченные
			for j := 0; j < i; j++ {
				<-cs.sem
			}
			return fmt.Errorf("не удалось захватить %d разрешений: таймаут", n)
		}
	}
	return nil
}

// ReleaseN — метод освобождения N разрешений у семафора
func (cs *CountingSemaphore) ReleaseN(n int) error {
	if n <= 0 {
		return fmt.Errorf("нельзя освободить %d разрешений", n)
	}

	// Проверяем, что мы не освобождаем больше разрешений, чем захвачено
	currentUsed := len(cs.sem)
	if n > currentUsed {
		return fmt.Errorf("нельзя освободить %d разрешений, только %d захвачено", n, currentUsed)
	}

	for i := 0; i < n; i++ {
		select {
		case <-cs.sem:
			continue
		case <-time.After(cs.timeout):
			return fmt.Errorf("не удалось освободить %d разрешений: таймаут", n)
		}
	}
	return nil
}

// NewCountingSemaphore — функция создания семафора подсчёта с указанным количеством разрешений
func NewCountingSemaphore(permits int, timeout time.Duration) *CountingSemaphore {
	if permits <= 0 {
		panic("Количество разрешений должно быть положительным")
	}

	return &CountingSemaphore{
		sem:          make(chan struct{}, permits), // Буферизованный канал размером permits
		totalPermits: permits,
		timeout:      timeout,
	}
}