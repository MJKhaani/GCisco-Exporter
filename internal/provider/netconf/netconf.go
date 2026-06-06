package netconf

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MJKhaani/GCiscoExporter/internal/config"
	"golang.org/x/crypto/ssh"
)

type Provider struct {
	device  *config.Device
	cred    *config.Credential
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
}

func New(device *config.Device, cred *config.Credential) *Provider {
	return &Provider{
		device: device,
		cred:   cred,
	}
}

func (p *Provider) Connect(ctx context.Context) error {
	sshConfig := &ssh.ClientConfig{
		User:            p.cred.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	if p.cred.KeyFile != "" {
		key, err := readKey(p.cred.KeyFile)
		if err != nil {
			return fmt.Errorf("reading SSH key: %w", err)
		}
		sshConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(key)}
	} else if p.cred.Password != "" {
		sshConfig.Auth = []ssh.AuthMethod{ssh.Password(p.cred.Password)}
	} else {
		return fmt.Errorf("no authentication method provided")
	}

	port := 830
	addr := fmt.Sprintf("%s:%d", p.device.Host, port)

	var err error
	p.client, err = ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("NETCONF dial %s: %w", addr, err)
	}

	session, err := p.client.NewSession()
	if err != nil {
		p.client.Close()
		return fmt.Errorf("creating NETCONF session: %w", err)
	}

	p.stdin, _ = session.StdinPipe()
	p.stdout, _ = session.StdoutPipe()

	if err := session.RequestSubsystem("netconf"); err != nil {
		session.Close()
		p.client.Close()
		return fmt.Errorf("requesting netconf subsystem: %w", err)
	}

	p.session = session

	return nil
}

func (p *Provider) Close() {
	if p.session != nil {
		p.session.Close()
	}
	if p.client != nil {
		p.client.Close()
	}
}

func (p *Provider) execRPC(ctx context.Context, rpcXML string) (string, error) {
	msg := rpcXML + "]]>]]>"
	if _, err := io.WriteString(p.stdin, msg); err != nil {
		return "", fmt.Errorf("writing NETCONF RPC: %w", err)
	}

	reader := bufio.NewReader(p.stdout)
	var output strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if output.Len() > 0 {
				break
			}
			return "", fmt.Errorf("reading NETCONF response: %w", err)
		}
		output.WriteString(line)
		if strings.Contains(line, "]]>]]>") {
			break
		}
	}

	result := output.String()
	result = strings.TrimSuffix(result, "]]>]]>")
	return result, nil
}

func (p *Provider) getSubtree(ctx context.Context, filterXML string) (string, error) {
	msg := fmt.Sprintf(`<rpc message-id="1" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get>
    <filter type="subtree">%s</filter>
  </rpc>`, filterXML)
	return p.execRPC(ctx, msg)
}

func (p *Provider) CollectSystem(ctx context.Context) (SystemData, error) {
	data, err := p.getSubtree(ctx, `<system-state xmlns="urn:ietf:params:xml:ns:yang:ietf-system"/>`)
	if err != nil {
		return SystemData{}, err
	}

	sys := SystemData{}
	m := parseXMLFields(data)

	if v, ok := m["system-name"].(string); ok {
		sys.Hostname = v
	}
	if v, ok := m["software-version"].(string); ok {
		sys.Version = v
	}

	return sys, nil
}

func (p *Provider) CollectInterfaces(ctx context.Context) ([]InterfaceData, error) {
	data, err := p.getSubtree(ctx, `<interfaces-state xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces"/>`)
	if err != nil {
		return nil, err
	}

	type xmlInterface struct {
		XMLName     xml.Name `xml:"interface"`
		Name        string   `xml:"name"`
		AdminStatus string   `xml:"admin-status"`
		OperStatus  string   `xml:"oper-status"`
		Statistics  struct {
			InOctets  string `xml:"in-octets"`
			OutOctets string `xml:"out-octets"`
			InErrors  string `xml:"in-errors"`
			OutErrors string `xml:"out-errors"`
		} `xml:"statistics"`
	}

	type interfacesState struct {
		XMLName    xml.Name       `xml:"interfaces-state"`
		Interfaces []xmlInterface `xml:"interface"`
	}

	var state interfacesState
	if err := xml.Unmarshal([]byte(data), &state); err != nil {
		return nil, fmt.Errorf("parse NETCONF interfaces: %w", err)
	}

	var interfaces []InterfaceData
	for _, xi := range state.Interfaces {
		iface := InterfaceData{
			Name:    xi.Name,
			AdminUp: xi.AdminStatus == "up",
			OperUp:  xi.OperStatus == "up",
		}
		if v, err := strconv.ParseFloat(xi.Statistics.InOctets, 64); err == nil {
			iface.RxBytes = v
		}
		if v, err := strconv.ParseFloat(xi.Statistics.OutOctets, 64); err == nil {
			iface.TxBytes = v
		}
		if v, err := strconv.ParseFloat(xi.Statistics.InErrors, 64); err == nil {
			iface.RxErrors = v
		}
		if v, err := strconv.ParseFloat(xi.Statistics.OutErrors, 64); err == nil {
			iface.TxErrors = v
		}
		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}

func (p *Provider) CollectResources(ctx context.Context) (ResourceData, error) {
	var res ResourceData

	data, err := p.getSubtree(ctx, `<cpu-usage xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-process-cpu-oper"/>`)
	if err == nil {
		m := parseXMLFields(data)
		if v, ok := m["five-seconds"].(float64); ok {
			res.CPUUsage = v
		}
	}

	data, err = p.getSubtree(ctx, `<memory-statistics xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-memory-oper"/>`)
	if err == nil {
		m := parseXMLFields(data)
		if v, ok := m["total-memory"].(float64); ok {
			res.MemTotal = v
		}
		if v, ok := m["used-memory"].(float64); ok {
			res.MemUsed = v
		}
		if v, ok := m["free-memory"].(float64); ok {
			res.MemFree = v
		}
	}

	return res, nil
}

func (p *Provider) CollectProcesses(ctx context.Context, limit int) ([]ProcessData, error) {
	data, err := p.getSubtree(ctx, `<cpu-usage xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-process-cpu-oper"/>`)
	if err != nil {
		return nil, err
	}

	type xmlProcess struct {
		XMLName  xml.Name `xml:"cpu-usage-process"`
		PID      string   `xml:"pid"`
		Name     string   `xml:"name"`
		FiveSec  string   `xml:"five-seconds"`
		TotalRun string   `xml:"total-run-time"`
	}

	type cpuUsageProcesses struct {
		XMLName   xml.Name      `xml:"cpu-usage-processes"`
		Processes []xmlProcess  `xml:"cpu-usage-process"`
	}

	type cpuUsage struct {
		XMLName   xml.Name           `xml:"cpu-utilization"`
		Processes cpuUsageProcesses  `xml:"cpu-usage-processes"`
	}

	var usage cpuUsage
	if err := xml.Unmarshal([]byte(data), &usage); err != nil {
		return nil, fmt.Errorf("parse NETCONF processes: %w", err)
	}

	var processes []ProcessData
	for i, xp := range usage.Processes.Processes {
		if i >= limit {
			break
		}
		pid, _ := strconv.Atoi(xp.PID)
		cpuPct, _ := strconv.ParseFloat(xp.FiveSec, 64)
		runtime, _ := strconv.ParseFloat(xp.TotalRun, 64)

		processes = append(processes, ProcessData{
			Name:    fmt.Sprintf("%s(%d)", xp.Name, pid),
			CPU:     cpuPct,
			Runtime: runtime,
		})
	}

	return processes, nil
}

func parseXMLFields(data string) map[string]interface{} {
	result := make(map[string]interface{})
	re := regexp.MustCompile(`<([^/>]+)>([^<]*)</\1>`)
	matches := re.FindAllStringSubmatch(data, -1)
	for _, m := range matches {
		if len(m) == 3 {
			key := strings.TrimSpace(m[1])
			val := strings.TrimSpace(m[2])
			if idx := strings.Index(key, " "); idx >= 0 {
				key = key[:idx]
			}
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				result[key] = f
			} else {
				result[key] = val
			}
		}
	}
	return result
}

func readKey(path string) (ssh.Signer, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(key)
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
