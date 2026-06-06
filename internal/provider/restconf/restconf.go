package restconf

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MJKhaani/GCiscoExporter/internal/config"
)

type Provider struct {
	device   *config.Device
	cred     *config.Credential
	client   *http.Client
	baseURL  string
	username string
	password string
}

func New(device *config.Device, cred *config.Credential) *Provider {
	port := 443
	if device.Port != 0 && device.Port != 22 {
		port = device.Port
	}

	transport := &tls.Config{
		InsecureSkipVerify: true,
	}

	return &Provider{
		device:  device,
		cred:    cred,
		baseURL: fmt.Sprintf("https://%s:%d/restconf/data", device.Host, port),
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: &http.Transport{TLSClientConfig: transport},
		},
		username: cred.Username,
		password: cred.Password,
	}
}

func (p *Provider) doRequest(ctx context.Context, path string) (map[string]interface{}, error) {
	url := p.baseURL + path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/yang-data+json")
	req.SetBasicAuth(p.username, p.password)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("RESTCONF request %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("RESTCONF %s returned %d: %s", path, resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode RESTCONF response %s: %w", path, err)
	}
	return result, nil
}

func (p *Provider) CollectSystem(ctx context.Context) (SystemData, error) {
	sys := SystemData{}

	nativeResult, err := p.doRequest(ctx, "/Cisco-IOS-XE-native:native")
	if err != nil {
		return sys, err
	}

	native, ok := nativeResult["Cisco-IOS-XE-native:native"].(map[string]interface{})
	if !ok {
		return sys, fmt.Errorf("unexpected native response format")
	}

	if v, ok := native["version"]; ok {
		sys.Version = fmt.Sprintf("%v", v)
	}
	if v, ok := native["hostname"]; ok {
		sys.Hostname = fmt.Sprintf("%v", v)
	}

	hwResult, err := p.doRequest(ctx, "/Cisco-IOS-XE-device-hardware-oper:device-hardware-data/device-hardware")
	if err == nil {
		if hw, ok := hwResult["Cisco-IOS-XE-device-hardware-oper:device-hardware"].(map[string]interface{}); ok {
			if inventory, ok := hw["device-inventory"].([]interface{}); ok {
				for _, item := range inventory {
					m, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					hwType, _ := m["hw-type"].(string)
					if hwType == "hw-type-chassis" {
						if v, ok := m["hw-description"].(string); ok {
							sys.Model = v
						}
						if v, ok := m["serial-number"].(string); ok {
							sys.Serial = v
						}
						break
					}
				}
			}
		}
	}

	return sys, nil
}

func (p *Provider) CollectInterfaces(ctx context.Context) ([]InterfaceData, error) {
	result, err := p.doRequest(ctx, "/Cisco-IOS-XE-interfaces-oper:interfaces")
	if err != nil {
		return nil, err
	}

	ifacesRaw, ok := result["Cisco-IOS-XE-interfaces-oper:interfaces"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected interfaces response format")
	}

	ifaceList, ok := ifacesRaw["interface"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no interface list in response")
	}

	var interfaces []InterfaceData
	for _, item := range ifaceList {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := m["name"].(string)
		iface := InterfaceData{Name: name}

		if v, ok := m["admin-status"].(string); ok {
			iface.AdminUp = v == "if-state-up"
		}
		if v, ok := m["oper-status"].(string); ok {
			iface.OperUp = v == "if-oper-state-up" || v == "if-oper-state-ready"
		}

		if stats, ok := m["statistics"].(map[string]interface{}); ok {
			if v, ok := stats["in-octets"].(string); ok {
				iface.RxBytes, _ = strconv.ParseFloat(v, 64)
			} else if v, ok := stats["in-octets"].(float64); ok {
				iface.RxBytes = v
			}
			if v, ok := stats["out-octets"].(string); ok {
				iface.TxBytes, _ = strconv.ParseFloat(v, 64)
			} else if v, ok := stats["out-octets"].(float64); ok {
				iface.TxBytes = v
			}
			if v, ok := stats["in-errors"].(float64); ok {
				iface.RxErrors = v
			}
			if v, ok := stats["out-errors"].(float64); ok {
				iface.TxErrors = v
			}
		}

		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}

func (p *Provider) CollectResources(ctx context.Context) (ResourceData, error) {
	var res ResourceData

	cpuResult, err := p.doRequest(ctx, "/Cisco-IOS-XE-process-cpu-oper:cpu-usage/cpu-utilization")
	if err == nil {
		if cpu, ok := cpuResult["Cisco-IOS-XE-process-cpu-oper:cpu-utilization"].(map[string]interface{}); ok {
			if v, ok := cpu["five-seconds"].(float64); ok {
				res.CPUUsage = v
			}
		}
	}

	memResult, err := p.doRequest(ctx, "/Cisco-IOS-XE-memory-oper:memory-statistics")
	if err == nil {
		if mem, ok := memResult["Cisco-IOS-XE-memory-oper:memory-statistics"].(map[string]interface{}); ok {
			if stats, ok := mem["memory-statistic"].([]interface{}); ok {
				for _, s := range stats {
					m, ok := s.(map[string]interface{})
					if !ok {
						continue
					}
					name, _ := m["name"].(string)
					if strings.Contains(strings.ToLower(name), "processor") && !strings.Contains(strings.ToLower(name), "reserve") {
						if v, ok := m["total-memory"].(string); ok {
							res.MemTotal, _ = strconv.ParseFloat(v, 64)
						} else if v, ok := m["total-memory"].(float64); ok {
							res.MemTotal = v
						}
						if v, ok := m["used-memory"].(string); ok {
							res.MemUsed, _ = strconv.ParseFloat(v, 64)
						} else if v, ok := m["used-memory"].(float64); ok {
							res.MemUsed = v
						}
						if v, ok := m["free-memory"].(string); ok {
							res.MemFree, _ = strconv.ParseFloat(v, 64)
						} else if v, ok := m["free-memory"].(float64); ok {
							res.MemFree = v
						}
						break
					}
				}
			}
		}
	}

	return res, nil
}

func (p *Provider) CollectProcesses(ctx context.Context, limit int) ([]ProcessData, error) {
	result, err := p.doRequest(ctx, "/Cisco-IOS-XE-process-cpu-oper:cpu-usage/cpu-utilization")
	if err != nil {
		return nil, err
	}

	cpu, ok := result["Cisco-IOS-XE-process-cpu-oper:cpu-utilization"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected cpu-utilization format")
	}

	procs, ok := cpu["cpu-usage-processes"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	procList, ok := procs["cpu-usage-process"].([]interface{})
	if !ok {
		return nil, nil
	}

	var processes []ProcessData
	for i, item := range procList {
		if i >= limit {
			break
		}
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := m["name"].(string)
		pid := 0
		if v, ok := m["pid"].(float64); ok {
			pid = int(v)
		}

		var cpuPct float64
		if v, ok := m["five-seconds"].(string); ok {
			cpuPct, _ = strconv.ParseFloat(v, 64)
		} else if v, ok := m["five-seconds"].(float64); ok {
			cpuPct = v
		}

		var runtime float64
		if v, ok := m["total-run-time"].(string); ok {
			runtime, _ = strconv.ParseFloat(v, 64)
		} else if v, ok := m["total-run-time"].(float64); ok {
			runtime = v
		}

		processes = append(processes, ProcessData{
			Name:    fmt.Sprintf("%s(%d)", name, pid),
			CPU:     cpuPct,
			Runtime: runtime,
		})
	}

	return processes, nil
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
