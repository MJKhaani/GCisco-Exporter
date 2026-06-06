package provider

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
	RxPower    float64
	TxPower    float64
	Temp       float64
	Voltage    float64
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
	Count   int
	Active  int
}

type PortChannelData struct {
	Status           map[string]bool
	MemberStatus     map[string]map[string]bool
	SuspendedMembers map[string]float64
	Members          map[string][]string
}

type L3TableData struct {
	ARPUtilization    float64
	ARPCount          float64
	RoutingUtilization float64
	FIBUtilization    float64
	AdjacencyUtilization float64
}

type HardwareData struct {
	Temperature     map[string]float64
	FanStatus       map[string]bool
	PowerSupply     map[string]bool
	PowerRedundant  bool
	ModuleStatus    map[string]bool
	StackMemberStatus map[string]bool
}

type SystemData2 struct {
	NTPSynced     bool
	Uptime        float64
	ReloadEvents  float64
	FlashUtil     float64
}

type CapacityData struct {
	PortUtilizationRatio float64
	TransceiverCount     int
	BufferUtilization    map[string]float64
	QueueUtilization     map[string]float64
}
