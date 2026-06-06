package capabilities

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mjkoot/GCiscoExporter/internal/config"
	"github.com/mjkoot/GCiscoExporter/internal/provider/ssh"
)

type DeviceCapabilities struct {
	Type             string
	SupportsJSON     bool
	SupportsRESTCONF bool
	SupportsNETCONF  bool
	Version          string
	Model            string
}

func Detect(ctx context.Context, device *config.Device, cred *config.Credential) (*DeviceCapabilities, error) {
	timeout := 10
	client, err := ssh.NewClient(device, cred, time.Duration(timeout)*time.Second)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	caps := &DeviceCapabilities{}

	verOutput, err := client.Execute(ctx, "show version")
	if err != nil {
		return nil, fmt.Errorf("show version failed: %w", err)
	}

	caps.parseVersion(verOutput)

	switch {
	case strings.Contains(verOutput, "NX-OS"):
		caps.Type = "nxos"
		caps.SupportsJSON = true
	case strings.Contains(verOutput, "IOS-XE"):
		caps.Type = "iosxe"
		caps.SupportsRESTCONF = true
		caps.SupportsNETCONF = true
	case strings.Contains(verOutput, "IOS"):
		caps.Type = "ios"
	}

	return caps, nil
}

func (c *DeviceCapabilities) parseVersion(output string) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "system:") || strings.HasPrefix(line, "System version:") {
			c.Version = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
		if strings.Contains(line, "cisco") && strings.Contains(line, "Chassis") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				c.Model = parts[1]
			}
		}
	}
}
