package optics

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
	return nil
}
