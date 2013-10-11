package sampling

import (
	"commons"
	"snmp"
	"testing"
	"time"
)

type MockContext struct {
	commons.StringMap
}

func (self MockContext) CreateCtx(metric_name string, managed_type, managed_id string) (MContext, error) {
	return nil, commons.NotImplemented
}

func (self MockContext) SetBodyClass(value interface{}) {
}

func (self MockContext) Read() Sampling {
	return nil
}
func (self MockContext) Body() (interface{}, error) {
	return nil, nil
}

func (self MockContext) BodyString() (string, error) {
	return "", nil
}

func TestSysOid(t *testing.T) {
	snmp, e := snmp.NewSnmpDriver(10*time.Second, nil)
	if nil != e {
		t.Error(e)
		return
	}
	defer snmp.Close()

	var sys systemOid
	e = sys.Init(map[string]interface{}{"snmp": snmp})
	if nil != e {
		t.Error(e)
		return
	}
	res := sys.Call(MockContext{commons.StringMap(map[string]string{"snmp.version": "v2c",
		"@address":            "127.0.0.1",
		"snmp.port":           "161",
		"snmp.read_community": "public"})})
	if res.HasError() {
		t.Error(res.Error())
		return
	}
	s, e := res.Value().AsString()
	if nil != e {
		t.Error(e)
		return
	}
	if 0 == len(s) {
		t.Error("valus is empty.")
	}
	//t.Error(res.InterfaceValue())
}

func TestSysOidFailed(t *testing.T) {
	snmp, e := snmp.NewSnmpDriver(10*time.Second, nil)
	if nil != e {
		t.Error(e)
		return
	}
	defer snmp.Close()

	var sys systemOid
	e = sys.Init(map[string]interface{}{"snmp": snmp})
	if nil != e {
		t.Error(e)
		return
	}
	res := sys.Call(MockContext{commons.StringMap(map[string]string{})})
	if !res.HasError() {
		t.Error("excepted error is ''snmp' is required.', actual is nil")
		return
	}
	if res.ErrorMessage() != "'snmp' is required." {
		t.Error(res.Error())
		return
	}
}
