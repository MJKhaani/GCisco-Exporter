package cache

import (
	"sync"
	"time"
)

type DeviceMetrics struct {
	Timestamp time.Time
	Data      map[string]interface{}
}

type Store struct {
	mu      sync.RWMutex
	metrics map[string]*DeviceMetrics
}

func New() *Store {
	return &Store{
		metrics: make(map[string]*DeviceMetrics),
	}
}

func (s *Store) Update(device string, data map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics[device] = &DeviceMetrics{
		Timestamp: time.Now(),
		Data:      data,
	}
}

func (s *Store) Get(device string) (*DeviceMetrics, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.metrics[device]
	return m, ok
}

func (s *Store) GetAll() map[string]*DeviceMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*DeviceMetrics, len(s.metrics))
	for k, v := range s.metrics {
		result[k] = v
	}
	return result
}
