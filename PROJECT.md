# PROJECT.md

# Cisco Switch Exporter v2

## Overview

A production-grade Prometheus exporter for Cisco NX-OS, IOS-XE, and IOS devices.

Key requirements:

- Single exporter for hundreds/thousands of switches
- Single Prometheus scrape target
- SSH authentication (password or SSH key)
- NX-OS JSON support when available
- IOS-XE RESTCONF support
- IOS-XE NETCONF support
- Automatic fallback to SSH parsing
- Background collection with cached metrics
- Trunk interface metrics
- Top process metrics
- Golden parser tests
- Debug capture mode
- Worker-pool architecture using Go routines

---

## Collection Priority

Global default:

1. JSON
2. RESTCONF
3. NETCONF
4. SSH CLI

Per-device override supported.

Example:

```yaml
devices:
  - name: nxos-core
    host: 10.0.0.1
    collection_methods:
      - json
      - ssh

  - name: iosxe-core
    host: 10.0.0.2
    collection_methods:
      - restconf
      - netconf
      - ssh
```

---

## Architecture

Prometheus
↓
Exporter
↓
Cache
↓
Background Scheduler
↓
Worker Pool
↓
Collection Providers
↓
Devices

Prometheus must NEVER trigger live device collection.

---

## Supported Providers

### JSON Provider

Preferred for NX-OS.

Examples:

- show version | json
- show interface | json
- show inventory | json
- show environment | json
- show system resources | json

### RESTCONF Provider

Preferred for IOS-XE.

### NETCONF Provider

Secondary for IOS-XE.

### SSH Provider

Universal fallback.

---

## Fallback Logic

JSON
↓
RESTCONF
↓
NETCONF
↓
SSH

Metric:

```text
cisco_collection_method
```

---

## Metrics

### System

- cisco_device_up
- cisco_device_uptime_seconds
- cisco_device_info

### CPU

- cisco_cpu_usage_percent
- cisco_cpu_5sec_percent
- cisco_cpu_1min_percent
- cisco_cpu_5min_percent

### Memory

- cisco_memory_total_bytes
- cisco_memory_used_bytes
- cisco_memory_free_bytes

### Interfaces

- cisco_interface_admin_up
- cisco_interface_oper_up
- cisco_interface_rx_bytes_total
- cisco_interface_tx_bytes_total
- cisco_interface_rx_errors_total
- cisco_interface_tx_errors_total

### Trunks

- cisco_trunk_enabled
- cisco_trunk_operational
- cisco_trunk_native_vlan
- cisco_trunk_allowed_vlan_count

### Processes

- cisco_process_cpu_percent
- cisco_process_runtime_seconds

### Environment

- cisco_temperature_celsius
- cisco_fan_status
- cisco_power_supply_status

### Optics

- cisco_optic_rx_power_dbm
- cisco_optic_tx_power_dbm
- cisco_optic_temperature_celsius

---

## Background Collection

Example:

```yaml
collection_interval: 60s
workers: 50
```

Prometheus reads cache only.

Exporter metrics:

- cisco_exporter_last_collection_timestamp
- cisco_exporter_collection_duration_seconds
- cisco_exporter_collection_success

---

## Syslog

Do NOT store raw syslog messages in Prometheus.

Recommended:

Switches
→ Syslog Receiver
→ Loki
→ Grafana

Optional summary metrics only.

---

## Golden Parser Tests

Required for every parser.

Directory:

```text
testdata/
  nxos/
  ios/
  iosxe/
```

Each parser requires:

- raw output
- golden expected output
- regression test

---

## Debug Capture

```yaml
debug_capture:
  enabled: true
  path: ./captures
```

Stores failed command outputs for troubleshooting.

---

## Configuration

Single YAML configuration file.

Supports:

- devices
- credentials
- workers
- timeouts
- collection methods
- cache settings
- process limits

---

## Project Structure

```text
cmd/exporter

internal/provider/json
internal/provider/ssh
internal/provider/netconf
internal/provider/restconf

internal/cache
internal/scheduler
internal/worker
internal/capabilities
internal/debugcapture

internal/collector/system
internal/collector/interfaces
internal/collector/trunks
internal/collector/processes
internal/collector/environment
internal/collector/optics

testdata/
```

---

## Success Criteria

- Reliable NX-OS support
- Reliable IOS support
- Reliable IOS-XE support
- JSON-first collection
- RESTCONF support
- NETCONF support
- SSH fallback
- Trunk metrics exported
- Top process metrics exported
- Single scrape target
- Background collection
- Golden parser coverage
- Stable operation with 1000+ devices
