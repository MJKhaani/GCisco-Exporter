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

	ifaceAdminUp   *prometheus.GaugeVec
	ifaceOperUp    *prometheus.GaugeVec
	ifaceRxBytes   *prometheus.CounterVec
	ifaceTxBytes   *prometheus.CounterVec
	ifaceRxErrors  *prometheus.CounterVec
	ifaceTxErrors  *prometheus.CounterVec

	trunkEnabled      *prometheus.GaugeVec
	trunkOperational  *prometheus.GaugeVec
	trunkNativeVLAN   *prometheus.GaugeVec
	trunkAllowedCount *prometheus.GaugeVec

	procCPU      *prometheus.GaugeVec
	procRuntime  *prometheus.GaugeVec

	temperature    *prometheus.GaugeVec
	fanStatus      *prometheus.GaugeVec
	powerSupply    *prometheus.GaugeVec

	opticRxPower    *prometheus.GaugeVec
	opticTxPower    *prometheus.GaugeVec
	opticTemp       *prometheus.GaugeVec

	collectionMethod       *prometheus.GaugeVec
	lastCollection         *prometheus.GaugeVec
	collectionDuration     *prometheus.GaugeVec
	collectionSuccess      *prometheus.GaugeVec

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

		trunkEnabled: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_trunk_enabled",
			Help: "Trunk enabled (1=enabled, 0=disabled)",
		}, []string{"device", "host", "interface"}),

		trunkOperational: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_trunk_operational",
			Help: "Trunk operational (1=up, 0=down)",
		}, []string{"device", "host", "interface"}),

		trunkNativeVLAN: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_trunk_native_vlan",
			Help: "Trunk native VLAN ID",
		}, []string{"device", "host", "interface"}),

		trunkAllowedCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_trunk_allowed_vlan_count",
			Help: "Number of allowed VLANs on trunk",
		}, []string{"device", "host", "interface"}),

		procCPU: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_process_cpu_percent",
			Help: "Process CPU usage percentage",
		}, []string{"device", "host", "process"}),

		procRuntime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_process_runtime_seconds",
			Help: "Process runtime in seconds",
		}, []string{"device", "host", "process"}),

		temperature: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_temperature_celsius",
			Help: "Temperature in celsius",
		}, []string{"device", "host", "sensor"}),

		fanStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Help: "Fan status (1=ok, 0=failed)",
			Name: "cisco_fan_status",
		}, []string{"device", "host", "fan"}),

		powerSupply: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cisco_power_supply_status",
			Help: "Power supply status (1=ok, 0=failed)",
		}, []string{"device", "host", "supply"}),

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
		c.ifaceRxErrors, c.ifaceTxErrors,
		c.trunkEnabled, c.trunkOperational, c.trunkNativeVLAN, c.trunkAllowedCount,
		c.procCPU, c.procRuntime,
		c.temperature, c.fanStatus, c.powerSupply,
		c.opticRxPower, c.opticTxPower, c.opticTemp,
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

func (c *Collector) SetTrunkEnabled(device, host, iface string, enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if enabled {
		c.trunkEnabled.WithLabelValues(device, host, iface).Set(1)
	} else {
		c.trunkEnabled.WithLabelValues(device, host, iface).Set(0)
	}
}

func (c *Collector) SetTrunkOperational(device, host, iface string, up bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if up {
		c.trunkOperational.WithLabelValues(device, host, iface).Set(1)
	} else {
		c.trunkOperational.WithLabelValues(device, host, iface).Set(0)
	}
}

func (c *Collector) SetTrunkNativeVLAN(device, host, iface string, vlan float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trunkNativeVLAN.WithLabelValues(device, host, iface).Set(vlan)
}

func (c *Collector) SetTrunkAllowedCount(device, host, iface string, count float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trunkAllowedCount.WithLabelValues(device, host, iface).Set(count)
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
