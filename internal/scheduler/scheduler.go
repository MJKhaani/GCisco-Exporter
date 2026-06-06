package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/MJKhaani/GCiscoExporter/internal/cache"
	"github.com/MJKhaani/GCiscoExporter/internal/config"
	"github.com/MJKhaani/GCiscoExporter/internal/debugcapture"
	"github.com/MJKhaani/GCiscoExporter/internal/metrics"
	"github.com/MJKhaani/GCiscoExporter/internal/provider/json"
	"github.com/MJKhaani/GCiscoExporter/internal/provider/netconf"
	"github.com/MJKhaani/GCiscoExporter/internal/provider/restconf"
	"github.com/MJKhaani/GCiscoExporter/internal/provider/ssh"
	"github.com/MJKhaani/GCiscoExporter/internal/worker"
)

type Scheduler struct {
	cfg       *config.Config
	store     *cache.Store
	collector *metrics.Collector
	debugCap  *debugcapture.Capture
	worker    *worker.Pool
}

func New(cfg *config.Config, store *cache.Store, collector *metrics.Collector, debugCap *debugcapture.Capture) *Scheduler {
	return &Scheduler{
		cfg:       cfg,
		store:     store,
		collector: collector,
		debugCap:  debugCap,
		worker:    worker.New(cfg.Workers, cfg),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	interval, err := time.ParseDuration(s.cfg.CollectionInterval)
	if err != nil {
		log.Printf("Invalid collection interval %q, defaulting to 60s", s.cfg.CollectionInterval)
		interval = 60 * time.Second
	}

	go func() {
		s.runCollection(ctx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runCollection(ctx)
			}
		}
	}()
}

func (s *Scheduler) runCollection(ctx context.Context) {
	log.Printf("Starting collection cycle for %d devices", len(s.cfg.Devices))

	start := time.Now()
	results := make(map[string]bool)
	var mu sync.Mutex

	for _, device := range s.cfg.Devices {
		select {
		case <-ctx.Done():
			return
		default:
		}

		d := device
		s.worker.Submit(ctx, func() {
			deviceStart := time.Now()
			err := s.collectDevice(ctx, &d)

			mu.Lock()
			results[d.Name] = err == nil
			mu.Unlock()

			duration := time.Since(deviceStart).Seconds()
			s.collector.SetCollectionDuration(d.Name, d.Host, duration)
			s.collector.SetCollectionSuccess(d.Name, d.Host, err == nil)

			if err != nil {
				log.Printf("Failed to collect from %s (%s): %v", d.Name, d.Host, err)
			} else {
				s.collector.SetLastCollection(d.Name, d.Host, float64(time.Now().Unix()))
			}
		})
	}

	s.worker.Wait()

	totalDuration := time.Since(start).Seconds()
	successCount := 0
	for _, ok := range results {
		if ok {
			successCount++
		}
	}
	log.Printf("Collection cycle complete: %d/%d devices successful in %.2fs", successCount, len(s.cfg.Devices), totalDuration)
}

func (s *Scheduler) collectDevice(ctx context.Context, device *config.Device) error {
	cred := s.findCredential(device.CredentialName)
	if cred == nil {
		return fmt.Errorf("credential %q not found for device %s", device.CredentialName, device.Name)
	}

	timeout := s.cfg.TimeoutDuration()

	for _, method := range device.CollectionMethods {
		var providerName string
		var result map[string]interface{}
		var err error

		switch method {
		case "json":
			providerName = "json"
			result, err = s.collectJSON(ctx, device, cred, timeout)
		case "ssh":
			providerName = "ssh"
			result, err = s.collectSSH(ctx, device, cred, timeout)
		case "restconf":
			providerName = "restconf"
			result, err = s.collectRESTCONF(ctx, device, cred, timeout)
		case "netconf":
			providerName = "netconf"
			result, err = s.collectNETCONF(ctx, device, cred, timeout)
		default:
			err = fmt.Errorf("unknown collection method: %s", method)
		}

		if err != nil {
			log.Printf("Collection method %s failed for %s: %v", method, device.Name, err)
			continue
		}

		s.store.Update(device.Name, result)
		s.collector.SetDeviceUp(device.Name, device.Host, device.Type, true)
		s.collector.SetCollectionMethod(device.Name, device.Host, s.methodToFloat(providerName))
		return nil
	}

	s.collector.SetDeviceUp(device.Name, device.Host, device.Type, false)
	return fmt.Errorf("all collection methods failed for %s", device.Name)
}

func (s *Scheduler) collectJSON(ctx context.Context, device *config.Device, cred *config.Credential, timeout time.Duration) (map[string]interface{}, error) {
	client, err := ssh.NewClient(device, cred, timeout, s.cfg.LegacyCiphers)
	if err != nil {
		return nil, fmt.Errorf("SSH connect: %w", err)
	}
	defer client.Close()

	provider := json.New(client, device)
	ms := s.cfg.MetricSelection
	result := make(map[string]interface{})

	if ms.HardwareHealth {
		sys, err := provider.CollectSystem(ctx)
		if err != nil {
			return nil, fmt.Errorf("collect system: %w", err)
		}
		s.collector.SetDeviceInfo(device.Name, device.Host, sys.Model, sys.Version, sys.Serial)
		s.collector.SetDeviceUptime(device.Name, device.Host, sys.Uptime)
		result["system"] = sys

		res, err := provider.CollectResources(ctx)
		if err != nil {
			log.Printf("Warning: collect resources (JSON) failed for %s: %v", device.Name, err)
		} else {
			s.collector.SetCPUUsage(device.Name, device.Host, res.CPUUsage)
			s.collector.SetMemoryTotal(device.Name, device.Host, res.MemTotal)
			s.collector.SetMemoryUsed(device.Name, device.Host, res.MemUsed)
			s.collector.SetMemoryFree(device.Name, device.Host, res.MemFree)
		}
		result["resources"] = res

		procs, err := provider.CollectProcesses(ctx, device.ProcessLimit)
		if err != nil {
			log.Printf("Warning: collect processes (JSON) failed for %s: %v", device.Name, err)
		} else {
			for _, proc := range procs {
				s.collector.SetProcessCPU(device.Name, device.Host, proc.Name, proc.CPU)
				s.collector.SetProcessRuntime(device.Name, device.Host, proc.Name, proc.Runtime)
			}
		}
		result["processes"] = procs

		hw, err := provider.CollectHardware(ctx)
		if err != nil {
			log.Printf("Warning: collect hardware (JSON) failed for %s: %v", device.Name, err)
		} else {
			for sensor, temp := range hw.Temperature {
				s.collector.SetTemperature(device.Name, device.Host, sensor, temp)
			}
			for fan, ok := range hw.FanStatus {
				s.collector.SetFanStatus(device.Name, device.Host, fan, ok)
			}
			for ps, ok := range hw.PowerSupply {
				s.collector.SetPowerSupplyStatus(device.Name, device.Host, ps, ok)
			}
			s.collector.SetPowerRedundancyStatus(device.Name, device.Host, hw.PowerRedundant)
			for mod, ok := range hw.ModuleStatus {
				s.collector.SetModuleStatus(device.Name, device.Host, mod, ok)
			}
			for member, active := range hw.StackMemberStatus {
				s.collector.SetStackMemberStatus(device.Name, device.Host, member, active)
			}
		}
		result["hardware"] = hw
	}

	if ms.Interfaces {
		ifaces, err := provider.CollectInterfaces(ctx)
		if err != nil {
			log.Printf("Warning: collect interfaces (JSON) failed for %s: %v", device.Name, err)
		}
		for _, iface := range ifaces {
			s.collector.SetInterfaceAdminUp(device.Name, device.Host, iface.Name, iface.AdminUp)
			s.collector.SetInterfaceOperUp(device.Name, device.Host, iface.Name, iface.OperUp)
			s.collector.SetInterfaceRxBytes(device.Name, device.Host, iface.Name, iface.RxBytes)
			s.collector.SetInterfaceTxBytes(device.Name, device.Host, iface.Name, iface.TxBytes)
			s.collector.SetInterfaceRxErrors(device.Name, device.Host, iface.Name, iface.RxErrors)
			s.collector.SetInterfaceTxErrors(device.Name, device.Host, iface.Name, iface.TxErrors)
			s.collector.SetInterfaceRxCrcErrors(device.Name, device.Host, iface.Name, iface.RxCrcErrors)
			s.collector.SetInterfaceRxFrameErrors(device.Name, device.Host, iface.Name, iface.RxFrame)
			s.collector.SetInterfaceRxRunts(device.Name, device.Host, iface.Name, iface.RxRunts)
			s.collector.SetInterfaceRxGiants(device.Name, device.Host, iface.Name, iface.RxGiants)
			s.collector.SetInterfaceRxDrops(device.Name, device.Host, iface.Name, iface.RxDrops)
			s.collector.SetInterfaceTxDrops(device.Name, device.Host, iface.Name, iface.TxDrops)
			s.collector.SetInterfaceRxDiscards(device.Name, device.Host, iface.Name, iface.RxDiscards)
			s.collector.SetInterfaceTxDiscards(device.Name, device.Host, iface.Name, iface.TxDiscards)
			s.collector.SetInterfaceResets(device.Name, device.Host, iface.Name, iface.Resets)
			s.collector.SetInterfaceSpeed(device.Name, device.Host, iface.Name, iface.Speed)
			s.collector.SetInterfaceMtu(device.Name, device.Host, iface.Name, iface.Mtu)
		}
		result["interfaces"] = ifaces

		optics, err := provider.CollectOptics(ctx)
		if err != nil {
			log.Printf("Warning: collect optics (JSON) failed for %s: %v", device.Name, err)
		}
		for iface, optic := range optics {
			s.collector.SetOpticRxPower(device.Name, device.Host, iface, optic.RxPower)
			s.collector.SetOpticTxPower(device.Name, device.Host, iface, optic.TxPower)
			s.collector.SetOpticTemperature(device.Name, device.Host, iface, optic.Temp)
			s.collector.SetOpticVoltage(device.Name, device.Host, iface, optic.Voltage)
			s.collector.SetOpticBiasCurrent(device.Name, device.Host, iface, optic.BiasCurrent)
		}
		result["optics"] = optics
	}

	if ms.Layer2Stability {
		stp, err := provider.CollectSTP(ctx)
		if err != nil {
			log.Printf("Warning: collect STP (JSON) failed for %s: %v", device.Name, err)
		} else {
			for vlan, count := range stp.RootChanges {
				s.collector.SetSTPRootChanges(device.Name, device.Host, vlan, count)
			}
			for vlan, count := range stp.TopoChanges {
				s.collector.SetSTPTopologyChanges(device.Name, device.Host, vlan, count)
			}
			for vlan, count := range stp.BlockedPorts {
				s.collector.SetSTPBlockedPorts(device.Name, device.Host, vlan, count)
			}
		}
		result["stp"] = stp

		mac, err := provider.CollectMAC(ctx)
		if err != nil {
			log.Printf("Warning: collect MAC (JSON) failed for %s: %v", device.Name, err)
		} else {
			s.collector.SetMACTableCount(device.Name, device.Host, mac.TableCount)
			s.collector.SetMACTableUtilization(device.Name, device.Host, mac.TableUtilization)
			for vlan, count := range mac.Moves {
				s.collector.SetMACMoves(device.Name, device.Host, vlan, count)
			}
		}
		result["mac"] = mac

		vlans, err := provider.CollectVLANs(ctx)
		if err != nil {
			log.Printf("Warning: collect VLANs (JSON) failed for %s: %v", device.Name, err)
		} else {
			s.collector.SetVLANCount(device.Name, device.Host, float64(vlans.Count))
			s.collector.SetVLANActiveCount(device.Name, device.Host, float64(vlans.Active))
		}
		result["vlans"] = vlans
	}

	if ms.EtherChannel {
		pcs, err := provider.CollectPortChannels(ctx)
		if err != nil {
			log.Printf("Warning: collect port-channels (JSON) failed for %s: %v", device.Name, err)
		} else {
			for pc, up := range pcs.Status {
				s.collector.SetPortChannelStatus(device.Name, device.Host, pc, up)
			}
			for pc, members := range pcs.MemberStatus {
				for member, active := range members {
					s.collector.SetPortChannelMemberStatus(device.Name, device.Host, pc, member, active)
				}
			}
			for pc, count := range pcs.SuspendedMembers {
				s.collector.SetPortChannelSuspendedMembers(device.Name, device.Host, pc, count)
			}
		}
		result["portchannels"] = pcs
	}

	if ms.Layer3Tables {
		l3, err := provider.CollectL3Tables(ctx)
		if err != nil {
			log.Printf("Warning: collect L3 tables (JSON) failed for %s: %v", device.Name, err)
		} else {
			s.collector.SetARPTableCount(device.Name, device.Host, l3.ARPCount)
			s.collector.SetARPTableUtilization(device.Name, device.Host, l3.ARPUtilization)
			s.collector.SetRoutingTableUtilization(device.Name, device.Host, l3.RoutingUtilization)
			s.collector.SetFIBUtilization(device.Name, device.Host, l3.FIBUtilization)
			s.collector.SetAdjacencyTableUtilization(device.Name, device.Host, l3.AdjacencyUtilization)
		}
		result["l3tables"] = l3
	}

	if ms.SystemMaintenance {
		sysInfo, err := provider.CollectSystemInfo(ctx)
		if err != nil {
			log.Printf("Warning: collect system info (JSON) failed for %s: %v", device.Name, err)
		} else {
			s.collector.SetNTPSynced(device.Name, device.Host, sysInfo.NTPSynced)
			s.collector.SetFlashUtilization(device.Name, device.Host, sysInfo.FlashUtil)
		}
		result["sysinfo"] = sysInfo
	}

	if ms.CapacityTrends {
		cap, err := provider.CollectCapacity(ctx)
		if err != nil {
			log.Printf("Warning: collect capacity (JSON) failed for %s: %v", device.Name, err)
		} else {
			s.collector.SetPortUtilizationRatio(device.Name, device.Host, cap.PortUtilizationRatio)
			s.collector.SetTransceiverCount(device.Name, device.Host, float64(cap.TransceiverCount))
			for pool, util := range cap.BufferUtilization {
				s.collector.SetBufferUtilization(device.Name, device.Host, pool, util)
			}
		}
		result["capacity"] = cap
	}

	return result, nil
}

func (s *Scheduler) collectSSH(ctx context.Context, device *config.Device, cred *config.Credential, timeout time.Duration) (map[string]interface{}, error) {
	client, err := ssh.NewClient(device, cred, timeout, s.cfg.LegacyCiphers)
	if err != nil {
		return nil, fmt.Errorf("SSH connect: %w", err)
	}
	defer client.Close()

	ms := s.cfg.MetricSelection
	result := make(map[string]interface{})

	if ms.HardwareHealth {
		ver, err := client.Execute(ctx, "show version")
		if err != nil {
			return nil, fmt.Errorf("show version: %w", err)
		}
		result["version_raw"] = ver

		res, err := client.Execute(ctx, "show processes cpu")
		if err != nil {
			log.Printf("Warning: show processes cpu failed for %s: %v", device.Name, err)
		}
		result["processes_raw"] = res

		env, err := client.Execute(ctx, "show environment")
		if err != nil {
			log.Printf("Warning: show environment failed for %s: %v", device.Name, err)
		}
		result["environment_raw"] = env
	}

	if ms.Interfaces {
		ifaces, err := client.Execute(ctx, "show interface")
		if err != nil {
			return nil, fmt.Errorf("show interface: %w", err)
		}
		result["interfaces_raw"] = ifaces

		trans, err := client.Execute(ctx, "show interface transceiver details")
		if err != nil {
			log.Printf("Warning: show transceiver failed for %s: %v", device.Name, err)
		}
		result["transceiver_raw"] = trans
	}

	if ms.Layer2Stability {
		stpSum, err := client.Execute(ctx, "show spanning-tree summary")
		if err != nil {
			log.Printf("Warning: show spanning-tree summary failed for %s: %v", device.Name, err)
		}
		result["stp_summary_raw"] = stpSum

		stpDetail, err := client.Execute(ctx, "show spanning-tree detail")
		if err != nil {
			log.Printf("Warning: show spanning-tree detail failed for %s: %v", device.Name, err)
		}
		result["stp_detail_raw"] = stpDetail

		macCount, err := client.Execute(ctx, "show mac address-table count")
		if err != nil {
			log.Printf("Warning: show mac count failed for %s: %v", device.Name, err)
		}
		result["mac_count_raw"] = macCount

		vlanBrief, err := client.Execute(ctx, "show vlan brief")
		if err != nil {
			log.Printf("Warning: show vlan brief failed for %s: %v", device.Name, err)
		}
		result["vlan_brief_raw"] = vlanBrief
	}

	if ms.EtherChannel {
		ecSum, err := client.Execute(ctx, "show etherchannel summary")
		if err != nil {
			log.Printf("Warning: show etherchannel summary failed for %s: %v", device.Name, err)
		}
		result["etherchannel_summary_raw"] = ecSum
	}

	if ms.Layer3Tables {
		arpSum, err := client.Execute(ctx, "show ip arp summary")
		if err != nil {
			log.Printf("Warning: show arp summary failed for %s: %v", device.Name, err)
		}
		result["arp_summary_raw"] = arpSum

		routeSum, err := client.Execute(ctx, "show ip route summary")
		if err != nil {
			log.Printf("Warning: show route summary failed for %s: %v", device.Name, err)
		}
		result["route_summary_raw"] = routeSum

		cefSum, err := client.Execute(ctx, "show ip cef summary")
		if err != nil {
			log.Printf("Warning: show cef summary failed for %s: %v", device.Name, err)
		}
		result["cef_summary_raw"] = cefSum

		adjSum, err := client.Execute(ctx, "show adjacency summary")
		if err != nil {
			log.Printf("Warning: show adjacency summary failed for %s: %v", device.Name, err)
		}
		result["adjacency_summary_raw"] = adjSum
	}

	if ms.SystemMaintenance {
		ntpStatus, err := client.Execute(ctx, "show ntp status")
		if err != nil {
			log.Printf("Warning: show ntp status failed for %s: %v", device.Name, err)
		}
		result["ntp_status_raw"] = ntpStatus

		fs, err := client.Execute(ctx, "show file systems")
		if err != nil {
			log.Printf("Warning: show file systems failed for %s: %v", device.Name, err)
		}
		result["filesystems_raw"] = fs
	}

	if ms.CapacityTrends {
		buf, err := client.Execute(ctx, "show buffers")
		if err != nil {
			log.Printf("Warning: show buffers failed for %s: %v", device.Name, err)
		}
		result["buffers_raw"] = buf
	}

	return result, nil
}

func (s *Scheduler) collectRESTCONF(ctx context.Context, device *config.Device, cred *config.Credential, timeout time.Duration) (map[string]interface{}, error) {
	provider := restconf.New(device, cred)
	ms := s.cfg.MetricSelection
	result := make(map[string]interface{})

	if ms.HardwareHealth {
		sys, err := provider.CollectSystem(ctx)
		if err != nil {
			return nil, fmt.Errorf("collect system: %w", err)
		}
		s.collector.SetDeviceInfo(device.Name, device.Host, sys.Model, sys.Version, sys.Serial)
		s.collector.SetDeviceUptime(device.Name, device.Host, sys.Uptime)
		result["system"] = sys

		res, err := provider.CollectResources(ctx)
		if err != nil {
			log.Printf("Warning: collect resources (RESTCONF) failed for %s: %v", device.Name, err)
		} else {
			s.collector.SetCPUUsage(device.Name, device.Host, res.CPUUsage)
			s.collector.SetMemoryTotal(device.Name, device.Host, res.MemTotal)
			s.collector.SetMemoryUsed(device.Name, device.Host, res.MemUsed)
			s.collector.SetMemoryFree(device.Name, device.Host, res.MemFree)
		}
		result["resources"] = res

		procs, err := provider.CollectProcesses(ctx, device.ProcessLimit)
		if err != nil {
			log.Printf("Warning: collect processes (RESTCONF) failed for %s: %v", device.Name, err)
		} else {
			for _, proc := range procs {
				s.collector.SetProcessCPU(device.Name, device.Host, proc.Name, proc.CPU)
				s.collector.SetProcessRuntime(device.Name, device.Host, proc.Name, proc.Runtime)
			}
		}
		result["processes"] = procs
	}

	if ms.Interfaces {
		ifaces, err := provider.CollectInterfaces(ctx)
		if err != nil {
			log.Printf("Warning: collect interfaces (RESTCONF) failed for %s: %v", device.Name, err)
		}
		for _, iface := range ifaces {
			s.collector.SetInterfaceAdminUp(device.Name, device.Host, iface.Name, iface.AdminUp)
			s.collector.SetInterfaceOperUp(device.Name, device.Host, iface.Name, iface.OperUp)
			s.collector.SetInterfaceRxBytes(device.Name, device.Host, iface.Name, iface.RxBytes)
			s.collector.SetInterfaceTxBytes(device.Name, device.Host, iface.Name, iface.TxBytes)
			s.collector.SetInterfaceRxErrors(device.Name, device.Host, iface.Name, iface.RxErrors)
			s.collector.SetInterfaceTxErrors(device.Name, device.Host, iface.Name, iface.TxErrors)
		}
		result["interfaces"] = ifaces
	}

	return result, nil
}

func (s *Scheduler) collectNETCONF(ctx context.Context, device *config.Device, cred *config.Credential, timeout time.Duration) (map[string]interface{}, error) {
	provider := netconf.New(device, cred)

	if err := provider.Connect(ctx); err != nil {
		return nil, fmt.Errorf("NETCONF connect: %w", err)
	}
	defer provider.Close()

	ms := s.cfg.MetricSelection
	result := make(map[string]interface{})

	if ms.HardwareHealth {
		sys, err := provider.CollectSystem(ctx)
		if err != nil {
			return nil, fmt.Errorf("collect system: %w", err)
		}
		s.collector.SetDeviceInfo(device.Name, device.Host, sys.Model, sys.Version, sys.Serial)
		s.collector.SetDeviceUptime(device.Name, device.Host, sys.Uptime)
		result["system"] = sys

		res, err := provider.CollectResources(ctx)
		if err != nil {
			log.Printf("Warning: collect resources (NETCONF) failed for %s: %v", device.Name, err)
		} else {
			s.collector.SetCPUUsage(device.Name, device.Host, res.CPUUsage)
			s.collector.SetMemoryTotal(device.Name, device.Host, res.MemTotal)
			s.collector.SetMemoryUsed(device.Name, device.Host, res.MemUsed)
			s.collector.SetMemoryFree(device.Name, device.Host, res.MemFree)
		}
		result["resources"] = res

		procs, err := provider.CollectProcesses(ctx, device.ProcessLimit)
		if err != nil {
			log.Printf("Warning: collect processes (NETCONF) failed for %s: %v", device.Name, err)
		} else {
			for _, proc := range procs {
				s.collector.SetProcessCPU(device.Name, device.Host, proc.Name, proc.CPU)
				s.collector.SetProcessRuntime(device.Name, device.Host, proc.Name, proc.Runtime)
			}
		}
		result["processes"] = procs
	}

	if ms.Interfaces {
		ifaces, err := provider.CollectInterfaces(ctx)
		if err != nil {
			log.Printf("Warning: collect interfaces (NETCONF) failed for %s: %v", device.Name, err)
		}
		for _, iface := range ifaces {
			s.collector.SetInterfaceAdminUp(device.Name, device.Host, iface.Name, iface.AdminUp)
			s.collector.SetInterfaceOperUp(device.Name, device.Host, iface.Name, iface.OperUp)
			s.collector.SetInterfaceRxBytes(device.Name, device.Host, iface.Name, iface.RxBytes)
			s.collector.SetInterfaceTxBytes(device.Name, device.Host, iface.Name, iface.TxBytes)
			s.collector.SetInterfaceRxErrors(device.Name, device.Host, iface.Name, iface.RxErrors)
			s.collector.SetInterfaceTxErrors(device.Name, device.Host, iface.Name, iface.TxErrors)
		}
		result["interfaces"] = ifaces
	}

	return result, nil
}

func (s *Scheduler) findCredential(name string) *config.Credential {
	for i := range s.cfg.Credentials {
		if s.cfg.Credentials[i].Name == name {
			return &s.cfg.Credentials[i]
		}
	}
	if name == "" && len(s.cfg.Credentials) > 0 {
		return &s.cfg.Credentials[0]
	}
	return nil
}

func (s *Scheduler) methodToFloat(method string) float64 {
	switch method {
	case "json":
		return 1
	case "restconf":
		return 2
	case "netconf":
		return 3
	case "ssh":
		return 4
	default:
		return 0
	}
}
