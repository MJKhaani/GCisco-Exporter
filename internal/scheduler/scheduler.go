package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mjkoot/GCiscoExporter/internal/cache"
	"github.com/mjkoot/GCiscoExporter/internal/config"
	"github.com/mjkoot/GCiscoExporter/internal/debugcapture"
	"github.com/mjkoot/GCiscoExporter/internal/metrics"
	"github.com/mjkoot/GCiscoExporter/internal/worker"
)

type Scheduler struct {
	cfg       *config.Config
	store     *cache.Store
	collector *metrics.Collector
	debugCap  *debugcapture.Capture
	worker    *worker.Pool
}

func New(cfg *config.Config, store *cache.Store, collector *metrics.Collector, debugCap *debugcapture.Capture) *Scheduler {
	return &Scheduler{
		cfg:       cfg,
		store:     store,
		collector: collector,
		debugCap:  debugCap,
		worker:    worker.New(cfg.Workers, cfg),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	interval, err := time.ParseDuration(s.cfg.CollectionInterval)
	if err != nil {
		log.Printf("Invalid collection interval %q, defaulting to 60s", s.cfg.CollectionInterval)
		interval = 60 * time.Second
	}

	go func() {
		s.runCollection(ctx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runCollection(ctx)
			}
		}
	}()
}

func (s *Scheduler) runCollection(ctx context.Context) {
	log.Printf("Starting collection cycle for %d devices", len(s.cfg.Devices))

	start := time.Now()
	results := make(map[string]bool)
	var mu sync.Mutex

	for _, device := range s.cfg.Devices {
		select {
		case <-ctx.Done():
			return
		default:
		}

		d := device
		s.worker.Submit(ctx, func() {
			deviceStart := time.Now()
			err := s.collectDevice(ctx, &d)

			mu.Lock()
			results[d.Name] = err == nil
			mu.Unlock()

			duration := time.Since(deviceStart).Seconds()
			s.collector.SetCollectionDuration(d.Name, d.Host, duration)
			s.collector.SetCollectionSuccess(d.Name, d.Host, err == nil)

			if err != nil {
				log.Printf("Failed to collect from %s (%s): %v", d.Name, d.Host, err)
			} else {
				s.collector.SetLastCollection(d.Name, d.Host, float64(time.Now().Unix()))
			}
		})
	}

	s.worker.Wait()

	totalDuration := time.Since(start).Seconds()
	successCount := 0
	for _, ok := range results {
		if ok {
			successCount++
		}
	}
	log.Printf("Collection cycle complete: %d/%d devices successful in %.2fs", successCount, len(s.cfg.Devices), totalDuration)
}

func (s *Scheduler) collectDevice(ctx context.Context, device *config.Device) error {
	cred := s.findCredential(device.CredentialName)
	if cred == nil {
		return fmt.Errorf("credential %q not found for device %s", device.CredentialName, device.Name)
	}

	for _, method := range device.CollectionMethods {
		result, provider, err := s.tryCollect(ctx, device, cred, method)
		if err != nil {
			log.Printf("Collection method %s failed for %s: %v", method, device.Name, err)
			continue
		}

		s.store.Update(device.Name, result)
		s.collector.SetDeviceUp(device.Name, device.Host, device.Type, true)
		s.collector.SetCollectionMethod(device.Name, device.Host, s.methodToFloat(provider))
		return nil
	}

	s.collector.SetDeviceUp(device.Name, device.Host, device.Type, false)
	return fmt.Errorf("all collection methods failed for %s", device.Name)
}

func (s *Scheduler) tryCollect(ctx context.Context, device *config.Device, cred *config.Credential, method string) (map[string]interface{}, string, error) {
	return nil, "", fmt.Errorf("not implemented: %s", method)
}

func (s *Scheduler) findCredential(name string) *config.Credential {
	for i := range s.cfg.Credentials {
		if s.cfg.Credentials[i].Name == name {
			return &s.cfg.Credentials[i]
		}
	}
	if name == "" && len(s.cfg.Credentials) > 0 {
		return &s.cfg.Credentials[0]
	}
	return nil
}

func (s *Scheduler) methodToFloat(method string) float64 {
	switch method {
	case "json":
		return 1
	case "restconf":
		return 2
	case "netconf":
		return 3
	case "ssh":
		return 4
	default:
		return 0
	}
}
