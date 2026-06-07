package ssh

import (
	"regexp"
	"strconv"
	"strings"
)

type IOSVersion struct {
	Hostname string
	Version  string
	Model    string
	Uptime   float64
	Serial   string
}

type IOSCPU struct {
	FiveSec float64
	OneMin  float64
	FiveMin float64
}

type IOSInterface struct {
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

func ParseVersion(output string) IOSVersion {
	v := IOSVersion{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`uptime is\s+(.+)`).FindStringSubmatch(line); len(m) > 1 {
			v.Uptime = parseUptimeString(m[1])
		}
		if m := regexp.MustCompile(`[Vv]ersion\s+([^\s,]+)`).FindStringSubmatch(line); len(m) > 1 {
			v.Version = m[1]
		}
		if m := regexp.MustCompile(`cisco\s+(\S+)\s+.*processor`).FindStringSubmatch(line); len(m) > 1 {
			v.Model = m[1]
		}
		if m := regexp.MustCompile(`[Ss]erial\s+[Nn]umber\s*:\s*(\S+)`).FindStringSubmatch(line); len(m) > 1 {
			v.Serial = m[1]
		}
		if m := regexp.MustCompile(`^(\S+)\s+uptime`).FindStringSubmatch(line); len(m) > 1 {
			v.Hostname = m[1]
		}
	}
	return v
}

func ParseCPU(output string) IOSCPU {
	cpu := IOSCPU{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`five\s+seconds:\s*(\d+)%`).FindStringSubmatch(line); len(m) > 1 {
			cpu.FiveSec, _ = strconv.ParseFloat(m[1], 64)
		}
		if m := regexp.MustCompile(`one\s+minute:\s*(\d+)%`).FindStringSubmatch(line); len(m) > 1 {
			cpu.OneMin, _ = strconv.ParseFloat(m[1], 64)
		}
		if m := regexp.MustCompile(`five\s+minutes:\s*(\d+)%`).FindStringSubmatch(line); len(m) > 1 {
			cpu.FiveMin, _ = strconv.ParseFloat(m[1], 64)
		}
		if cpu.FiveSec == 0 && cpu.OneMin == 0 && cpu.FiveMin == 0 {
			if m := regexp.MustCompile(`CPU\s+utilization.*?(\d+)%.*?(\d+)%.*?(\d+)%`).FindStringSubmatch(line); len(m) > 3 {
				cpu.FiveSec, _ = strconv.ParseFloat(m[1], 64)
				cpu.OneMin, _ = strconv.ParseFloat(m[2], 64)
				cpu.FiveMin, _ = strconv.ParseFloat(m[3], 64)
			}
		}
	}
	return cpu
}

func ParseInterfaces(output string) []IOSInterface {
	var interfaces []IOSInterface

	blockRe := regexp.MustCompile(`(?m)^(\S+)\s+is\s+(up|down|administratively down)`)
	matches := blockRe.FindAllStringIndex(output, -1)

	for i, match := range matches {
		name := output[match[0]:match[1]]
		name = regexp.MustCompile(`\s+is\s+.*$`).ReplaceAllString(name, "")
		statusStr := output[match[0]:match[1]]
		statusMatch := regexp.MustCompile(`is\s+(up|down|administratively down)`).FindStringSubmatch(statusStr)
		status := ""
		if len(statusMatch) > 1 {
			status = statusMatch[1]
		}

		var block string
		if i+1 < len(matches) {
			block = output[match[0]:matches[i+1][0]]
		} else {
			block = output[match[0]:]
		}

		iface := IOSInterface{
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
			if m := regexp.MustCompile(`\b(\d+)\s+input\s+drops\b`).FindStringSubmatch(line); len(m) > 1 {
				iface.RxDrops, _ = strconv.ParseFloat(m[1], 64)
			}
			if m := regexp.MustCompile(`\b(\d+)\s+output\s+drops\b`).FindStringSubmatch(line); len(m) > 1 {
				iface.TxDrops, _ = strconv.ParseFloat(m[1], 64)
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

func parseUptimeString(s string) float64 {
	var total float64
	re := regexp.MustCompile(`(\d+)\s*(year|week|day|hour|minute|second|yr|wk|hr|min|sec)s?`)
	matches := re.FindAllStringSubmatch(s, -1)
	for _, m := range matches {
		val, _ := strconv.ParseFloat(m[1], 64)
		switch {
		case strings.HasPrefix(m[2], "y"):
			total += val * 365 * 86400
		case strings.HasPrefix(m[2], "w"):
			total += val * 7 * 86400
		case strings.HasPrefix(m[2], "d"):
			total += val * 86400
		case strings.HasPrefix(m[2], "h"):
			total += val * 3600
		case strings.HasPrefix(m[2], "m"):
			total += val * 60
		case strings.HasPrefix(m[2], "s"):
			total += val
		}
	}
	return total
}

func ParseARPTable(output string) float64 {
	if m := regexp.MustCompile(`Total\s+(?:number\s+of\s+)?entries\s*:\s*(\d+)`).FindStringSubmatch(output); len(m) > 1 {
		v, _ := strconv.ParseFloat(m[1], 64)
		return v
	}
	if m := regexp.MustCompile(`(\d+)\s+(?:entries|lines)`).FindStringSubmatch(output); len(m) > 1 {
		v, _ := strconv.ParseFloat(m[1], 64)
		return v
	}
	count := 0.0
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Internet") {
			count++
		}
	}
	return count
}

func ParseRoutingTable(output string) float64 {
	if m := regexp.MustCompile(`Total\s+(?:number\s+of\s+)?(?:routes|entries)\s*:\s*(\d+)`).FindStringSubmatch(output); len(m) > 1 {
		v, _ := strconv.ParseFloat(m[1], 64)
		return v
	}
	return 0
}

func ParseNTPStatus(output string) bool {
	s := strings.ToLower(output)
	return strings.Contains(s, "synchronized") && !strings.Contains(s, "unsynchronized")
}

func ParseFlashUtilization(output string) float64 {
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

func ParseSTP(output string) IOSSTPData {
	stp := IOSSTPData{
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

func ParseVLANs(output string) IOSVLANData {
	data := IOSVLANData{}
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

func ParseMACCount(output string) float64 {
	if m := regexp.MustCompile(`Total\s+Mac\s+Addresses:\s*(\d+)`).FindStringSubmatch(output); len(m) > 1 {
		v, _ := strconv.ParseFloat(m[1], 64)
		return v
	}
	if m := regexp.MustCompile(`(\d+)\s+entries`).FindStringSubmatch(output); len(m) > 1 {
		v, _ := strconv.ParseFloat(m[1], 64)
		return v
	}
	return 0
}

type IOSSTPData struct {
	BlockedPorts map[string]float64
	TopoChanges  map[string]float64
}

type IOSVLANData struct {
	VLANCount  int
	ActiveVLAN int
}
