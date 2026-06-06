package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	registry *prometheus.Registry

	deviceUp     *prometheus.GaugeVec
	deviceUptime *prometheus.GaugeVec
	deviceInfo   *prometheus.GaugeVec

	cpuUsage *prometheus.GaugeVec
	cpu5Sec  *prometheus.GaugeVec
	cpu1Min  *prometheus.GaugeVec
	cpu5Min  *prometheus.GaugeVec

	memTotal *prometheus.GaugeVec
	memUsed  *prometheus.GaugeVec
	memFree  *prometheus.GaugeVec

	ifaceAdminUp     *prometheus.GaugeVec
	ifaceOperUp      *prometheus.GaugeVec
	ifaceRxBytes     *prometheus.CounterVec
	ifaceTxBytes     *prometheus.CounterVec
	ifaceRxErrors    *prometheus.CounterVec
	ifaceTxErrors    *prometheus.CounterVec
	ifaceRxCrcErrors *prometheus.CounterVec
	ifaceRxFrame     *prometheus.CounterVec
	ifaceRxRunts     *prometheus.CounterVec
	ifaceRxGiants    *prometheus.CounterVec
	ifaceRxDrops     *prometheus.CounterVec
	ifaceTxDrops     *prometheus.CounterVec
	ifaceRxDiscards  *prometheus.CounterVec
	ifaceTxDiscards  *prometheus.CounterVec
	ifaceResets      *prometheus.CounterVec
	ifaceFlaps       *prometheus.CounterVec
	ifaceSpeed       *prometheus.GaugeVec
	ifaceMtu         *prometheus.GaugeVec
	ifaceQueueDrops  *prometheus.CounterVec

	opticRxPower    *prometheus.GaugeVec
	opticTxPower    *prometheus.GaugeVec
	opticTemp       *prometheus.GaugeVec
	opticVoltage    *prometheus.GaugeVec
	opticBiasCurrent *prometheus.GaugeVec

	stpRootChanges      *prometheus.CounterVec
	stpTopoChanges      *prometheus.CounterVec
	stpPortStateChanges *prometheus.CounterVec
	stpBlockedPorts     *prometheus.GaugeVec
	bpduGuardEvents     *prometheus.CounterVec

	macTableUtilization *prometheus.GaugeVec
	macTableCount       *prometheus.GaugeVec
	macMoves            *prometheus.CounterVec

	vlanCount      *prometheus.GaugeVec
	vlanActiveCount *prometheus.GaugeVec

	pcStatus          *prometheus.GaugeVec
	pcMemberStatus    *prometheus.GaugeVec
	pcSuspendedMembers *prometheus.GaugeVec
	pcUtilization     *prometheus.GaugeVec
	pcRxErrors        *prometheus.CounterVec
	pcTxErrors        *prometheus.CounterVec

	arpTableUtilization *prometheus.GaugeVec
	arpTableCount       *prometheus.GaugeVec
	routingTableUtil    *prometheus.GaugeVec
	fibUtilization      *prometheus.GaugeVec
	adjacencyTableUtil  *prometheus.GaugeVec

	temperature    *prometheus.GaugeVec
	fanStatus      *prometheus.GaugeVec
	powerSupply    *prometheus.GaugeVec
	powerRedundancy *prometheus.GaugeVec
	moduleStatus   *prometheus.GaugeVec
	stackMemberStatus *prometheus.GaugeVec

	ntpSynced        *prometheus.GaugeVec
	configChanges    *prometheus.CounterVec
	reloadEvents     *prometheus.CounterVec
	licenseStatus    *prometheus.GaugeVec
	flashUtilization *prometheus.GaugeVec

	portUtilizationRatio    *prometheus.GaugeVec
	transceiverCount        *prometheus.GaugeVec
	bufferUtilization       *prometheus.GaugeVec
	queueUtilization        *prometheus.GaugeVec

	procCPU     *prometheus.GaugeVec
	procRuntime *prometheus.GaugeVec

	collectionMethod   *prometheus.GaugeVec
	lastCollection     *prometheus.GaugeVec
	collectionDuration *prometheus.GaugeVec
	collectionSuccess  *prometheus.GaugeVec

	mu sync.RWMutex
}

func NewCollector() *Collector {
	reg := prometheus.NewRegistry()

	c := &Collector{
		registry: reg,

		deviceUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_device_up",
			Help: "Device reachability (1=up, 0=down)",
		}, []string{"device", "host", "type"}),

		deviceUptime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_device_uptime_seconds",
			Help: "Device uptime in seconds",
		}, []string{"device", "host"}),

		deviceInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_device_info",
			Help: "Device information",
		}, []string{"device", "host", "model", "version", "serial"}),

		cpuUsage: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_cpu_usage_percent",
			Help: "CPU usage percentage",
		}, []string{"device", "host"}),

		cpu5Sec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_cpu_5sec_percent",
			Help: "CPU 5-second average percentage",
		}, []string{"device", "host"}),

		cpu1Min: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_cpu_1min_percent",
			Help: "CPU 1-minute average percentage",
		}, []string{"device", "host"}),

		cpu5Min: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_cpu_5min_percent",
			Help: "CPU 5-minute average percentage",
		}, []string{"device", "host"}),

		memTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_memory_total_bytes",
			Help: "Total memory in bytes",
		}, []string{"device", "host"}),

		memUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_memory_used_bytes",
			Help: "Used memory in bytes",
		}, []string{"device", "host"}),

		memFree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_memory_free_bytes",
			Help: "Free memory in bytes",
		}, []string{"device", "host"}),

		ifaceAdminUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_interface_admin_up",
			Help: "Interface admin status (1=up, 0=down)",
		}, []string{"device", "host", "interface"}),

		ifaceOperUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_interface_oper_up",
			Help: "Interface operational status (1=up, 0=down)",
		}, []string{"device", "host", "interface"}),

		ifaceRxBytes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_rx_bytes_total",
			Help: "Total received bytes",
		}, []string{"device", "host", "interface"}),

		ifaceTxBytes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_tx_bytes_total",
			Help: "Total transmitted bytes",
		}, []string{"device", "host", "interface"}),

		ifaceRxErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_rx_errors_total",
			Help: "Total receive errors",
		}, []string{"device", "host", "interface"}),

		ifaceTxErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_tx_errors_total",
			Help: "Total transmit errors",
		}, []string{"device", "host", "interface"}),

		ifaceRxCrcErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_rx_crc_errors_total",
			Help: "Total CRC errors",
		}, []string{"device", "host", "interface"}),

		ifaceRxFrame: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_rx_frame_errors_total",
			Help: "Total frame errors",
		}, []string{"device", "host", "interface"}),

		ifaceRxRunts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_rx_runts_total",
			Help: "Total runt frames",
		}, []string{"device", "host", "interface"}),

		ifaceRxGiants: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_rx_giants_total",
			Help: "Total giant frames",
		}, []string{"device", "host", "interface"}),

		ifaceRxDrops: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_rx_drops_total",
			Help: "Total input drops",
		}, []string{"device", "host", "interface"}),

		ifaceTxDrops: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_tx_drops_total",
			Help: "Total output drops",
		}, []string{"device", "host", "interface"}),

		ifaceRxDiscards: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_rx_discards_total",
			Help: "Total input discards",
		}, []string{"device", "host", "interface"}),

		ifaceTxDiscards: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_tx_discards_total",
			Help: "Total output discards",
		}, []string{"device", "host", "interface"}),

		ifaceResets: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_resets_total",
			Help: "Total interface resets",
		}, []string{"device", "host", "interface"}),

		ifaceFlaps: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_flaps_total",
			Help: "Total interface flaps",
		}, []string{"device", "host", "interface"}),

		ifaceSpeed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_interface_speed_bps",
			Help: "Interface speed in bits per second",
		}, []string{"device", "host", "interface"}),

		ifaceMtu: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_interface_mtu",
			Help: "Interface MTU",
		}, []string{"device", "host", "interface"}),

		ifaceQueueDrops: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_interface_queue_drops_total",
			Help: "Total queue drops",
		}, []string{"device", "host", "interface", "queue"}),

		opticRxPower: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_optic_rx_power_dbm",
			Help: "Optic RX power in dBm",
		}, []string{"device", "host", "interface"}),

		opticTxPower: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_optic_tx_power_dbm",
			Help: "Optic TX power in dBm",
		}, []string{"device", "host", "interface"}),

		opticTemp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_optic_temperature_celsius",
			Help: "Optic temperature in celsius",
		}, []string{"device", "host", "interface"}),

		opticVoltage: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_optic_voltage",
			Help: "Optic voltage",
		}, []string{"device", "host", "interface"}),

		opticBiasCurrent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_optic_bias_current_ma",
			Help: "Optic bias current in mA",
		}, []string{"device", "host", "interface"}),

		stpRootChanges: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_stp_root_changes_total",
			Help: "Total STP root bridge changes",
		}, []string{"device", "host", "vlan"}),

		stpTopoChanges: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_stp_topology_changes_total",
			Help: "Total STP topology changes",
		}, []string{"device", "host", "vlan"}),

		stpPortStateChanges: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_stp_port_state_changes_total",
			Help: "Total STP port state changes",
		}, []string{"device", "host", "interface", "vlan"}),

		stpBlockedPorts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_stp_blocked_ports",
			Help: "Number of STP blocked ports",
		}, []string{"device", "host", "vlan"}),

		bpduGuardEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_bpdu_guard_events_total",
			Help: "Total BPDU Guard events",
		}, []string{"device", "host", "interface"}),

		macTableUtilization: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_mac_table_utilization_percent",
			Help: "MAC address table utilization percentage",
		}, []string{"device", "host"}),

		macTableCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_mac_table_count",
			Help: "Number of MAC address table entries",
		}, []string{"device", "host"}),

		macMoves: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_mac_moves_total",
			Help: "Total MAC address moves",
		}, []string{"device", "host", "vlan"}),

		vlanCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_vlan_count",
			Help: "Total number of VLANs",
		}, []string{"device", "host"}),

		vlanActiveCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_vlan_active_count",
			Help: "Number of active VLANs",
		}, []string{"device", "host"}),

		pcStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_portchannel_status",
			Help: "Port-channel status (1=up, 0=down)",
		}, []string{"device", "host", "portchannel"}),

		pcMemberStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_portchannel_member_status",
			Help: "Port-channel member status (1=active, 0=inactive/suspended)",
		}, []string{"device", "host", "portchannel", "member"}),

		pcSuspendedMembers: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_portchannel_suspended_members",
			Help: "Number of suspended port-channel members",
		}, []string{"device", "host", "portchannel"}),

		pcUtilization: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_portchannel_utilization_percent",
			Help: "Port-channel utilization percentage",
		}, []string{"device", "host", "portchannel", "direction"}),

		pcRxErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_portchannel_rx_errors_total",
			Help: "Port-channel receive errors",
		}, []string{"device", "host", "portchannel"}),

		pcTxErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_portchannel_tx_errors_total",
			Help: "Port-channel transmit errors",
		}, []string{"device", "host", "portchannel"}),

		arpTableUtilization: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_arp_table_utilization_percent",
			Help: "ARP table utilization percentage",
		}, []string{"device", "host"}),

		arpTableCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_arp_table_count",
			Help: "Number of ARP table entries",
		}, []string{"device", "host"}),

		routingTableUtil: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_routing_table_utilization_percent",
			Help: "Routing table utilization percentage",
		}, []string{"device", "host"}),

		fibUtilization: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_fib_utilization_percent",
			Help: "FIB/CEF utilization percentage",
		}, []string{"device", "host"}),

		adjacencyTableUtil: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_adjacency_table_utilization_percent",
			Help: "Adjacency table utilization percentage",
		}, []string{"device", "host"}),

		temperature: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_temperature_celsius",
			Help: "Temperature in celsius",
		}, []string{"device", "host", "sensor"}),

		fanStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_fan_status",
			Help: "Fan status (1=ok, 0=failed)",
		}, []string{"device", "host", "fan"}),

		powerSupply: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_power_supply_status",
			Help: "Power supply status (1=ok, 0=failed)",
		}, []string{"device", "host", "supply"}),

		powerRedundancy: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_power_redundancy_status",
			Help: "Power redundancy status (1=redundant, 0=non-redundant)",
		}, []string{"device", "host"}),

		moduleStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_module_status",
			Help: "Module/linecard status (1=ok, 0=failed)",
		}, []string{"device", "host", "module"}),

		stackMemberStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_stack_member_status",
			Help: "Stack member status (1=active, 0=inactive)",
		}, []string{"device", "host", "member"}),

		ntpSynced: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_ntp_synced",
			Help: "NTP synchronization status (1=synced, 0=not synced)",
		}, []string{"device", "host"}),

		configChanges: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_config_changes_total",
			Help: "Total configuration changes",
		}, []string{"device", "host"}),

		reloadEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cisco_reload_events_total",
			Help: "Total reload events",
		}, []string{"device", "host"}),

		licenseStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_license_status",
			Help: "License status (1=valid, 0=invalid/expired)",
		}, []string{"device", "host", "feature"}),

		flashUtilization: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_flash_utilization_percent",
			Help: "Flash/storage utilization percentage",
		}, []string{"device", "host"}),

		portUtilizationRatio: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_port_utilization_ratio",
			Help: "Ratio of active ports to total ports",
		}, []string{"device", "host"}),

		transceiverCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_transceiver_count",
			Help: "Number of installed transceivers",
		}, []string{"device", "host"}),

		bufferUtilization: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_buffer_utilization_percent",
			Help: "Buffer utilization percentage",
		}, []string{"device", "host", "pool"}),

		queueUtilization: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_queue_utilization_percent",
			Help: "Queue utilization percentage",
		}, []string{"device", "host", "queue"}),

		procCPU: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_process_cpu_percent",
			Help: "Process CPU usage percentage",
		}, []string{"device", "host", "process"}),

		procRuntime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_process_runtime_seconds",
			Help: "Process runtime in seconds",
		}, []string{"device", "host", "process"}),

		collectionMethod: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_collection_method",
			Help: "Collection method used (1=json, 2=restconf, 3=netconf, 4=ssh)",
		}, []string{"device", "host"}),

		lastCollection: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_exporter_last_collection_timestamp",
			Help: "Timestamp of last successful collection",
		}, []string{"device", "host"}),

		collectionDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_exporter_collection_duration_seconds",
			Help: "Duration of last collection in seconds",
		}, []string{"device", "host"}),

		collectionSuccess: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_exporter_collection_success",
			Help: "Whether last collection was successful (1=yes, 0=no)",
		}, []string{"device", "host"}),
	}

	reg.MustRegister(
		c.deviceUp, c.deviceUptime, c.deviceInfo,
		c.cpuUsage, c.cpu5Sec, c.cpu1Min, c.cpu5Min,
		c.memTotal, c.memUsed, c.memFree,
		c.ifaceAdminUp, c.ifaceOperUp, c.ifaceRxBytes, c.ifaceTxBytes,
		c.ifaceRxErrors, c.ifaceTxErrors, c.ifaceRxCrcErrors, c.ifaceRxFrame,
		c.ifaceRxRunts, c.ifaceRxGiants, c.ifaceRxDrops, c.ifaceTxDrops,
		c.ifaceRxDiscards, c.ifaceTxDiscards, c.ifaceResets, c.ifaceFlaps,
		c.ifaceSpeed, c.ifaceMtu, c.ifaceQueueDrops,
		c.opticRxPower, c.opticTxPower, c.opticTemp, c.opticVoltage, c.opticBiasCurrent,
		c.stpRootChanges, c.stpTopoChanges, c.stpPortStateChanges, c.stpBlockedPorts, c.bpduGuardEvents,
		c.macTableUtilization, c.macTableCount, c.macMoves,
		c.vlanCount, c.vlanActiveCount,
		c.pcStatus, c.pcMemberStatus, c.pcSuspendedMembers, c.pcUtilization, c.pcRxErrors, c.pcTxErrors,
		c.arpTableUtilization, c.arpTableCount, c.routingTableUtil, c.fibUtilization, c.adjacencyTableUtil,
		c.temperature, c.fanStatus, c.powerSupply, c.powerRedundancy, c.moduleStatus, c.stackMemberStatus,
		c.ntpSynced, c.configChanges, c.reloadEvents, c.licenseStatus, c.flashUtilization,
		c.portUtilizationRatio, c.transceiverCount, c.bufferUtilization, c.queueUtilization,
		c.procCPU, c.procRuntime,
		c.collectionMethod, c.lastCollection, c.collectionDuration, c.collectionSuccess,
	)

	return c
}

func (c *Collector) Registry() *prometheus.Registry {
	return c.registry
}

func (c *Collector) SetDeviceUp(device, host, deviceType string, up bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if up {
		c.deviceUp.WithLabelValues(device, host, deviceType).Set(1)
	} else {
		c.deviceUp.WithLabelValues(device, host, deviceType).Set(0)
	}
}

func (c *Collector) SetDeviceUptime(device, host string, seconds float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deviceUptime.WithLabelValues(device, host).Set(seconds)
}

func (c *Collector) SetDeviceInfo(device, host, model, version, serial string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deviceInfo.WithLabelValues(device, host, model, version, serial).Set(1)
}

func (c *Collector) SetCPUUsage(device, host string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cpuUsage.WithLabelValues(device, host).Set(percent)
}

func (c *Collector) SetCPU5Sec(device, host string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cpu5Sec.WithLabelValues(device, host).Set(percent)
}

func (c *Collector) SetCPU1Min(device, host string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cpu1Min.WithLabelValues(device, host).Set(percent)
}

func (c *Collector) SetCPU5Min(device, host string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cpu5Min.WithLabelValues(device, host).Set(percent)
}

func (c *Collector) SetMemoryTotal(device, host string, bytes float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.memTotal.WithLabelValues(device, host).Set(bytes)
}

func (c *Collector) SetMemoryUsed(device, host string, bytes float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.memUsed.WithLabelValues(device, host).Set(bytes)
}

func (c *Collector) SetMemoryFree(device, host string, bytes float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.memFree.WithLabelValues(device, host).Set(bytes)
}

func (c *Collector) SetInterfaceAdminUp(device, host, iface string, up bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if up {
		c.ifaceAdminUp.WithLabelValues(device, host, iface).Set(1)
	} else {
		c.ifaceAdminUp.WithLabelValues(device, host, iface).Set(0)
	}
}

func (c *Collector) SetInterfaceOperUp(device, host, iface string, up bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if up {
		c.ifaceOperUp.WithLabelValues(device, host, iface).Set(1)
	} else {
		c.ifaceOperUp.WithLabelValues(device, host, iface).Set(0)
	}
}

func (c *Collector) SetInterfaceRxBytes(device, host, iface string, bytes float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceRxBytes.WithLabelValues(device, host, iface).Add(bytes)
}

func (c *Collector) SetInterfaceTxBytes(device, host, iface string, bytes float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceTxBytes.WithLabelValues(device, host, iface).Add(bytes)
}

func (c *Collector) SetInterfaceRxErrors(device, host, iface string, errors float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceRxErrors.WithLabelValues(device, host, iface).Add(errors)
}

func (c *Collector) SetInterfaceTxErrors(device, host, iface string, errors float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceTxErrors.WithLabelValues(device, host, iface).Add(errors)
}

func (c *Collector) SetInterfaceRxCrcErrors(device, host, iface string, errors float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceRxCrcErrors.WithLabelValues(device, host, iface).Add(errors)
}

func (c *Collector) SetInterfaceRxFrameErrors(device, host, iface string, errors float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceRxFrame.WithLabelValues(device, host, iface).Add(errors)
}

func (c *Collector) SetInterfaceRxRunts(device, host, iface string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceRxRunts.WithLabelValues(device, host, iface).Add(count)
}

func (c *Collector) SetInterfaceRxGiants(device, host, iface string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceRxGiants.WithLabelValues(device, host, iface).Add(count)
}

func (c *Collector) SetInterfaceRxDrops(device, host, iface string, drops float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceRxDrops.WithLabelValues(device, host, iface).Add(drops)
}

func (c *Collector) SetInterfaceTxDrops(device, host, iface string, drops float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceTxDrops.WithLabelValues(device, host, iface).Add(drops)
}

func (c *Collector) SetInterfaceRxDiscards(device, host, iface string, discards float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceRxDiscards.WithLabelValues(device, host, iface).Add(discards)
}

func (c *Collector) SetInterfaceTxDiscards(device, host, iface string, discards float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceTxDiscards.WithLabelValues(device, host, iface).Add(discards)
}

func (c *Collector) SetInterfaceResets(device, host, iface string, resets float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceResets.WithLabelValues(device, host, iface).Add(resets)
}

func (c *Collector) SetInterfaceFlaps(device, host, iface string, flaps float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceFlaps.WithLabelValues(device, host, iface).Add(flaps)
}

func (c *Collector) SetInterfaceSpeed(device, host, iface string, speed float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceSpeed.WithLabelValues(device, host, iface).Set(speed)
}

func (c *Collector) SetInterfaceMtu(device, host, iface string, mtu float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceMtu.WithLabelValues(device, host, iface).Set(mtu)
}

func (c *Collector) SetInterfaceQueueDrops(device, host, iface, queue string, drops float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ifaceQueueDrops.WithLabelValues(device, host, iface, queue).Add(drops)
}

func (c *Collector) SetOpticRxPower(device, host, iface string, dbm float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.opticRxPower.WithLabelValues(device, host, iface).Set(dbm)
}

func (c *Collector) SetOpticTxPower(device, host, iface string, dbm float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.opticTxPower.WithLabelValues(device, host, iface).Set(dbm)
}

func (c *Collector) SetOpticTemperature(device, host, iface string, celsius float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.opticTemp.WithLabelValues(device, host, iface).Set(celsius)
}

func (c *Collector) SetOpticVoltage(device, host, iface string, voltage float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.opticVoltage.WithLabelValues(device, host, iface).Set(voltage)
}

func (c *Collector) SetOpticBiasCurrent(device, host, iface string, ma float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.opticBiasCurrent.WithLabelValues(device, host, iface).Set(ma)
}

func (c *Collector) SetSTPRootChanges(device, host, vlan string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stpRootChanges.WithLabelValues(device, host, vlan).Add(count)
}

func (c *Collector) SetSTPTopologyChanges(device, host, vlan string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stpTopoChanges.WithLabelValues(device, host, vlan).Add(count)
}

func (c *Collector) SetSTPPortStateChanges(device, host, iface, vlan string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stpPortStateChanges.WithLabelValues(device, host, iface, vlan).Add(count)
}

func (c *Collector) SetSTPBlockedPorts(device, host, vlan string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stpBlockedPorts.WithLabelValues(device, host, vlan).Set(count)
}

func (c *Collector) SetBPDUGuardEvents(device, host, iface string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bpduGuardEvents.WithLabelValues(device, host, iface).Add(count)
}

func (c *Collector) SetMACTableUtilization(device, host string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.macTableUtilization.WithLabelValues(device, host).Set(percent)
}

func (c *Collector) SetMACTableCount(device, host string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.macTableCount.WithLabelValues(device, host).Set(count)
}

func (c *Collector) SetMACMoves(device, host, vlan string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.macMoves.WithLabelValues(device, host, vlan).Add(count)
}

func (c *Collector) SetVLANCount(device, host string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.vlanCount.WithLabelValues(device, host).Set(count)
}

func (c *Collector) SetVLANActiveCount(device, host string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.vlanActiveCount.WithLabelValues(device, host).Set(count)
}

func (c *Collector) SetPortChannelStatus(device, host, pc string, up bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if up {
		c.pcStatus.WithLabelValues(device, host, pc).Set(1)
	} else {
		c.pcStatus.WithLabelValues(device, host, pc).Set(0)
	}
}

func (c *Collector) SetPortChannelMemberStatus(device, host, pc, member string, active bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if active {
		c.pcMemberStatus.WithLabelValues(device, host, pc, member).Set(1)
	} else {
		c.pcMemberStatus.WithLabelValues(device, host, pc, member).Set(0)
	}
}

func (c *Collector) SetPortChannelSuspendedMembers(device, host, pc string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pcSuspendedMembers.WithLabelValues(device, host, pc).Set(count)
}

func (c *Collector) SetPortChannelUtilization(device, host, pc, direction string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pcUtilization.WithLabelValues(device, host, pc, direction).Set(percent)
}

func (c *Collector) SetPortChannelRxErrors(device, host, pc string, errors float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pcRxErrors.WithLabelValues(device, host, pc).Add(errors)
}

func (c *Collector) SetPortChannelTxErrors(device, host, pc string, errors float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pcTxErrors.WithLabelValues(device, host, pc).Add(errors)
}

func (c *Collector) SetARPTableUtilization(device, host string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.arpTableUtilization.WithLabelValues(device, host).Set(percent)
}

func (c *Collector) SetARPTableCount(device, host string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.arpTableCount.WithLabelValues(device, host).Set(count)
}

func (c *Collector) SetRoutingTableUtilization(device, host string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.routingTableUtil.WithLabelValues(device, host).Set(percent)
}

func (c *Collector) SetFIBUtilization(device, host string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fibUtilization.WithLabelValues(device, host).Set(percent)
}

func (c *Collector) SetAdjacencyTableUtilization(device, host string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.adjacencyTableUtil.WithLabelValues(device, host).Set(percent)
}

func (c *Collector) SetTemperature(device, host, sensor string, celsius float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.temperature.WithLabelValues(device, host, sensor).Set(celsius)
}

func (c *Collector) SetFanStatus(device, host, fan string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ok {
		c.fanStatus.WithLabelValues(device, host, fan).Set(1)
	} else {
		c.fanStatus.WithLabelValues(device, host, fan).Set(0)
	}
}

func (c *Collector) SetPowerSupplyStatus(device, host, supply string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ok {
		c.powerSupply.WithLabelValues(device, host, supply).Set(1)
	} else {
		c.powerSupply.WithLabelValues(device, host, supply).Set(0)
	}
}

func (c *Collector) SetPowerRedundancyStatus(device, host string, redundant bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if redundant {
		c.powerRedundancy.WithLabelValues(device, host).Set(1)
	} else {
		c.powerRedundancy.WithLabelValues(device, host).Set(0)
	}
}

func (c *Collector) SetModuleStatus(device, host, module string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ok {
		c.moduleStatus.WithLabelValues(device, host, module).Set(1)
	} else {
		c.moduleStatus.WithLabelValues(device, host, module).Set(0)
	}
}

func (c *Collector) SetStackMemberStatus(device, host, member string, active bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if active {
		c.stackMemberStatus.WithLabelValues(device, host, member).Set(1)
	} else {
		c.stackMemberStatus.WithLabelValues(device, host, member).Set(0)
	}
}

func (c *Collector) SetNTPSynced(device, host string, synced bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if synced {
		c.ntpSynced.WithLabelValues(device, host).Set(1)
	} else {
		c.ntpSynced.WithLabelValues(device, host).Set(0)
	}
}

func (c *Collector) SetConfigChanges(device, host string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configChanges.WithLabelValues(device, host).Add(count)
}

func (c *Collector) SetReloadEvents(device, host string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reloadEvents.WithLabelValues(device, host).Add(count)
}

func (c *Collector) SetLicenseStatus(device, host, feature string, valid bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if valid {
		c.licenseStatus.WithLabelValues(device, host, feature).Set(1)
	} else {
		c.licenseStatus.WithLabelValues(device, host, feature).Set(0)
	}
}

func (c *Collector) SetFlashUtilization(device, host string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flashUtilization.WithLabelValues(device, host).Set(percent)
}

func (c *Collector) SetPortUtilizationRatio(device, host string, ratio float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.portUtilizationRatio.WithLabelValues(device, host).Set(ratio)
}

func (c *Collector) SetTransceiverCount(device, host string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.transceiverCount.WithLabelValues(device, host).Set(count)
}

func (c *Collector) SetBufferUtilization(device, host, pool string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bufferUtilization.WithLabelValues(device, host, pool).Set(percent)
}

func (c *Collector) SetQueueUtilization(device, host, queue string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.queueUtilization.WithLabelValues(device, host, queue).Set(percent)
}

func (c *Collector) SetProcessCPU(device, host, process string, percent float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.procCPU.WithLabelValues(device, host, process).Set(percent)
}

func (c *Collector) SetProcessRuntime(device, host, process string, seconds float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.procRuntime.WithLabelValues(device, host, process).Set(seconds)
}

func (c *Collector) SetCollectionMethod(device, host string, method float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.collectionMethod.WithLabelValues(device, host).Set(method)
}

func (c *Collector) SetLastCollection(device, host string, ts float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastCollection.WithLabelValues(device, host).Set(ts)
}

func (c *Collector) SetCollectionDuration(device, host string, seconds float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.collectionDuration.WithLabelValues(device, host).Set(seconds)
}

func (c *Collector) SetCollectionSuccess(device, host string, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if success {
		c.collectionSuccess.WithLabelValues(device, host).Set(1)
	} else {
		c.collectionSuccess.WithLabelValues(device, host).Set(0)
	}
}
