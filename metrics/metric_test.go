package metrics

import (
	"commons"
	"log"
	"lua_binding"
	"testing"
	"time"
)

func TestRoutes(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)

	drv := lua_binding.NewLuaDriver(1*time.Second, nil)
	drv.InitLoggerWithCallback(func(s []byte) { t.Log(string(s)) }, "", 0)
	drv.Name = "TestRoutes"
	drv.Start()
	defer func() {
		drv.Stop()
	}()
	params := map[string]string{"schema": "metric_tests", "target": "unit_test"}
	v, e := drv.Get(params)
	if nil != e {
		t.Error(e)
		return
	}

	s, ok := commons.GetReturn(v).(string)
	if !ok {
		t.Errorf("return is not a string, %T", v)
		return
	}

	if "ok" != s {
		t.Errorf("return != 'ok', it is %s", s)
		return
	}
}
