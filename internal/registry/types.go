package registry

type ServiceRef struct {
	ServiceName string
	Group       string
	Clusters    []string
	HealthyOnly bool
	Tags        map[string]string
}

type Instance struct {
	ServiceName string
	Group       string
	Cluster     string
	IP          string
	Port        uint64
	Weight      float64
	Healthy     bool
	Enabled     bool
	Ephemeral   bool
	Metadata    map[string]string
}

func (i Instance) Address() string {
	if i.IP == "" || i.Port == 0 {
		return ""
	}
	return i.IP + ":" + uintToString(i.Port)
}

func uintToString(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for v > 0 {
		pos--
		buf[pos] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[pos:])
}
