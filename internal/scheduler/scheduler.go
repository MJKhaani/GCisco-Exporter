package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/MJKhaani/GCiscoExporter/internal/cache"
	"github.com/MJKhaani/GCiscoExporter/internal/config"
	"github.com/MJKhaani/GCiscoExporter/internal/debugcapture"
	"github.com/MJKhaani/GCiscoExporter/internal/metrics"
	"github.com/MJKhaani/GCiscoExporter/internal/provider/json"
	"github.com/MJKhaani/GCiscoExporter/internal/provider/ssh"
	"github.com/MJKhaani/GCiscoExporter/internal/worker"
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

	timeout := s.cfg.TimeoutDuration()

	for _, method := range device.CollectionMethods {
		var providerName string
		var result map[string]interface{}
		var err error

		switch method {
		case "json":
			providerName = "json"
			result, err = s.collectJSON(ctx, device, cred, timeout)
		case "ssh":
			providerName = "ssh"
			result, err = s.collectSSH(ctx, device, cred, timeout)
		case "restconf":
			providerName = "restconf"
			err = fmt.Errorf("RESTCONF not yet implemented")
		case "netconf":
			providerName = "netconf"
			err = fmt.Errorf("NETCONF not yet implemented")
		default:
			err = fmt.Errorf("unknown collection method: %s", method)
		}

		if err != nil {
			log.Printf("Collection method %s failed for %s: %v", method, device.Name, err)
			continue
		}

		s.store.Update(device.Name, result)
		s.collector.SetDeviceUp(device.Name, device.Host, device.Type, true)
		s.collector.SetCollectionMethod(device.Name, device.Host, s.methodToFloat(providerName))
		return nil
	}

	s.collector.SetDeviceUp(device.Name, device.Host, device.Type, false)
	return fmt.Errorf("all collection methods failed for %s", device.Name)
}

func (s *Scheduler) collectJSON(ctx context.Context, device *config.Device, cred *config.Credential, timeout time.Duration) (map[string]interface{}, error) {
	client, err := ssh.NewClient(device, cred, timeout, s.cfg.LegacyCiphers)
	if err != nil {
		return nil, fmt.Errorf("SSH connect: %w", err)
	}
	defer client.Close()

	provider := json.New(client, device)

	result := make(map[string]interface{})

	sys, err := provider.CollectSystem(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect system: %w", err)
	}
	s.collector.SetDeviceInfo(device.Name, device.Host, sys.Model, sys.Version, sys.Serial)
	s.collector.SetDeviceUptime(device.Name, device.Host, sys.Uptime)
	result["system"] = sys

	ifaces, err := provider.CollectInterfaces(ctx)
	if err != nil {
		log.Printf("Warning: collect interfaces (JSON) failed for %s: %v", device.Name, err)
	}
	for _, iface := range ifaces {
		s.collector.SetInterfaceAdminUp(device.Name, device.Host, iface.Name, iface.AdminUp)
		s.collector.SetInterfaceOperUp(device.Name, device.Host, iface.Name, iface.OperUp)
		s.collector.SetInterfaceRxBytes(device.Name, device.Host, iface.Name, iface.RxBytes)
		s.collector.SetInterfaceTxBytes(device.Name, device.Host, iface.Name, iface.TxBytes)
		s.collector.SetInterfaceRxErrors(device.Name, device.Host, iface.Name, iface.RxErrors)
		s.collector.SetInterfaceTxErrors(device.Name, device.Host, iface.Name, iface.TxErrors)
	}
	result["interfaces"] = ifaces

	res, err := provider.CollectResources(ctx)
	if err != nil {
		log.Printf("Warning: collect resources (JSON) failed for %s: %v", device.Name, err)
	} else {
		s.collector.SetCPUUsage(device.Name, device.Host, res.CPUUsage)
		s.collector.SetMemoryTotal(device.Name, device.Host, res.MemTotal)
		s.collector.SetMemoryUsed(device.Name, device.Host, res.MemUsed)
		s.collector.SetMemoryFree(device.Name, device.Host, res.MemFree)
	}
	result["resources"] = res

	procs, err := provider.CollectProcesses(ctx, device.ProcessLimit)
	if err != nil {
		log.Printf("Warning: collect processes (JSON) failed for %s: %v", device.Name, err)
	} else {
		for _, proc := range procs {
			s.collector.SetProcessCPU(device.Name, device.Host, proc.Name, proc.CPU)
			s.collector.SetProcessRuntime(device.Name, device.Host, proc.Name, proc.Runtime)
		}
	}
	result["processes"] = procs

	return result, nil
}

func (s *Scheduler) collectSSH(ctx context.Context, device *config.Device, cred *config.Credential, timeout time.Duration) (map[string]interface{}, error) {
	client, err := ssh.NewClient(device, cred, timeout, s.cfg.LegacyCiphers)
	if err != nil {
		return nil, fmt.Errorf("SSH connect: %w", err)
	}
	defer client.Close()

	result := make(map[string]interface{})

	ver, err := client.Execute(ctx, "show version")
	if err != nil {
		return nil, fmt.Errorf("show version: %w", err)
	}
	result["version_raw"] = ver

	ifaces, err := client.Execute(ctx, "show interface")
	if err != nil {
		return nil, fmt.Errorf("show interface: %w", err)
	}
	result["interfaces_raw"] = ifaces

	res, err := client.Execute(ctx, "show processes cpu")
	if err != nil {
		log.Printf("Warning: show processes cpu failed for %s: %v", device.Name, err)
	}
	result["processes_raw"] = res

	env, err := client.Execute(ctx, "show environment")
	if err != nil {
		log.Printf("Warning: show environment failed for %s: %v", device.Name, err)
	}
	result["environment_raw"] = env

	return result, nil
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
