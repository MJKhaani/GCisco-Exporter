# Cisco Switch Exporter v2

A production-grade Prometheus exporter for Cisco NX-OS, IOS-XE, and IOS devices.

## Overview

A single exporter that collects metrics from hundreds/thousands of Cisco switches via multiple collection methods, caches them in the background, and exposes them as a single Prometheus scrape target.

## Features

- **Multi-device support**: NX-OS, IOS-XE, IOS
- **Multiple collection methods**: JSON, RESTCONF, NETCONF, SSH CLI (with automatic fallback)
- **Background collection**: Prometheus reads from cache, never triggers live device collection
- **Worker-pool architecture**: Concurrent collection using Go routines
- **Golden parser tests**: Regression tests for every parser
- **Debug capture mode**: Stores failed command outputs for troubleshooting

## Collection Priority

1. JSON (preferred for NX-OS)
2. RESTCONF (preferred for IOS-XE)
3. NETCONF (secondary for IOS-XE)
4. SSH CLI (universal fallback)

Per-device overrides are supported.

## Architecture

```
Prometheus → Exporter → Cache → Background Scheduler → Worker Pool → Collection Providers → Devices
```

## Quick Start

### Build

```bash
go build -o cisco-exporter ./cmd/exporter
```

### Configuration

Copy `config.example.yaml` to `config.yaml` and edit:

```bash
cp config.example.yaml config.yaml
```

### Run

```bash
./cisco-exporter -config config.yaml
```

### Prometheus Scrape

```yaml
scrape_configs:
  - job_name: 'cisco-exporter'
    static_configs:
      - targets: ['localhost:9427']
    scrape_interval: 30s
```

## Metrics

### System
- `cisco_device_up` - Device reachability
- `cisco_device_uptime_seconds` - Device uptime
- `cisco_device_info` - Device information (model, version, serial)

### CPU
- `cisco_cpu_usage_percent` - CPU usage
- `cisco_cpu_5sec_percent` - 5-second CPU average
- `cisco_cpu_1min_percent` - 1-minute CPU average
- `cisco_cpu_5min_percent` - 5-minute CPU average

### Memory
- `cisco_memory_total_bytes` - Total memory
- `cisco_memory_used_bytes` - Used memory
- `cisco_memory_free_bytes` - Free memory

### Interfaces
- `cisco_interface_admin_up` - Admin status
- `cisco_interface_oper_up` - Operational status
- `cisco_interface_rx_bytes_total` - RX bytes
- `cisco_interface_tx_bytes_total` - TX bytes
- `cisco_interface_rx_errors_total` - RX errors
- `cisco_interface_tx_errors_total` - TX errors

### Trunks
- `cisco_trunk_enabled` - Trunk enabled
- `cisco_trunk_operational` - Trunk operational
- `cisco_trunk_native_vlan` - Native VLAN
- `cisco_trunk_allowed_vlan_count` - Allowed VLAN count

### Processes
- `cisco_process_cpu_percent` - Process CPU usage
- `cisco_process_runtime_seconds` - Process runtime

### Environment
- `cisco_temperature_celsius` - Temperature
- `cisco_fan_status` - Fan status
- `cisco_power_supply_status` - Power supply status

### Optics
- `cisco_optic_rx_power_dbm` - RX power
- `cisco_optic_tx_power_dbm` - TX power
- `cisco_optic_temperature_celsius` - Optic temperature

### Exporter
- `cisco_exporter_last_collection_timestamp` - Last collection time
- `cisco_exporter_collection_duration_seconds` - Collection duration
- `cisco_exporter_collection_success` - Collection success

## Project Structure

```
cmd/exporter/           - Main entry point
internal/config/        - Configuration parsing
internal/cache/         - Metrics cache
internal/scheduler/     - Background collection scheduler
internal/worker/        - Worker pool
internal/capabilities/  - Device capability detection
internal/debugcapture/  - Debug capture mode
internal/metrics/       - Prometheus metrics registration
internal/provider/      - Collection providers
  json/                 - JSON provider (NX-OS)
  ssh/                  - SSH CLI provider
  restconf/             - RESTCONF provider (IOS-XE)
  netconf/              - NETCONF provider (IOS-XE)
internal/collector/     - Metric collectors
  system/               - System metrics
  interfaces/           - Interface metrics
  trunks/               - Trunk metrics
  processes/            - Process metrics
  environment/          - Environment metrics
  optics/               - Optic metrics
testdata/               - Golden parser tests
  nxos/                 - NX-OS test data
  ios/                  - IOS test data
  iosxe/                - IOS-XE test data
```

## Testing

```bash
go test ./...
```

## License

MIT
