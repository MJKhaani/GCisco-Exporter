package processes

import (
	"context"

	"github.com/mjkoot/GCiscoExporter/internal/config"
	"github.com/mjkoot/GCiscoExporter/internal/metrics"
	"github.com/mjkoot/GCiscoExporter/internal/provider/json"
)

type Collector struct {
	collector *metrics.Collector
}

func New(collector *metrics.Collector) *Collector {
	return &Collector{collector: collector}
}

func (c *Collector) Collect(ctx context.Context, device *config.Device, provider *json.Provider) error {
	limit := device.ProcessLimit
	if limit == 0 {
		limit = 10
	}

	data, err := provider.CollectProcesses(ctx, limit)
	if err != nil {
		return err
	}

	for _, proc := range data {
		c.collector.SetProcessCPU(device.Name, device.Host, proc.Name, proc.CPU)
		c.collector.SetProcessRuntime(device.Name, device.Host, proc.Name, proc.Runtime)
	}
	return nil
}
