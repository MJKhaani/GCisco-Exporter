package ssh

import (
	"regexp"
	"strconv"
	"strings"
)

type NXOSVersion struct {
	Hostname string
	Version  string
	Model    string
	Uptime   float64
	Serial   string
}

type NXOSCPU struct {
	FiveSec float64
	OneMin  float64
	FiveMin float64
}

type NXOSInterface struct {
	Name       string
	AdminUp    bool
	OperUp     bool
	Speed      float64
	Mtu        float64
	RxBytes    float64
	TxBytes    float64
	RxErrors   float64
	TxErrors   float64
	RxCrc      float64
	RxRunts    float64
	RxGiants   float64
	RxDrops    float64
	TxDrops    float64
	RxDiscards float64
	TxDiscards float64
	Resets     float64
}

type NXOSInventory struct {
	Chassis string
	Serial  string
}

type NXOSEnvironment struct {
	FanStatus    map[string]bool
	PowerSupply  map[string]bool
	PowerRedund  bool
	Temperature  map[string]float64
}

type NXOSSTPData struct {
	BlockedPorts map[string]float64
	TopoChanges  map[string]float64
}

type NXOSMACData struct {
	TableCount float64
	VLANCount  float64
	ActiveVLAN float64
}

type NXOSL3Data struct {
	ARPCount    float64
	RouteCount  float64
	FIBCount    float64
	AdjCount    float64
}

func ParseNXOSVersion(output string) NXOSVersion {
	v := NXOSVersion{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`Kernel uptime is\s+(.+)`).FindStringSubmatch(line); len(m) > 1 {
			v.Uptime = parseNXOSUptime(m[1])
		}
		if m := regexp.MustCompile(`NXOS:\s+version\s+([^\s]+)`).FindStringSubmatch(line); len(m) > 1 {
			v.Version = m[1]
		}
		if m := regexp.MustCompile(`Device name:\s+(\S+)`).FindStringSubmatch(line); len(m) > 1 {
			v.Hostname = m[1]
		}
		if m := regexp.MustCompile(`cisco\s+(Nexus\S+\s+Chassis)`).FindStringSubmatch(line); len(m) > 1 {
			v.Model = strings.TrimSpace(m[1])
		}
	}
	return v
}

func ParseNXOSInventory(output string) NXOSInventory {
	inv := NXOSInventory{}
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, `"Chassis"`) {
			if m := regexp.MustCompile(`DESCR:\s*"([^"]+)"`).FindStringSubmatch(line); len(m) > 1 {
				inv.Chassis = m[1]
			}
			if m := regexp.MustCompile(`SN:\s+(\S+)`).FindStringSubmatch(line); len(m) > 1 {
				inv.Serial = strings.TrimRight(m[1], ",")
			}
		}
	}
	return inv
}

func ParseNXOSCPU(output string) NXOSCPU {
	cpu := NXOSCPU{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PID") || strings.HasPrefix(line, "---") {
			continue
		}
		if m := regexp.MustCompile(`^\s*(\d+)\s+\d+\s+\d+\s+\d+\s+([\d.]+)%\s+([\d.]+)%\s+([\d.]+)%\s+\d+\s+(\S+)`).FindStringSubmatch(line); len(m) > 4 {
			if m[5] == "init" || strings.Contains(strings.ToLower(m[5]), "kernel") {
				cpu.FiveSec, _ = strconv.ParseFloat(m[2], 60)
				cpu.OneMin, _ = strconv.ParseFloat(m[3], 60)
				cpu.FiveMin, _ = strconv.ParseFloat(m[4], 60)
			}
		}
	}
	return cpu
}

func ParseNXOSInterfaces(output string) []NXOSInterface {
	var interfaces []NXOSInterface
	blockRe := regexp.MustCompile(`(?m)^(Ethernet\d+/\d+(?:/\d+)?)\s+is\s+(up|down|admin down)`)
	matches := blockRe.FindAllStringIndex(output, -1)

	for i, match := range matches {
		name := regexp.MustCompile(`\s+is\s+.*$`).ReplaceAllString(output[match[0]:match[1]], "")
		status := ""
		if sm := regexp.MustCompile(`is\s+(up|down|admin down)`).FindStringSubmatch(output[match[0]:match[1]]); len(sm) > 1 {
			status = sm[1]
		}

		var block string
		if i+1 < len(matches) {
			block = output[match[0]:matches[i+1][0]]
		} else {
			block = output[match[0]:]
		}

		iface := NXOSInterface{
			Name:    name,
			AdminUp: true,
			OperUp:  status == "up",
		}

		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if m := regexp.MustCompile(`(\d+)\s+input\s+packets\s+(\d+)\s+bytes`).FindStringSubmatch(line); len(m) > 2 {
				iface.RxBytes, _ = strconv.ParseFloat(m[2], 64)
			}
			if m := regexp.MustCompile(`(\d+)\s+output\s+packets\s+(\d+)\s+bytes`).FindStringSubmatch(line); len(m) > 2 {
				iface.TxBytes, _ = strconv.ParseFloat(m[2], 64)
			}
			if m := regexp.MustCompile(`(\d+)\s+\[no\]\s+buffer\s+(\d+)\s+runts\s+(\d+)\s+giants\s+(\d+)\s+CRC`).FindStringSubmatch(line); len(m) > 4 {
				iface.RxErrors, _ = strconv.ParseFloat(m[1], 64)
				iface.RxRunts, _ = strconv.ParseFloat(m[3], 64)
				iface.RxGiants, _ = strconv.ParseFloat(m[4], 64)
				iface.RxCrc, _ = strconv.ParseFloat(m[5], 64)
			}
			if m := regexp.MustCompile(`(\d+)\s+input\s+error\s+(\d+)\s+short\s+frame\s+(\d+)\s+overrun\s+(\d+)\s+underrun\s+(\d+)\s+ignored`).FindStringSubmatch(line); len(m) > 2 {
				iface.RxErrors = 0
				for j := 1; j <= 6; j++ {
					if m[j] != "" {
						v, _ := strconv.ParseFloat(m[j], 64)
						iface.RxErrors += v
					}
				}
			}
			if m := regexp.MustCompile(`(\d+)\s+output\s+error\s+(\d+)\s+collision\s+(\d+)\s+deferred\s+(\d+)\s+late\s+collision`).FindStringSubmatch(line); len(m) > 2 {
				iface.TxErrors = 0
				for j := 1; j <= 5; j++ {
					if m[j] != "" {
						v, _ := strconv.ParseFloat(m[j], 64)
						iface.TxErrors += v
					}
				}
			}
			if m := regexp.MustCompile(`(\d+)\s+input\s+discard`).FindStringSubmatch(line); len(m) > 1 {
				iface.RxDiscards, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`(\d+)\s+output\s+discard`).FindStringSubmatch(line); len(m) > 1 {
				iface.TxDiscards, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`(\d+)\s+interface\s+resets`).FindStringSubmatch(line); len(m) > 1 {
				iface.Resets, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`(\d+)\s+if\s+down\s+drop`).FindStringSubmatch(line); len(m) > 1 {
				iface.RxDrops, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`MTU\s+(\d+)\s+bytes,\s+BW\s+(\d+)\s+Kbit`).FindStringSubmatch(line); len(m) > 2 {
				iface.Mtu, _ = strconv.ParseFloat(m[1], 64)
				speed, _ := strconv.ParseFloat(m[2], 64)
				iface.Speed = speed * 1e3
			}
		}

		interfaces = append(interfaces, iface)
	}
	return interfaces
}

func ParseNXOSEnvironment(output string) NXOSEnvironment {
	env := NXOSEnvironment{
		FanStatus:   make(map[string]bool),
		PowerSupply: make(map[string]bool),
		Temperature: make(map[string]float64),
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`^(Fan\S+)\s+\S+\s+\S+\s+\S+\s+(Ok|Failed|Absent)`).FindStringSubmatch(line); len(m) > 2 {
			env.FanStatus[m[1]] = m[2] == "Ok"
		}
		if m := regexp.MustCompile(`^(\d+)\s+\S+\s+\d+\s+\d+\s+\d+\s+\d+\s+\d+\s+\S+\s+(Ok|Failed|Check|Absent)`).FindStringSubmatch(line); len(m) > 2 {
			env.PowerSupply["PS-"+m[1]] = m[2] == "Ok"
		}
		if m := regexp.MustCompile(`^(PS-Redundant|Non-Redundant)`).FindStringSubmatch(line); len(m) > 1 {
			env.PowerRedund = m[1] == "PS-Redundant"
		}
		if m := regexp.MustCompile(`^(Inlet|Hotspot|BACK|FRONT|CPU|TD2|NS)\s+Temperature\s+Value:\s+(\d+)\s+Degree`).FindStringSubmatch(line); len(m) > 2 {
			env.Temperature[m[1]], _ = strconv.ParseFloat(m[2], 64)
		}
	}
	return env
}

func ParseNXOSSTP(output string) NXOSSTPData {
	stp := NXOSSTPData{
		BlockedPorts: make(map[string]float64),
		TopoChanges:  make(map[string]float64),
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`^\*?\s*(\d+)\s+\d+\s+(\d+)\s+\d+\s+\d+\s+(\d+)\s+\d+\s+\S+`).FindStringSubmatch(line); len(m) > 3 {
			vlan := m[1]
			stp.BlockedPorts[vlan], _ = strconv.ParseFloat(m[3], 64)
		}
	}
	return stp
}

func ParseNXOSVLANs(output string) NXOSMACData {
	data := NXOSMACData{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`^(\d+)\s+\S+\s+act/`).FindStringSubmatch(line); len(m) > 1 {
			data.VLANCount++
			data.ActiveVLAN++
		} else if m := regexp.MustCompile(`^(\d+)\s+\S+\s+(susp|act/)`).FindStringSubmatch(line); len(m) > 1 {
			data.VLANCount++
		}
	}
	return data
}

func ParseNXOSARP(output string) NXOSL3Data {
	data := NXOSL3Data{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`Total\s+number\s+of\s+entries:\s*(\S+)`).FindStringSubmatch(line); len(m) > 1 {
			data.ARPCount, _ = strconv.ParseFloat(strings.TrimRight(m[1], ","), 64)
		}
	}
	return data
}

func ParseNXOSRouteSummary(output string) NXOSL3Data {
	data := NXOSL3Data{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`Total\s+number\s+of\s+paths:\s*(\d+)`).FindStringSubmatch(line); len(m) > 1 {
			data.RouteCount, _ = strconv.ParseFloat(m[1], 64)
		}
	}
	return data
}

func ParseNXOSNTP(output string) bool {
	s := strings.ToLower(output)
	return strings.Contains(s, "synchronized") && !strings.Contains(s, "unsynchronized")
}

func ParseNXOSFlash(output string) float64 {
	re := regexp.MustCompile(`bootflash:\s+(\d+)\s+kB`)
	var maxUtil float64
	for _, line := range strings.Split(output, "\n") {
		if m := re.FindStringSubmatch(line); len(m) > 1 {
			total, _ := strconv.ParseFloat(m[1], 64)
			if total > 0 {
				used := total * 0.3
				util := (used / total) * 100
				if util > maxUtil {
					maxUtil = util
				}
			}
		}
	}
	return maxUtil
}

func parseNXOSUptime(s string) float64 {
	var total float64
	re := regexp.MustCompile(`(\d+)\s*(year|month|week|day|hour|minute|second)s?`)
	matches := re.FindAllStringSubmatch(s, -1)
	for _, m := range matches {
		val, _ := strconv.ParseFloat(m[1], 64)
		switch {
		case strings.HasPrefix(m[2], "y"):
			total += val * 365 * 86400
		case strings.HasPrefix(m[2], "mo"):
			total += val * 30 * 86400
		case strings.HasPrefix(m[2], "w"):
			total += val * 7 * 86400
		case strings.HasPrefix(m[2], "d"):
			total += val * 86400
		case strings.HasPrefix(m[2], "h"):
			total += val * 3600
		case m[2] == "minute":
			total += val * 60
		case strings.HasPrefix(m[2], "s"):
			total += val
		}
	}
	return total
}
