package json

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

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

func (p *Provider) parseJSON(raw string) map[string]interface{} {
	start := strings.Index(raw, "{")
	if start < 0 {
		return map[string]interface{}{"raw": raw, "parse_error": "no JSON found"}
	}
	depth := 0
	end := start
	for i := start; i < len(raw); i++ {
		if raw[i] == '{' {
			depth++
		} else if raw[i] == '}' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(raw[start:end]), &result); err != nil {
		return map[string]interface{}{"raw": raw, "parse_error": err.Error()}
	}
	return result
}

func (p *Provider) exec(ctx context.Context, cmd string) (map[string]interface{}, error) {
	output, err := p.client.Execute(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return p.parseJSON(output), nil
}

func (p *Provider) stringField(data map[string]interface{}, key string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (p *Provider) numField(data map[string]interface{}, key string) float64 {
	if v, ok := data[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case string:
			if f, err := strconv.ParseFloat(n, 64); err == nil {
				return f
			}
		}
	}
	return 0
}

func (p *Provider) CollectSystem(ctx context.Context) (SystemData, error) {
	data, err := p.exec(ctx, "show version | json")
	if err != nil {
		return SystemData{}, err
	}
	return SystemData{
		Hostname: p.stringField(data, "host_name"),
		Version:  p.stringField(data, "kickstart_ver_str"),
		Model:    p.stringField(data, "chassis_id"),
		Serial:   p.stringField(data, "proc_board_id"),
		Uptime:   p.parseUptime(data),
	}, nil
}

func (p *Provider) CollectInterfaces(ctx context.Context) ([]InterfaceData, error) {
	data, err := p.exec(ctx, "show interface | json")
	if err != nil {
		return nil, err
	}

	var interfaces []InterfaceData
	table, ok := data["TABLE_interface"]
	if !ok {
		return nil, nil
	}
	rows, ok := table.(map[string]interface{})["ROW_interface"].([]interface{})
	if !ok {
		return nil, nil
	}

	for _, row := range rows {
		m, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		iface := InterfaceData{
			Name:        p.stringField(m, "interface"),
			AdminUp:     p.stringField(m, "admin_state") == "up",
			OperUp:      p.stringField(m, "state") == "up",
			RxBytes:     p.numField(m, "eth_inbytes"),
			TxBytes:     p.numField(m, "eth_outbytes"),
			RxErrors:    p.numField(m, "eth_inerr"),
			TxErrors:    p.numField(m, "eth_outerr"),
			RxCrcErrors: p.numField(m, "eth_crc"),
			RxFrame:     p.numField(m, "eth_frame"),
			RxRunts:     p.numField(m, "eth_runts"),
			RxGiants:    p.numField(m, "eth_giants"),
			RxDrops:     p.numField(m, "eth_in_drops") + p.numField(m, "eth_underrun"),
			TxDrops:     p.numField(m, "eth_out_drops") + p.numField(m, "eth_overrun"),
			RxDiscards:  p.numField(m, "eth_in_discards"),
			TxDiscards:  p.numField(m, "eth_out_discards"),
			Resets:      p.numField(m, "eth_resets"),
			Speed:       p.numField(m, "eth_speed"),
			Mtu:         p.numField(m, "eth_mtu"),
		}
		if iface.Speed > 0 {
			iface.Speed = iface.Speed * 1e6
		}
		interfaces = append(interfaces, iface)
	}
	return interfaces, nil
}

func (p *Provider) CollectOptics(ctx context.Context) (map[string]OpticData, error) {
	result := make(map[string]OpticData)
	data, err := p.exec(ctx, "show interface transceiver | json")
	if err != nil {
		return result, nil
	}

	table, ok := data["TABLE_interface"]
	if !ok {
		return result, nil
	}
	rows, ok := table.(map[string]interface{})["ROW_interface"].([]interface{})
	if !ok {
		return result, nil
	}

	for _, row := range rows {
		m, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		name := p.stringField(m, "interface")
		if name == "" {
			continue
		}
		optic := OpticData{}
		if raw, ok := m["TABLE_sfp_info"]; ok {
			switch sfpData := raw.(type) {
			case map[string]interface{}:
				if sfpRows, ok := sfpData["ROW_sfp_info"].([]interface{}); ok {
					for _, sr := range sfpRows {
						s, ok := sr.(map[string]interface{})
						if !ok {
							continue
						}
						optic.RxPower = p.numField(s, "rx_pwr")
						optic.TxPower = p.numField(s, "tx_pwr")
						optic.Temp = p.numField(s, "temperature")
						optic.Voltage = p.numField(s, "voltage")
						optic.BiasCurrent = p.numField(s, "bias_current")
					}
				} else if sfpRow, ok := sfpData["ROW_sfp_info"].(map[string]interface{}); ok {
					optic.RxPower = p.numField(sfpRow, "rx_pwr")
					optic.TxPower = p.numField(sfpRow, "tx_pwr")
					optic.Temp = p.numField(sfpRow, "temperature")
					optic.Voltage = p.numField(sfpRow, "voltage")
					optic.BiasCurrent = p.numField(sfpRow, "bias_current")
				}
			}
		}
		if optic.RxPower != 0 || optic.TxPower != 0 || optic.Temp != 0 {
			result[name] = optic
		}
	}
	return result, nil
}

func (p *Provider) CollectResources(ctx context.Context) (ResourceData, error) {
	data, err := p.exec(ctx, "show system resources | json")
	if err != nil {
		return ResourceData{}, err
	}
	return ResourceData{
		CPUUsage: p.numField(data, "cpu_state_user") + p.numField(data, "cpu_state_kernel"),
		CPU5Sec:  p.numField(data, "load_avg_5min"),
		CPU1Min:  0,
		CPU5Min:  0,
		MemTotal: p.numField(data, "memory_usage_total"),
		MemUsed:  p.numField(data, "memory_usage_used"),
		MemFree:  p.numField(data, "memory_usage_free"),
	}, nil
}

func (p *Provider) CollectProcesses(ctx context.Context, limit int) ([]ProcessData, error) {
	data, err := p.exec(ctx, "show processes cpu | json")
	if err != nil {
		return nil, err
	}

	var processes []ProcessData
	table, ok := data["TABLE_process_cpu"]
	if !ok {
		return nil, nil
	}
	rows, ok := table.(map[string]interface{})["ROW_process_cpu"].([]interface{})
	if !ok {
		return nil, nil
	}

	for i, row := range rows {
		if i >= limit {
			break
		}
		m, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		processes = append(processes, ProcessData{
			Name:    p.stringField(m, "process"),
			CPU:     p.numField(m, "cpu_percent"),
			Runtime: p.numField(m, "runtime"),
		})
	}
	return processes, nil
}

func (p *Provider) CollectSTP(ctx context.Context) (STPData, error) {
	result := STPData{
		RootChanges:      make(map[string]float64),
		TopoChanges:      make(map[string]float64),
		PortStateChanges: make(map[string]map[string]float64),
		BlockedPorts:     make(map[string]float64),
		BPDUGuardEvents:  make(map[string]float64),
	}

	data, err := p.exec(ctx, "show spanning-tree summary | json")
	if err != nil {
		return result, nil
	}

	if table, ok := data["TABLE_tree"]; ok {
		switch t := table.(type) {
		case map[string]interface{}:
			if rows, ok := t["ROW_tree"].([]interface{}); ok {
				for _, row := range rows {
					p.parseSTPRow(row, result)
				}
			} else if row, ok := t["ROW_tree"].(map[string]interface{}); ok {
				p.parseSTPRow(row, result)
			}
		case []interface{}:
			for _, row := range t {
				p.parseSTPRow(row, result)
			}
		}
	}

	detailData, err := p.exec(ctx, "show spanning-tree detail | json")
	if err != nil {
		return result, nil
	}

	if table, ok := detailData["TABLE_vlan"]; ok {
		switch rows := table.(type) {
		case map[string]interface{}:
			if rowList, ok := rows["ROW_vlan"].([]interface{}); ok {
				for _, row := range rowList {
					p.parseSTPDetailRow(row, result)
				}
			} else if rowMap, ok := rows["ROW_vlan"].(map[string]interface{}); ok {
				p.parseSTPDetailRow(rowMap, result)
			}
		case []interface{}:
			for _, row := range rows {
				p.parseSTPDetailRow(row, result)
			}
		}
	}

	return result, nil
}

func (p *Provider) parseSTPRow(row interface{}, result STPData) {
	m, ok := row.(map[string]interface{})
	if !ok {
		return
	}
	vlan := p.stringField(m, "stp_tree_summary")
	if vlan == "" {
		return
	}
	result.BlockedPorts[vlan] = p.numField(m, "blocking")
}

func (p *Provider) parseSTPDetailRow(row interface{}, result STPData) {
	m, ok := row.(map[string]interface{})
	if !ok {
		return
	}
	vlan := p.stringField(m, "vlan_id")
	if vlan == "" {
		return
	}
	result.RootChanges[vlan] = p.numField(m, "root_changes")
	result.TopoChanges[vlan] = p.numField(m, "topology_changes")
}

func (p *Provider) CollectMAC(ctx context.Context) (MACData, error) {
	result := MACData{Moves: make(map[string]float64)}

	data, err := p.exec(ctx, "show mac address-table count | json")
	if err != nil {
		return result, nil
	}

	result.TableCount = p.numField(data, "mac_table_count")

	if total, ok := data["TABLE_vlan"]; ok {
		if rows, ok := total.(map[string]interface{})["ROW_vlan"].([]interface{}); ok {
			for _, row := range rows {
				m, ok := row.(map[string]interface{})
				if !ok {
					continue
				}
				vlan := p.stringField(m, "vlan_id")
				if vlan == "" {
					continue
				}
				result.Moves[vlan] = p.numField(m, "mac_move_count")
			}
		}
	}

	return result, nil
}

func (p *Provider) CollectVLANs(ctx context.Context) (VLANData, error) {
	result := VLANData{}

	data, err := p.exec(ctx, "show vlan brief | json")
	if err != nil {
		return result, nil
	}

	if table, ok := data["TABLE_vlanbriefxbrief"]; ok {
		if rows, ok := table.(map[string]interface{})["ROW_vlanbriefxbrief"].([]interface{}); ok {
			for _, row := range rows {
				m, ok := row.(map[string]interface{})
				if !ok {
					continue
				}
				result.Count++
				if p.stringField(m, "vlan_state") == "active" || p.stringField(m, "vlan_state") == "act/unsup" {
					result.Active++
				}
			}
		}
	}

	return result, nil
}

func (p *Provider) CollectPortChannels(ctx context.Context) (PortChannelData, error) {
	result := PortChannelData{
		Status:           make(map[string]bool),
		MemberStatus:     make(map[string]map[string]bool),
		SuspendedMembers: make(map[string]float64),
		Members:          make(map[string][]string),
	}

	data, err := p.exec(ctx, "show port-channel summary | json")
	if err != nil {
		return result, nil
	}

	table, ok := data["TABLE_channel"]
	if !ok {
		return result, nil
	}

	rows, ok := table.(map[string]interface{})["ROW_channel"].([]interface{})
	if !ok {
		return result, nil
	}

	for _, row := range rows {
		m, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		pc := p.stringField(m, "port-channel")
		if pc == "" {
			pc = p.stringField(m, "port_channel")
		}
		if pc == "" {
			continue
		}
		status := p.stringField(m, "status")
		result.Status[pc] = status == "U" || status == "P" || status == "D" || status == "up"

		if members, ok := m["TABLE_member"]; ok {
			if memberRows, ok := members.(map[string]interface{})["ROW_member"].([]interface{}); ok {
				suspended := 0.0
				memberMap := make(map[string]bool)
				var memberNames []string
				for _, mr := range memberRows {
					mm, ok := mr.(map[string]interface{})
					if !ok {
						continue
					}
					iface := p.stringField(mm, "port")
					if iface == "" {
						continue
					}
					memberNames = append(memberNames, iface)
					portStatus := p.stringField(mm, "port-status")
					isActive := portStatus == "P" || portStatus == "U" || portStatus == "D"
					memberMap[iface] = isActive
					if !isActive {
						suspended++
					}
				}
				result.MemberStatus[pc] = memberMap
				result.SuspendedMembers[pc] = suspended
				result.Members[pc] = memberNames
			}
		}
	}

	return result, nil
}

func (p *Provider) CollectL3Tables(ctx context.Context) (L3TableData, error) {
	result := L3TableData{}

	arpData, err := p.exec(ctx, "show ip arp summary | json")
	if err == nil {
		result.ARPCount = p.numField(arpData, "total_entries")
		result.ARPUtilization = p.numField(arpData, "total_entries")
	}

	routeData, err := p.exec(ctx, "show ip route summary | json")
	if err == nil {
		result.RoutingUtilization = p.numField(routeData, "total_routes")
	}

	cefData, err := p.exec(ctx, "show ip cef summary | json")
	if err == nil {
		result.FIBUtilization = p.numField(cefData, "total_routes")
	}

	adjData, err := p.exec(ctx, "show adjacency summary | json")
	if err == nil {
		result.AdjacencyUtilization = p.numField(adjData, "total_entries")
	}

	return result, nil
}

func (p *Provider) CollectHardware(ctx context.Context) (HardwareData, error) {
	result := HardwareData{
		Temperature:       make(map[string]float64),
		FanStatus:         make(map[string]bool),
		PowerSupply:       make(map[string]bool),
		ModuleStatus:      make(map[string]bool),
		StackMemberStatus: make(map[string]bool),
	}

	envData, err := p.exec(ctx, "show environment | json")
	if err != nil {
		return result, nil
	}

	if tempTable, ok := envData["TABLE_tempinfo"]; ok {
		if rows, ok := tempTable.(map[string]interface{})["ROW_tempinfo"].([]interface{}); ok {
			for _, row := range rows {
				m, ok := row.(map[string]interface{})
				if !ok {
					continue
				}
				sensor := p.stringField(m, "sensor")
				if sensor == "" {
					sensor = p.stringField(m, "name")
				}
				if sensor != "" {
					result.Temperature[sensor] = p.numField(m, "temperature")
				}
			}
		}
	}

	if fanTable, ok := envData["TABLE_faninfo"]; ok {
		if rows, ok := fanTable.(map[string]interface{})["ROW_faninfo"].([]interface{}); ok {
			for _, row := range rows {
				m, ok := row.(map[string]interface{})
				if !ok {
					continue
				}
				fan := p.stringField(m, "fanname")
				if fan == "" {
					fan = p.stringField(m, "name")
				}
				if fan != "" {
					result.FanStatus[fan] = p.stringField(m, "status") == "ok" || p.stringField(m, "status") == "normal"
				}
			}
		}
	}

	if psTable, ok := envData["TABLE_mod_pow_info"]; ok {
		if rows, ok := psTable.(map[string]interface{})["ROW_mod_pow_info"].([]interface{}); ok {
			for _, row := range rows {
				m, ok := row.(map[string]interface{})
				if !ok {
					continue
				}
				ps := p.stringField(m, "psname")
				if ps == "" {
					ps = p.stringField(m, "name")
				}
				if ps != "" {
					result.PowerSupply[ps] = p.stringField(m, "ps_status") == "ok" || p.stringField(m, "status") == "ok"
				}
			}
		}
	}

	if psTable, ok := envData["TABLE_psinfo"]; ok {
		if rows, ok := psTable.(map[string]interface{})["ROW_psinfo"].([]interface{}); ok {
			for _, row := range rows {
				m, ok := row.(map[string]interface{})
				if !ok {
					continue
				}
				ps := p.stringField(m, "psnum")
				if ps != "" {
					result.PowerSupply["PS-"+ps] = p.stringField(m, "ps_status") == "ok"
				}
			}
		}
	}

	if modTable, ok := envData["TABLE_modinfo"]; ok {
		if rows, ok := modTable.(map[string]interface{})["ROW_modinfo"].([]interface{}); ok {
			for _, row := range rows {
				m, ok := row.(map[string]interface{})
				if !ok {
					continue
				}
				mod := p.stringField(m, "modinf")
				if mod == "" {
					mod = p.stringField(m, "module_number")
				}
				if mod != "" {
					result.ModuleStatus[mod] = p.stringField(m, "status") == "ok" || p.stringField(m, "status") == "active"
				}
			}
		}
	}

	return result, nil
}

func (p *Provider) CollectSystemInfo(ctx context.Context) (SystemData2, error) {
	result := SystemData2{}

	ntpData, err := p.exec(ctx, "show ntp status | json")
	if err == nil {
		result.NTPSynced = p.stringField(ntpData, "sync_state") == "synchronized" || p.stringField(ntpData, "status") == "synchronized"
	}

	verData, err := p.exec(ctx, "show version | json")
	if err == nil {
		result.Uptime = p.parseUptime(verData)
	}

	fsData, err := p.exec(ctx, "show file systems | json")
	if err == nil {
		if table, ok := fsData["TABLE_filesys"]; ok {
			if rows, ok := table.(map[string]interface{})["ROW_filesys"].([]interface{}); ok {
				for _, row := range rows {
					m, ok := row.(map[string]interface{})
					if !ok {
						continue
					}
					fs := p.stringField(m, "filesystem")
					if fs != "" {
						total := p.numField(m, "total_kbytes")
						used := p.numField(m, "used_kbytes")
						if total > 0 {
							result.FlashUtil = (used / total) * 100
						}
						break
					}
				}
			}
		}
	}

	return result, nil
}

func (p *Provider) CollectCapacity(ctx context.Context) (CapacityData, error) {
	result := CapacityData{
		BufferUtilization: make(map[string]float64),
		QueueUtilization:  make(map[string]float64),
	}

	ifaceData, err := p.exec(ctx, "show interface | json")
	if err == nil {
		totalPorts := 0
		activePorts := 0
		if table, ok := ifaceData["TABLE_interface"]; ok {
			if rows, ok := table.(map[string]interface{})["ROW_interface"].([]interface{}); ok {
				for _, row := range rows {
					m, ok := row.(map[string]interface{})
					if !ok {
						continue
					}
					name := p.stringField(m, "interface")
					if strings.HasPrefix(name, "Vlan") || strings.HasPrefix(name, "Loopback") || strings.HasPrefix(name, "Null") {
						continue
					}
					totalPorts++
					if p.stringField(m, "state") == "up" {
						activePorts++
					}
				}
			}
		}
		if totalPorts > 0 {
			result.PortUtilizationRatio = float64(activePorts) / float64(totalPorts) * 100
		}
	}

	bufData, err := p.exec(ctx, "show buffers | json")
	if err == nil {
		if table, ok := bufData["TABLE_buffer_pool"]; ok {
			if rows, ok := table.(map[string]interface{})["ROW_buffer_pool"].([]interface{}); ok {
				for _, row := range rows {
					m, ok := row.(map[string]interface{})
					if !ok {
						continue
					}
					pool := p.stringField(m, "pool_name")
					if pool == "" {
						pool = p.stringField(m, "name")
					}
					if pool != "" {
						total := p.numField(m, "total")
						used := p.numField(m, "used")
						if total > 0 {
							result.BufferUtilization[pool] = (used / total) * 100
						}
					}
				}
			}
		}
	}

	return result, nil
}

func (p *Provider) parseUptime(data map[string]interface{}) float64 {
	days := p.numField(data, "kern_uptm_days")
	hours := p.numField(data, "kern_uptm_hrs")
	mins := p.numField(data, "kern_uptm_mins")
	secs := p.numField(data, "kern_uptm_secs")
	return days*86400 + hours*3600 + mins*60 + secs
}

type SystemData struct {
	Hostname string
	Version  string
	Model    string
	Serial   string
	Uptime   float64
}

type InterfaceData struct {
	Name        string
	AdminUp     bool
	OperUp      bool
	RxBytes     float64
	TxBytes     float64
	RxErrors    float64
	TxErrors    float64
	RxCrcErrors float64
	RxFrame     float64
	RxRunts     float64
	RxGiants    float64
	RxDrops     float64
	TxDrops     float64
	RxDiscards  float64
	TxDiscards  float64
	Resets      float64
	Flaps       float64
	Speed       float64
	Mtu         float64
}

type OpticData struct {
	RxPower     float64
	TxPower     float64
	Temp        float64
	Voltage     float64
	BiasCurrent float64
}

type ResourceData struct {
	CPUUsage float64
	CPU5Sec  float64
	CPU1Min  float64
	CPU5Min  float64
	MemTotal float64
	MemUsed  float64
	MemFree  float64
}

type ProcessData struct {
	Name    string
	CPU     float64
	Runtime float64
}

type STPData struct {
	RootChanges      map[string]float64
	TopoChanges      map[string]float64
	PortStateChanges map[string]map[string]float64
	BlockedPorts     map[string]float64
	BPDUGuardEvents  map[string]float64
}

type MACData struct {
	TableUtilization float64
	TableCount       float64
	Moves            map[string]float64
}

type VLANData struct {
	Count  int
	Active int
}

type PortChannelData struct {
	Status           map[string]bool
	MemberStatus     map[string]map[string]bool
	SuspendedMembers map[string]float64
	Members          map[string][]string
}

type L3TableData struct {
	ARPUtilization       float64
	ARPCount             float64
	RoutingUtilization   float64
	FIBUtilization       float64
	AdjacencyUtilization float64
}

type HardwareData struct {
	Temperature       map[string]float64
	FanStatus         map[string]bool
	PowerSupply       map[string]bool
	PowerRedundant    bool
	ModuleStatus      map[string]bool
	StackMemberStatus map[string]bool
}

type SystemData2 struct {
	NTPSynced    bool
	Uptime       float64
	ReloadEvents float64
	FlashUtil    float64
}

type CapacityData struct {
	PortUtilizationRatio float64
	TransceiverCount     int
	BufferUtilization    map[string]float64
	QueueUtilization     map[string]float64
}
