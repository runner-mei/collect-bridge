package discovery

import (
	"commons"
	"commons/netutils"
	"errors"
	"fmt"
	"net"
	"snmp"
	"strings"
	"sync"
	"time"
)

const (
	DEBUG         = "[DEBUG]"
	WARN          = "[WARN]"
	ERROR         = "[ERROR]"
	FATAL         = "[FATAL]"
	INFO          = "[INFO]"
	END_TOKEN     = "end"
	TIMEOUT_TOKEN = "timeout"
)

type replyResult struct {
	ok  string
	err error
}
type Discoverer struct {
	ch          chan string
	drv_ch      chan Device
	control_ch  chan map[string]interface{}
	result_ch   chan error
	isCompleted bool

	params *DiscoveryParams

	//icmp_pinger *netutils.Pingers
	snmp_pinger *snmp.Pingers
	snmp_drv    commons.Driver
	metrics_drv commons.Driver

	devices    map[string]Device
	ip2managed map[string]string

	range_scanned map[string]int
	lock          sync.Mutex
	is_running    int32
}

func NewDiscoverer(params *DiscoveryParams, drvMgr *commons.DriverManager) (*Discoverer, error) {

	if nil == params {
		return nil, errors.New("params is nil.")
	}
	if 0 >= params.Timeout {
		params.Timeout = 10
	}

	if nil == params.Communities {
		params.Communities = []string{"public"}
	} else {
		isFound := false
		for _, s := range params.Communities {
			if s == "public" {
				isFound = true
			}
		}

		if !isFound {
			params.Communities = append(params.Communities, "public")
		}
	}

	// icmp_pinger, err := netutils.NewPingers(nil, 10000)
	// if nil != err {
	//	return nil, errors.New("icmp failed, " + err.Error())
	// }

	snmp_pinger := snmp.NewPingers(10000)

	defer func() {
		if nil != snmp_pinger {
			snmp_pinger.Close()
		}
	}()

	for _, community := range params.Communities {
		err := snmp_pinger.Listen("udp4", "0.0.0.0:0", snmp.SNMP_V2C, community)
		if nil != err {
			return nil, errors.New("snmp failed, " + err.Error())
		}
	}

	snmp_drv, ok := drvMgr.Connect("snmp")
	if !ok {
		return nil, errors.New("snmp failed, driver is not found.")
	}

	metrics_drv, ok := drvMgr.Connect("metrics")
	if !ok {
		return nil, errors.New("metrics failed, driver is not found.")
	}

	discoverer := &Discoverer{params: params,
		ch:         make(chan string, 1000),
		drv_ch:     make(chan Device),
		control_ch: make(chan map[string]interface{}, 100),
		result_ch:  make(chan error, 1),
		//icmp_pinger: icmp_pinger,
		snmp_pinger: snmp_pinger,
		snmp_drv:    snmp_drv,
		metrics_drv: metrics_drv,

		devices:       make(map[string]Device),
		ip2managed:    make(map[string]string),
		range_scanned: make(map[string]int),
		is_running:    1}

	if 0 == params.Depth {
		params.Depth = 2
	}

	go discoverer.serve()

	for i := 0; i < 5; i++ {
		go discoverer.pollAddress()
	}

	snmp_pinger = nil
	return discoverer, nil
}

func (self *Discoverer) readMetric(drv Device, name string) (interface{}, error) {
	params := drv["$access_param"]
	if nil == params {
		return nil, fmt.Errorf("access params of %v is not exists.", drv["address"])
	}

	access_param, _ := params.([]interface{})
	if nil == access_param || 0 == len(access_param) {
		return nil, fmt.Errorf("access params of %v is empty.", drv["address"])
	}
	for _, param := range access_param {
		p, _ := param.(map[string]interface{})
		if nil == p || "snmp_param" != p["type"] {
			continue
		}
		metric_params := map[string]string{"id": drv["address"].(string), "metric": name, "charset": "GB18030"}
		for k, v := range p {
			metric_params["snmp."+k] = fmt.Sprint(v)
		}

		res := self.metrics_drv.Get(metric_params)
		if res.HasError() {
			return nil, res.Error()
		}
		return res.InterfaceValue(), nil
	}
	return nil, fmt.Errorf("snmp params of %v is not exists.", drv["address"])
}

func (self *Discoverer) readLocal() ([]interface{}, error) {
	interfaces := make([]interface{}, 0)
	ifs, e := net.Interfaces()
	if nil != e {
		return nil, e
	}

	for _, f := range ifs {
		addrs, e := f.Addrs()
		if nil != e {
			return nil, e
		}

		for _, addr := range addrs {
			interfaces = append(interfaces, map[string]interface{}{"ifIndex": f.Index,
				"ifMtu":         f.MTU,
				"ifDescr":       f.Name,
				"address":       addr.String(),
				"ifPhysAddress": f.HardwareAddr.String()})
		}
	}
	return interfaces, nil
}

func (self *Discoverer) initDevice(drv Device) error {
	// read basic attributes
	for name, alias := range map[string]string{"sys.oid": "oid", "sys.descr": "description",
		"sys.type":       "catalog",
		"sys.services":   "services",
		"sys.name":       "name",
		"sys.location":   "location",
		"interfaceDescr": "$interface",
		"ipAddress":      "$address"} {
		self.logf(DEBUG, "read '%s' for '%v", name, drv["address"])
		metric, e := self.readMetric(drv, name)
		if nil != e {
			self.logf(ERROR, "read %s of '%v' failed, %v.", name, drv["address"], e)
		} else if nil == metric {
			self.logf(ERROR, "read %s of '%v' failed, result is nil.", name, drv["address"])
		} else {
			self.logf(DEBUG, "read '%s' for '%v' successed", name, drv["address"])
			drv[alias] = metric
		}
	}

	return nil
}

func (self *Discoverer) log(level string, message string) {
	self.ch <- time.Now().String() + level + " " + message
}

func (self *Discoverer) logf(level string, format string, params ...interface{}) {
	self.ch <- fmt.Sprintf(time.Now().String()+level+" "+format, params...)
}

func (self *Discoverer) addDevice(drv Device) {
	addr := drv["address"].(string)
	self.devices[addr] = drv

	ipAddresses, ok := drv["$address"].(map[string]interface{})
	if !ok || nil == ipAddresses || 0 == len(ipAddresses) {
		return
	}

	for _, r := range ipAddresses {
		if row, ok := r.(map[string]interface{}); ok && nil != row {
			self.ip2managed[row["address"].(string)] = addr
		}
	}
}

func (self *Discoverer) is_scanned(ip_range string) bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	_, ok := self.range_scanned[ip_range]
	return ok
}
func (self *Discoverer) already_scanned(ip_range string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.range_scanned[ip_range] = 0
}

func (self *Discoverer) detectNewAddress(table interface{}) {
	if nil == table {
		return
	}

	ip_list := map[string]int{}

	if addresses, ok := table.([]interface{}); ok {
		for _, ifs := range addresses {
			row, ok := ifs.(map[string]interface{})
			if !ok {
				self.logf(FATAL, "detectNewAddress() - it is not map[string]interface{}, actual is %T.", ifs)
				continue
			}
			ip_list[row["address"].(string)] = 0
		}
	} else if addresses, ok := table.(map[string]interface{}); ok {
		for k, ifs := range addresses {
			row, ok := ifs.(map[string]interface{})
			if !ok {
				self.logf(FATAL, "detectNewAddress() - %s is not map[string]interface{}, actual is %T.", k, ifs)
				continue
			}
			ip_list[row["address"].(string)] = 0
		}
	}
	range_list := make([]string, 0, len(ip_list))
	for ip, _ := range ip_list {
		range_list = append(range_list, ip+"/24")
	}
	self.detectNewRange(range_list)
}

func (self *Discoverer) detectNewRange(range_list []string) {
	for _, ip := range range_list {
		if 0 == len(ip) {
			continue
		}
		if netutils.IsInvalidAddress(ip) {
			self.log(DEBUG, "skip invalid address - "+ip)
			continue
		}

		ip_range, err := netutils.ParseIPRange(ip)
		if nil != err {
			self.log(DEBUG, "parse ip range '"+ip+"' failed, "+err.Error())
			continue
		}

		if self.is_scanned(ip_range.String()) {
			self.log(DEBUG, "ip range '"+ip+"' is scanned")
			continue
		}

		self.log(DEBUG, "scan ip range '"+ip_range.String()+"'")
		for i := 0; i < self.snmp_pinger.Length(); i++ {
			for ip_range.HasNext() {
				addr := ip_range.Current().String()
				err := self.snmp_pinger.Send(i, net.JoinHostPort(addr, "161"))
				if nil != err {
					self.log(DEBUG, "send snmp to '"+addr+"' failed, "+err.Error())
				}
			}
			ip_range.Reset()
		}

		self.already_scanned(ip_range.String())
	}
}

func (self *Discoverer) isExists(ip string) bool {
	_, ok := self.devices[ip]
	if ok {
		return false
	}
	_, ok = self.ip2managed[ip]
	if ok {
		return false
	}
	return true
}

func (self *Discoverer) serve() {
	defer func() {
		self.ch <- END_TOKEN
	}()

	if self.params.IsReadLocal {
		local, e := self.readLocal()
		if nil != e {
			self.log(FATAL, e.Error())
			return
		}
		self.detectNewAddress(local)
	}
	self.detectNewRange(self.params.IP_Range)

	for d := 1; ; d++ {
		pending_drvs := make([]Device, 0, 10)
		running := true
		for running {

			select {
			case cmd := <-self.control_ch:
				c := cmd["command"]
				switch c {
				case "ping_failed":
					running = false
				case "new_device":
				}
			case drv := <-self.drv_ch:
				self.addDevice(drv)
				pending_drvs = append(pending_drvs, drv)
			case <-time.After(time.Duration(self.params.Timeout) * time.Second):
				running = false
			}
		}

		if 0 == len(pending_drvs) {
			self.log(INFO, "pending device is empty and exit.")
			break
		}
		if d >= self.params.Depth {
			self.logf(INFO, "Reach the specified depth '%d' and exit.", self.params.Depth)
			break
		}
		for _, drv := range pending_drvs {
			self.detectNewAddress(drv["$address"])
		}
	}
}

func (self *Discoverer) pollAddress() {
	for {
		reply := <-self.snmp_pinger.GetChannel()
		if nil == reply {
			break // pinger is exit
		}
		if nil != reply.Error {
			self.log(ERROR, "recv icmp failed - "+reply.Error.Error())

			self.control_ch <- map[string]interface{}{"command": "ping_failed"}
			break
		}

		self.log(ERROR, "new address - "+reply.Addr.String())
		addr := reply.Addr.String()
		port := "161"

		self.control_ch <- map[string]interface{}{"command": "new_device", "address": addr}

		idx := strings.IndexRune(addr, ':')
		if -1 != idx {
			port = addr[idx+1:]
			addr = addr[0:idx]
		}

		if netutils.IsInvalidAddress(addr) {
			self.log(DEBUG, "skip invalid address - "+addr)
			continue
		}

		if !self.isExists(addr) {
			self.log(DEBUG, "skip old address - "+addr)
			continue
		}

		drv := Device{"address": addr, "$access_param": []interface{}{map[string]interface{}{"type": "snmp_param", "address": addr,
			"port": port, "version": reply.Version.String(), "community": reply.Community}}}
		e := self.initDevice(drv)
		if nil != e {
			self.log(ERROR, "init device '"+addr+":"+port+"' failed, "+e.Error())
		} else {
			self.log(INFO, "new device '"+addr+":"+port+"'")
		}

		if nil != drv {
			self.drv_ch <- drv
		}
	}
}

func (self *Discoverer) Result() map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range self.devices {
		result[k] = v
	}
	return result
}

func (self *Discoverer) IsCompleted() bool {
	return self.isCompleted
}

func (self *Discoverer) Control(params map[string]interface{}) error {
	self.control_ch <- params
	return <-self.result_ch
}

func (self *Discoverer) Read(timeout time.Duration) string {
	select {
	case res := <-self.ch:
		if END_TOKEN == res {
			self.isCompleted = true
		}
		return res
	case <-time.After(timeout):
		return TIMEOUT_TOKEN
	}
	return ""
}

func (self *Discoverer) Close() {
	//self.icmp_pinger.Close()
	self.snmp_pinger.Close()
}