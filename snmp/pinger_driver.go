package snmp

import (
	"commons"
	"commons/errutils"
	"commons/netutils"
	"encoding/json"
	"net"
	"time"
)

type PingerDriver struct {
	drvMgr  *commons.DriverManager
	pingers map[string]*Pinger
}

func NewPingerDriver(drvMgr *commons.DriverManager) *PingerDriver {
	return &PingerDriver{drvMgr: drvMgr, pingers: make(map[string]*Pinger)}
}

// func (self *PingerDriver) Start(*Pinger) (err error) {
// 	if nil == self.pinger {
// 		self.pinger, err = NewPinger(self.network, self.laddr)
// 	}
// 	return
// }

// func (self *PingerDriver) Stop(*Pinger) {
// 	if nil != self.pinger {
// 		self.pinger.Close()
// 		self.pinger = nil
// 	}
// }

// func (self *PingerDriver) Reset() error {
// 	self.Stop()
// 	return self.Start()
// }

func (self *PingerDriver) Get(params map[string]string) (map[string]interface{}, commons.RuntimeError) {
	id, ok := params["id"]
	if !ok {
		return nil, commons.IdNotExists
	}
	pinger, ok := self.pingers[id]
	if !ok {
		return nil, errutils.RecordNotFound(id)
	}

	values := make([][2]string, 0, 10)
	for {
		addr, version, e := pinger.Recv(time.Duration(1))
		if nil != e {
			if commons.IsTimeout(e) {
				break
			}
			return nil, errutils.InternalError(e.Error())
		}
		values = append(values, [2]string{addr.String(), version.String()})
	}
	return commons.Return(values), nil
}

func (self *PingerDriver) Put(params map[string]string) (map[string]interface{}, commons.RuntimeError) {
	id, ok := params["id"]
	if !ok {
		return nil, commons.IdNotExists
	}

	pinger, ok := self.pingers[id]
	if !ok {
		return nil, errutils.RecordNotFound(id)
	}

	port, ok := params["port"]
	if !ok {
		port = "161"
	}

	body, ok := params["body"]
	if !ok {
		return nil, commons.BodyNotExists
	}
	if "" == body {
		return nil, errutils.IsRequired("body")
	}
	ipList := make([]string, 0, 100)
	e := json.Unmarshal([]byte(body), &ipList)
	if nil != e {
		return nil, errutils.BadRequest("read body failed, it is not []string of json - " + e.Error() + body)
	}

	versions := []SnmpVersion{SNMP_V2C, SNMP_V3}
	version, e := getVersion(params)
	if SNMP_Verr != version {
		versions = []SnmpVersion{version}
	}

	for _, ip_raw := range ipList {
		ip_range, e := netutils.ParseIPRange(ip_raw)
		if nil != e {
			return nil, errutils.InternalError(e.Error())
		}

		for i, v := range versions {
			if i != 0 {
				time.Sleep(500 * time.Millisecond)
				ip_range.Reset()
			}

			for ip_range.HasNext() {
				e = pinger.Send(net.JoinHostPort(ip_range.Current().String(), port), v)
				if nil != e {
					return nil, errutils.InternalError(e.Error())
				}
			}
		}
	}
	return commons.ReturnOK(), nil
}

func (self *PingerDriver) Create(params map[string]string) (map[string]interface{}, commons.RuntimeError) {
	body, _ := params["body"]
	if "" == body {
		body = "{}"
	}

	params2 := make(map[string]string)
	e := json.Unmarshal([]byte(body), &params2)
	if nil != e {
		return nil, errutils.BadRequest("read body failed, it is not map[string]string of json - " + e.Error())
	}
	network, _ := params2["network"]
	if "" == network {
		network, _ = params["network"]
		if "" == network {
			return nil, errutils.IsRequired("network")
		}
	}

	address, _ := params2["address"]
	if "" == address {
		address, _ = params["address"]
	}

	id := network + "," + address
	_, ok := self.pingers[id]
	if ok {
		return nil, errutils.RecordAlreadyExists(id)
	}

	pinger, err := NewPinger(network, address, 256)
	if nil != err {
		return nil, commons.NewRuntimeError(500, err.Error())
	}
	self.pingers[id] = pinger
	return commons.ReturnWithKV(map[string]interface{}{}, "id", id), nil
}

func (self *PingerDriver) Delete(params map[string]string) (bool, commons.RuntimeError) {
	id, ok := params["id"]
	if !ok {
		return false, commons.IdNotExists
	}
	pinger, ok := self.pingers[id]
	if !ok {
		return false, errutils.RecordNotFound(id)
	}
	delete(self.pingers, id)
	pinger.Close()

	return true, nil
}
