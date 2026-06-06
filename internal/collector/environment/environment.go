package environment

import (
	"context"

	"github.com/MJKhaani/GCiscoExporter/internal/config"
	"github.com/MJKhaani/GCiscoExporter/internal/metrics"
	"github.com/MJKhaani/GCiscoExporter/internal/provider/json"
)

type Collector struct {
	collector *metrics.Collector
}

func New(collector *metrics.Collector) *Collector {
	return &Collector{collector: collector}
}

func (c *Collector) Collect(ctx context.Context, device *config.Device, provider *json.Provider) error {
	data, err := provider.CollectResources(ctx)
	if err != nil {
		return err
	}

	c.collector.SetCPUUsage(device.Name, device.Host, data.CPUUsage)
	c.collector.SetMemoryTotal(device.Name, device.Host, data.MemTotal)
	c.collector.SetMemoryUsed(device.Name, device.Host, data.MemUsed)
	c.collector.SetMemoryFree(device.Name, device.Host, data.MemFree)
	return nil
}
