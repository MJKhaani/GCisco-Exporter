# Cisco Switch Monitoring Metrics

This guide lists the most important metrics to monitor on Cisco switches running IOS, IOS-XE, and NX-OS. It focuses on operational health, interface stability, Layer 2 behavior, and vPC readiness.

## 1) Hardware Health

**Metrics**
- CPU utilization
- Memory utilization
- Memory free
- Temperature sensors
- Fan status
- Power supply status
- Power redundancy status
- Supervisor status
- Line card/module status
- Stack member status

**What it tells you**
Hardware problems usually show up here first. Rising CPU or memory pressure, fan failures, PSU issues, or temperature alarms can lead to instability, reloads, or service impact.

**Sample commands**
```bash
show processes cpu sorted
show processes cpu history
show processes memory
show environment all
show inventory
show module
show version
show switch
```

---

## 2) Interfaces

**Metrics**
- Interface operational status
- Interface administrative status
- Interface flaps
- Interface availability
- Interface utilization (ingress)
- Interface utilization (egress)
- Input errors
- Output errors
- CRC errors
- Frame errors
- Runts
- Giants
- Input drops
- Output drops
- Input discards
- Output discards
- Queue drops
- Interface resets
- MTU mismatches
- Optical TX power
- Optical RX power
- Optical temperature
- Optical voltage
- Optical bias current

**What it tells you**
Interface metrics show physical issues, congestion, bad optics, cabling problems, duplex or MTU mismatches, and traffic drops before users notice a service outage.

**Sample commands**
```bash
show interfaces status
show interfaces
show interfaces counters errors
show interfaces counters
show interfaces counters detailed
show interfaces description
show interfaces transceiver details
show interfaces transceiver diagnostics
```

---

## 3) Layer 2 Stability

**Metrics**
- STP root changes
- STP topology changes
- STP port state changes
- STP blocked ports
- MAC table utilization
- MAC move events
- MAC flapping events
- VLAN utilization
- VLAN status
- BPDU Guard events
- Loop Guard events
- UDLD status
- UDLD events

**What it tells you**
Layer 2 monitoring helps detect loops, mispatches, unstable access ports, and unexpected topology changes.

**Sample commands**
```bash
show spanning-tree
show spanning-tree summary
show spanning-tree detail
show mac address-table count
show mac address-table dynamic
show vlan brief
show udld neighbors
show udld interface
show errdisable recovery
```

---

## 4) EtherChannel / Port-Channel

**Metrics**
- Port-channel status
- LACP status
- LACP member status
- Suspended members
- Port-channel utilization
- Port-channel errors

**What it tells you**
These metrics confirm that bundled links are healthy, balanced, and fully participating in the channel.

**Sample commands**
```bash
show etherchannel summary
show etherchannel detail
show interfaces port-channel <id>
show lacp neighbor
show lacp counters
show port-channel traffic
```

---

## 5) Layer 3 Tables

**Metrics**
- ARP table utilization
- ARP entry count
- Routing table utilization
- FIB/CEF utilization
- Adjacency table utilization

**What it tells you**
These are important when the switch participates in Layer 3 forwarding or when large table growth could affect scale.

**Sample commands**
```bash
show ip arp
show ip arp summary
show ip route summary
show ip cef summary
show adjacency summary
show forwarding distribution summary
```

---

## 6) Routing Protocols

**Metrics**
- OSPF neighbor state
- OSPF adjacency count
- EIGRP neighbor state
- EIGRP adjacency count
- BGP neighbor state
- BGP prefix count
- BGP route flaps
- ISIS neighbor state

**What it tells you**
Routing protocol health confirms the switch is learning and advertising paths correctly.

**Sample commands**
```bash
show ip ospf neighbor
show ip ospf neighbor detail
show ip eigrp neighbors
show ip bgp summary
show ip bgp neighbors
show isis neighbors
```

---

## 7) First Hop Redundancy

**Metrics**
- HSRP state
- HSRP role changes
- VRRP state
- VRRP role changes
- GLBP state
- GLBP role changes

**What it tells you**
These metrics matter when the switch provides default gateway redundancy.

**Sample commands**
```bash
show standby brief
show standby
show vrrp brief
show vrrp
show glbp brief
show glbp
```

---

## 8) NX-OS / vPC

**Metrics**
- vPC peer status
- vPC peer-link status
- vPC keepalive status
- vPC consistency status
- vPC Type-1 inconsistencies
- vPC Type-2 inconsistencies
- vPC orphan ports
- vPC member status
- vPC VLAN consistency
- FEX status
- FEX fabric interface status

**What it tells you**
These are critical for NX-OS designs using vPC. Peer-link or keepalive failures can cause major traffic disruption.

**Sample commands**
```bash
show vpc brief
show vpc consistency-parameters global
show vpc consistency-parameters interface port-channel <id>
show vpc role
show vpc peer-keepalive
show port-channel summary
show fex
show fex detail
```

---

## 9) Security / Control Plane

**Metrics**
- Control Plane CPU utilization
- CoPP drops
- DHCP snooping events
- Dynamic ARP Inspection events
- Port-security violations
- ACL drop counters

**What it tells you**
These metrics help spot attack traffic, misconfigured hosts, and control-plane overload.

**Sample commands**
```bash
show policy-map control-plane
show policy-map interface control-plane
show ip dhcp snooping
show ip arp inspection
show port-security
show access-lists
show control-plane host open-ports
```

---

## 10) System / Maintenance

**Metrics**
- Syslog critical events
- Syslog warning events
- NTP synchronization status
- Configuration change events
- Device uptime
- Reload events
- Crash events
- License status
- Flash/storage utilization

**What it tells you**
These metrics support basic operational hygiene and help correlate incidents with changes or reloads.

**Sample commands**
```bash
show logging
show ntp status
show clock detail
show archive log config all
show version
show reload
show logging last 50
show file systems
show license status
```

---

## 11) Capacity / Trend Metrics

**Metrics**
- CPU trend
- Memory trend
- Interface utilization trend
- MAC table utilization trend
- ARP table utilization trend
- Routing table utilization trend
- VLAN count
- Port utilization ratio
- Transceiver inventory utilization
- Buffer utilization
- Queue utilization

**What it tells you**
Trend metrics show when the switch is approaching limits, which helps you plan upgrades before problems happen.

**Sample commands**
```bash
show processes cpu history
show processes memory sorted
show interfaces counters rate
show mac address-table count
show ip arp summary
show ip route summary
show vlan summary
show buffers
show platform hardware capacity
```

---
