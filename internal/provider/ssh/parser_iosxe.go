package ssh

import (
	"regexp"
	"strconv"
	"strings"
)

type IOSXEVersion struct {
	Hostname string
	Version  string
	Model    string
	Uptime   float64
	Serial   string
}

type IOSXECPU struct {
	FiveSec float64
	OneMin  float64
	FiveMin float64
}

type IOSXEInterface struct {
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

type IOSXEInventory struct {
	Model  string
	Serial string
}

type IOSXEEnvironment struct {
	FanStatus   map[string]bool
	PowerSupply map[string]bool
	Temperature map[string]float64
}

type IOSXESTPData struct {
	BlockedPorts map[string]float64
	TopoChanges  map[string]float64
}

type IOSXEMACData struct {
	TableCount float64
	VLANCount  int
	ActiveVLAN int
}

type IOSXEL3Data struct {
	ARPCount   float64
	RouteCount float64
	FIBCount   float64
}

func ParseIOSXEVersion(output string) IOSXEVersion {
	v := IOSXEVersion{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`uptime is\s+(.+)`).FindStringSubmatch(line); len(m) > 1 {
			v.Uptime = parseIOSXEUptime(m[1])
		}
		if m := regexp.MustCompile(`[Vv]ersion\s+([^\s,]+)`).FindStringSubmatch(line); len(m) > 1 {
			v.Version = m[1]
		}
		if m := regexp.MustCompile(`cisco\s+(\S+)\s+.*processor`).FindStringSubmatch(line); len(m) > 1 {
			v.Model = m[1]
		}
		if m := regexp.MustCompile(`^(\S+)\s+uptime`).FindStringSubmatch(line); len(m) > 1 {
			v.Hostname = m[1]
		}
	}
	return v
}

func ParseIOSXEInventory(output string) IOSXEInventory {
	inv := IOSXEInventory{}
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, `"Chassis"`) || strings.Contains(line, `"Switch 1"`) {
			if m := regexp.MustCompile(`DESCR:\s*"([^"]+)"`).FindStringSubmatch(line); len(m) > 1 {
				inv.Model = m[1]
			}
			if m := regexp.MustCompile(`SN:\s+(\S+)`).FindStringSubmatch(line); len(m) > 1 {
				inv.Serial = strings.TrimRight(m[1], ",")
			}
		}
	}
	return inv
}

func ParseIOSXECPU(output string) IOSXECPU {
	cpu := IOSXECPU{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`CPU\s+utilization.*?(\d+)%/(\d+)%;\s+one\s+minute:\s+(\d+)%;\s+five\s+minutes:\s+(\d+)%`).FindStringSubmatch(line); len(m) > 4 {
			cpu.FiveSec, _ = strconv.ParseFloat(m[1], 64)
			cpu.OneMin, _ = strconv.ParseFloat(m[3], 64)
			cpu.FiveMin, _ = strconv.ParseFloat(m[4], 64)
		}
	}
	return cpu
}

func ParseIOSXEInterfaces(output string) []IOSXEInterface {
	var interfaces []IOSXEInterface
	blockRe := regexp.MustCompile(`(?m)^(\S+)\s+is\s+(up|down|administratively down)`)
	matches := blockRe.FindAllStringIndex(output, -1)

	for i, match := range matches {
		name := regexp.MustCompile(`\s+is\s+.*$`).ReplaceAllString(output[match[0]:match[1]], "")
		status := ""
		if sm := regexp.MustCompile(`is\s+(up|down|administratively down)`).FindStringSubmatch(output[match[0]:match[1]]); len(sm) > 1 {
			status = sm[1]
		}

		var block string
		if i+1 < len(matches) {
			block = output[match[0]:matches[i+1][0]]
		} else {
			block = output[match[0]:]
		}

		iface := IOSXEInterface{
			Name:    name,
			AdminUp: status != "administratively down",
			OperUp:  status == "up",
		}

		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if m := regexp.MustCompile(`(\d+)\s+packets\s+input[,\s]+(\d+)\s+bytes`).FindStringSubmatch(line); len(m) > 2 {
				iface.RxBytes, _ = strconv.ParseFloat(m[2], 64)
			}
			if m := regexp.MustCompile(`(\d+)\s+packets\s+output[,\s]+(\d+)\s+bytes`).FindStringSubmatch(line); len(m) > 2 {
				iface.TxBytes, _ = strconv.ParseFloat(m[2], 64)
			}
			if m := regexp.MustCompile(`(\d+)\s+input\s+errors[,\s]+(\d+)\s+CRC`).FindStringSubmatch(line); len(m) > 2 {
				iface.RxErrors, _ = strconv.ParseFloat(m[1], 64)
				iface.RxCrc, _ = strconv.ParseFloat(m[2], 64)
			}
			if m := regexp.MustCompile(`(\d+)\s+output\s+errors[,\s]+(\d+)\s+collisions`).FindStringSubmatch(line); len(m) > 2 {
				iface.TxErrors, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`\b(\d+)\s+runts\b`).FindStringSubmatch(line); len(m) > 1 {
				iface.RxRunts, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`\b(\d+)\s+giants\b`).FindStringSubmatch(line); len(m) > 1 {
				iface.RxGiants, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`Total\s+output\s+drops:\s*(\d+)`).FindStringSubmatch(line); len(m) > 1 {
				iface.TxDrops, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`\b(\d+)\s+input\s+drops\b`).FindStringSubmatch(line); len(m) > 1 {
				iface.RxDrops, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`\b(\d+)\s+input\s+discards\b`).FindStringSubmatch(line); len(m) > 1 {
				iface.RxDiscards, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`\b(\d+)\s+output\s+discards\b`).FindStringSubmatch(line); len(m) > 1 {
				iface.TxDiscards, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`\b(\d+)\s+interface\s+resets\b`).FindStringSubmatch(line); len(m) > 1 {
				iface.Resets, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`[Mm]TU\s+(\d+)`).FindStringSubmatch(line); len(m) > 1 {
				iface.Mtu, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`BW\s+(\d+)\s+([KMGT]?)bit`).FindStringSubmatch(line); len(m) > 2 {
				speed, _ := strconv.ParseFloat(m[1], 64)
				switch m[2] {
				case "K":
					speed *= 1e3
				case "M":
					speed *= 1e6
				case "G":
					speed *= 1e9
				case "T":
					speed *= 1e12
				}
				iface.Speed = speed
			}
		}

		interfaces = append(interfaces, iface)
	}
	return interfaces
}

func ParseIOSXEEnvironment(output string) IOSXEEnvironment {
	env := IOSXEEnvironment{
		FanStatus:   make(map[string]bool),
		PowerSupply: make(map[string]bool),
		Temperature: make(map[string]float64),
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`^(Switch\s+\d+\s+Fan\s+\d+)\s+is\s+(OK|FAILED)`).FindStringSubmatch(line); len(m) > 2 {
			env.FanStatus[m[1]] = m[2] == "OK"
		}
		if m := regexp.MustCompile(`^(FAN\s+PS-\d+)\s+is\s+(OK|FAILED)`).FindStringSubmatch(line); len(m) > 2 {
			env.FanStatus[m[1]] = m[2] == "OK"
		}
		if m := regexp.MustCompile(`^(Inlet|Hotspot)\s+Temperature\s+Value:\s+(\d+)\s+Degree`).FindStringSubmatch(line); len(m) > 2 {
			env.Temperature[m[1]], _ = strconv.ParseFloat(m[2], 64)
		}
		if m := regexp.MustCompile(`^(\S+)\s+\S+\s+\S+\s+(OK|FAILED)\s+Good`).FindStringSubmatch(line); len(m) > 2 {
			env.PowerSupply[m[1]] = m[2] == "OK"
		}
	}
	return env
}

func ParseIOSXESTP(output string) IOSXESTPData {
	stp := IOSXESTPData{
		BlockedPorts: make(map[string]float64),
		TopoChanges:  make(map[string]float64),
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`VLAN0*(\d+)\s+.*\s+(\d+)\s+.*\s+(\d+)\s+.*\s+(\d+)`).FindStringSubmatch(line); len(m) > 4 {
			vlan := m[1]
			stp.BlockedPorts[vlan], _ = strconv.ParseFloat(m[4], 64)
		}
	}
	return stp
}

func ParseIOSXEVLANs(output string) IOSXEMACData {
	data := IOSXEMACData{}
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

func ParseIOSXEMACCount(output string) float64 {
	if m := regexp.MustCompile(`Total\s+Mac\s+Addresses:\s*(\d+)`).FindStringSubmatch(output); len(m) > 1 {
		v, _ := strconv.ParseFloat(m[1], 64)
		return v
	}
	return 0
}

func ParseIOSXEARP(output string) IOSXEL3Data {
	data := IOSXEL3Data{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`Total\s+(?:number\s+of\s+)?entries\s*:\s*(\d+)`).FindStringSubmatch(line); len(m) > 1 {
			data.ARPCount, _ = strconv.ParseFloat(m[1], 64)
		}
	}
	return data
}

func ParseIOSXERouteSummary(output string) IOSXEL3Data {
	data := IOSXEL3Data{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`Total\s+(?:number\s+of\s+)?(?:routes|entries)\s*:\s*(\d+)`).FindStringSubmatch(line); len(m) > 1 {
			data.RouteCount, _ = strconv.ParseFloat(m[1], 64)
		}
	}
	return data
}

func ParseIOSXENTP(output string) bool {
	s := strings.ToLower(output)
	return strings.Contains(s, "synchronized") && !strings.Contains(s, "unsynchronized")
}

func ParseIOSXEFlash(output string) float64 {
	re := regexp.MustCompile(`(\d+)\s+(\d+)\s+`)
	matches := re.FindAllStringSubmatch(output, -1)
	var maxUtil float64
	for _, m := range matches {
		total, _ := strconv.ParseFloat(m[1], 64)
		used, _ := strconv.ParseFloat(m[2], 64)
		if total > 0 {
			util := (used / total) * 100
			if util > maxUtil {
				maxUtil = util
			}
		}
	}
	return maxUtil
}

func parseIOSXEUptime(s string) float64 {
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
