package json

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/MJKhaani/GCiscoExporter/internal/config"
	"github.com/MJKhaani/GCiscoExporter/internal/provider/ssh"
)

type Provider struct {
	client *ssh.Client
	device *config.Device
}

func New(client *ssh.Client, device *config.Device) *Provider {
	return &Provider{client: client, device: device}
}

func (p *Provider) Collect(ctx context.Context) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	ver, err := p.client.Execute(ctx, "show version | json")
	if err != nil {
		return nil, fmt.Errorf("show version: %w", err)
	}
	result["version"] = p.parseJSON(ver)

	iface, err := p.client.Execute(ctx, "show interface | json")
	if err != nil {
		return nil, fmt.Errorf("show interface: %w", err)
	}
	result["interfaces"] = p.parseJSON(iface)

	env, err := p.client.Execute(ctx, "show environment | json")
	if err != nil {
		return nil, fmt.Errorf("show environment: %w", err)
	}
	result["environment"] = p.parseJSON(env)

	res, err := p.client.Execute(ctx, "show system resources | json")
	if err != nil {
		return nil, fmt.Errorf("show system resources: %w", err)
	}
	result["resources"] = p.parseJSON(res)

	proc, err := p.client.Execute(ctx, "show processes cpu | json")
	if err != nil {
		return nil, fmt.Errorf("show processes cpu: %w", err)
	}
	result["processes"] = p.parseJSON(proc)

	return result, nil
}

func (p *Provider) parseJSON(raw string) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return map[string]interface{}{"raw": raw, "parse_error": err.Error()}
	}
	return result
}

func (p *Provider) CollectSystem(ctx context.Context) (SystemData, error) {
	ver, err := p.client.Execute(ctx, "show version | json")
	if err != nil {
		return SystemData{}, err
	}
	data := p.parseJSON(ver)
	return SystemData{
		Hostname:  p.stringField(data, "host_name"),
		Version:   p.stringField(data, "nxos_ver_str"),
		Model:     p.stringField(data, "chassis_id"),
		Serial:    p.stringField(data, "proc_board_id"),
		Uptime:    p.parseIntField(data, "uptime"),
	}, nil
}

func (p *Provider) CollectInterfaces(ctx context.Context) ([]InterfaceData, error) {
	iface, err := p.client.Execute(ctx, "show interface | json")
	if err != nil {
		return nil, err
	}
	data := p.parseJSON(iface)

	var interfaces []InterfaceData
	if table, ok := data["TABLE_interface"]; ok {
		if rows, ok := table.(map[string]interface{})["ROW_interface"].([]interface{}); ok {
			for _, row := range rows {
				if m, ok := row.(map[string]interface{}); ok {
					interfaces = append(interfaces, InterfaceData{
						Name:      p.stringField(m, "interface"),
						AdminUp:   p.stringField(m, "admin_state") == "up",
						OperUp:    p.stringField(m, "state") == "up",
						RxBytes:   p.parseIntField(m, "eth_inbytes"),
						TxBytes:   p.parseIntField(m, "eth_outbytes"),
						RxErrors:  p.parseIntField(m, "eth_inerr"),
						TxErrors:  p.parseIntField(m, "eth_outerr"),
					})
				}
			}
		}
	}
	return interfaces, nil
}

func (p *Provider) CollectEnvironment(ctx context.Context) (EnvironmentData, error) {
	env, err := p.client.Execute(ctx, "show environment | json")
	if err != nil {
		return EnvironmentData{}, err
	}
	data := p.parseJSON(env)
	return EnvironmentData{
		Raw: data,
	}, nil
}

func (p *Provider) CollectResources(ctx context.Context) (ResourceData, error) {
	res, err := p.client.Execute(ctx, "show system resources | json")
	if err != nil {
		return ResourceData{}, err
	}
	data := p.parseJSON(res)
	return ResourceData{
		CPUUsage: p.parseFloatField(data, "cpu_usage"),
		MemTotal: p.parseIntField(data, "memory_usage_total"),
		MemUsed:  p.parseIntField(data, "memory_usage_used"),
		MemFree:  p.parseIntField(data, "memory_usage_free"),
	}, nil
}

func (p *Provider) CollectProcesses(ctx context.Context, limit int) ([]ProcessData, error) {
	proc, err := p.client.Execute(ctx, "show processes cpu | json")
	if err != nil {
		return nil, err
	}
	data := p.parseJSON(proc)

	var processes []ProcessData
	if table, ok := data["TABLE_process_cpu"]; ok {
		if rows, ok := table.(map[string]interface{})["ROW_process_cpu"].([]interface{}); ok {
			for i, row := range rows {
				if i >= limit {
					break
				}
				if m, ok := row.(map[string]interface{}); ok {
					processes = append(processes, ProcessData{
						Name:     p.stringField(m, "process"),
						CPU:      p.parseFloatField(m, "cpu_percent"),
						Runtime:  p.parseIntField(m, "runtime"),
					})
				}
			}
		}
	}
	return processes, nil
}

func (p *Provider) stringField(data map[string]interface{}, key string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (p *Provider) parseIntField(data map[string]interface{}, key string) float64 {
	if v, ok := data[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case string:
			if i, err := strconv.ParseFloat(n, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

func (p *Provider) parseFloatField(data map[string]interface{}, key string) float64 {
	return p.parseIntField(data, key)
}

type SystemData struct {
	Hostname string
	Version  string
	Model    string
	Serial   string
	Uptime   float64
}

type InterfaceData struct {
	Name     string
	AdminUp  bool
	OperUp   bool
	RxBytes  float64
	TxBytes  float64
	RxErrors float64
	TxErrors float64
}

type EnvironmentData struct {
	Raw map[string]interface{}
}

type ResourceData struct {
	CPUUsage float64
	MemTotal float64
	MemUsed  float64
	MemFree  float64
}

type ProcessData struct {
	Name    string
	CPU     float64
	Runtime float64
}
