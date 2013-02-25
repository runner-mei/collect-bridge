package metrics

import (
	"commons"
	"testing"
)

type dispatcher_test struct {
	dispatcherBase
}

func (self *dispatcher_test) Init(params map[string]interface{}, drvName string) commons.RuntimeError {
	e := self.dispatcherBase.Init(params, drvName)
	if nil != e {
		return e
	}

	self.RegisterGetFunc([]string{"1.3.6.1.4.1.5655"}, func(params map[string]string) (map[string]interface{}, commons.RuntimeError) {
		return commons.Return("5655"), nil
	})
	self.RegisterGetFunc([]string{"1.3.6.1.4.1.9"}, func(params map[string]string) (map[string]interface{}, commons.RuntimeError) {
		return commons.Return("9"), nil
	})
	self.RegisterGetFunc([]string{"1.3.6.1.4.1.9.1.746"}, func(params map[string]string) (map[string]interface{}, commons.RuntimeError) {
		return commons.Return("9.1.746"), nil
	})
	self.RegisterGetFunc([]string{"1.3.6.1.4.1.9.12.3.1.3"}, func(params map[string]string) (map[string]interface{}, commons.RuntimeError) {
		return nil, commons.ContinueError
	})
	self.RegisterGetFunc([]string{"1.12.3.1.3"}, func(params map[string]string) (map[string]interface{}, commons.RuntimeError) {
		return commons.Return("other"), nil
	})
	self.RegisterGetFunc([]string{"1.12.3.1"}, func(params map[string]string) (map[string]interface{}, commons.RuntimeError) {
		return nil, commons.ContinueError
	})
	self.RegisterGetFunc([]string{"1.3.6.1.4.1.9.1.965", "1.3.6.1.4.1.9.1.966", "1.3.6.1.4.1.9.1.967"},
		func(params map[string]string) (map[string]interface{}, commons.RuntimeError) {
			return commons.Return("9.1.965"), nil
		})

	self.get = func(params map[string]string) (map[string]interface{}, commons.RuntimeError) {
		return commons.Return("default"), nil
	}
	return nil
}

func TestDispatcherBase(t *testing.T) {
	test := &dispatcher_test{}
	drvMgr := commons.NewDriverManager()
	drvMgr.Register("snmp", &commons.DefaultDrv{})
	e := test.Init(map[string]interface{}{"drvMgr": drvMgr, "metrics": commons.NewDriverManager()}, "snmp")
	if nil != e {
		t.Error(e)
		return
	}

	res, e := test.Get(map[string]string{"sys.oid": "1.3.6.1.4.1.5655"})
	if "5655" != commons.GetReturn(res) {
		t.Errorf("1.3.6.1.4.1.5655 != 5655, %v, %v", commons.GetReturn(res), e)
	}
	res, e = test.Get(map[string]string{"sys.oid": "1.3.6.1.4.1.9"})
	if "9" != commons.GetReturn(res) {
		t.Errorf("1.3.6.1.4.1.9 != 9, %v, %v", commons.GetReturn(res), e)
	}
	res, e = test.Get(map[string]string{"sys.oid": "1.3.6.1.4.1.9.1.746"})
	if "9.1.746" != commons.GetReturn(res) {
		t.Errorf("1.3.6.1.4.1.9.1.746 != 9.1.746, %v, %v", commons.GetReturn(res), e)
	}
	res, e = test.Get(map[string]string{"sys.oid": "1.3.6.1.4.1.9.12.3.1.3"})
	if "9" != commons.GetReturn(res) {
		t.Errorf("1.3.6.1.4.1.9.1.746 != 9, %v, %v", commons.GetReturn(res), e)
	}
	res, e = test.Get(map[string]string{"sys.oid": "1.12.3.1.3"})
	if "other" != commons.GetReturn(res) {
		t.Errorf("1.12.3.1.3 != other, %v, %v", commons.GetReturn(res), e)
	}
	res, e = test.Get(map[string]string{"sys.oid": "1.3.6.1.4.1.9.1.965"})
	if "9.1.965" != commons.GetReturn(res) {
		t.Errorf("1.3.6.1.4.1.9.1.965 != 9.1.965, %v, %v", commons.GetReturn(res), e)
	}
	res, e = test.Get(map[string]string{"sys.oid": "1.3.6.1.4.1.9.1.966"})
	if "9.1.965" != commons.GetReturn(res) {
		t.Errorf("1.3.6.1.4.1.9.1.966 != 9.1.965, %v, %v", commons.GetReturn(res), e)
	}
	res, e = test.Get(map[string]string{"sys.oid": "1.3.6.1.4.1.9.1.967"})
	if "9.1.965" != commons.GetReturn(res) {
		t.Errorf("1.3.6.1.4.1.9.1.967 != 9.1.965, %v, %v", commons.GetReturn(res), e)
	}
}