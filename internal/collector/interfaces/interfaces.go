package interfaces

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
	data, err := provider.CollectInterfaces(ctx)
	if err != nil {
		return err
	}

	for _, iface := range data {
		c.collector.SetInterfaceAdminUp(device.Name, device.Host, iface.Name, iface.AdminUp)
		c.collector.SetInterfaceOperUp(device.Name, device.Host, iface.Name, iface.OperUp)
		c.collector.SetInterfaceRxBytes(device.Name, device.Host, iface.Name, iface.RxBytes)
		c.collector.SetInterfaceTxBytes(device.Name, device.Host, iface.Name, iface.TxBytes)
		c.collector.SetInterfaceRxErrors(device.Name, device.Host, iface.Name, iface.RxErrors)
		c.collector.SetInterfaceTxErrors(device.Name, device.Host, iface.Name, iface.TxErrors)
	}
	return nil
}
