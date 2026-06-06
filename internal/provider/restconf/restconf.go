package restconf

import (
	"context"
	"fmt"

	"github.com/MJKhaani/GCiscoExporter/internal/config"
)

type Provider struct {
	device *config.Device
	cred   *config.Credential
}

func New(device *config.Device, cred *config.Credential) *Provider {
	return &Provider{device: device, cred: cred}
}

func (p *Provider) Collect(ctx context.Context) (map[string]interface{}, error) {
	return nil, fmt.Errorf("RESTCONF provider not yet implemented")
}
