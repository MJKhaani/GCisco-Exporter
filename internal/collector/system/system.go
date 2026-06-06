package system

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
	data, err := provider.CollectSystem(ctx)
	if err != nil {
		return err
	}

	c.collector.SetDeviceInfo(device.Name, device.Host, data.Model, data.Version, data.Serial)
	c.collector.SetDeviceUptime(device.Name, device.Host, data.Uptime)
	return nil
}
