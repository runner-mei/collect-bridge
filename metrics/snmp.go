package metrics

import (
	"commons"
	"errors"
	"fmt"
	"strconv"
)

type result_type int

const (
	RES_STRING result_type = iota
	RES_OID
	RES_INT32
	RES_INT64
	RES_UINT32
	RES_UINT64
)

var (
	metricNotExistsError = commons.IsRequired("metric")
	snmpNotExistsError   = commons.IsRequired("parameter of snmp")
)

type snmpBase struct {
	drv commons.Driver
}

func (self *snmpBase) Init(params map[string]interface{}) error {
	v := params["snmp"]
	if nil != v {
		if drv, ok := v.(commons.Driver); ok {
			self.drv = drv
			return nil
		}
	}

	v = params["drv_manager"]
	if nil == v {
		return commons.IsRequired("snmp' or 'drv_manager'")
	}

	drvMgr, ok := v.(commons.DriverManager)
	if !ok {
		return errors.New("'drv_manager' is not a driver manager.")
	}

	drv, _ := drvMgr.Connect("snmp")
	if nil == v {
		return errors.New("'snmp' is not exists in the driver manager")
	}
	self.drv = drv
	return nil
}

func (self *snmpBase) copyParameter(params commons.Map, snmp_params map[string]string, rw string) error {
	version := params.GetStringWithDefault("snmp.version", "")
	if 0 == len(version) {
		return snmpNotExistsError
	}

	snmp_params["snmp.version"] = version

	address := params.GetStringWithDefault("@address", "")
	if 0 == len(address) {
		return commons.IsRequired("@address")
	}
	snmp_params["snmp.address"] = address
	snmp_params["snmp.port"] = params.GetStringWithDefault("snmp.port", "")

	switch version {
	case "v3", "V3", "3":
		snmp_params["snmp.secmodel"] = params.GetStringWithDefault("snmp.sec_model", "")
		snmp_params["snmp.auth_pass"] = params.GetStringWithDefault("snmp."+rw+"_auth_pass", "")
		snmp_params["snmp.priv_pass"] = params.GetStringWithDefault("snmp."+rw+"_priv_pass", "")
		snmp_params["snmp.max_msg_size"] = params.GetStringWithDefault("snmp.max_msg_size", "")
		snmp_params["snmp.context_name"] = params.GetStringWithDefault("snmp.context_name", "")
		snmp_params["snmp.identifier"] = params.GetStringWithDefault("snmp.identifier", "")
		snmp_params["snmp.engine_id"] = params.GetStringWithDefault("snmp.engine_id", "")
		break
	default:
		community := params.GetStringWithDefault("snmp."+rw+"_community", "")
		if 0 == len(community) {
			return commons.IsRequired("snmp." + rw + "_community")
		}

		snmp_params["snmp.community"] = community
	}
	return nil
}

func (self *snmpBase) Get(params commons.Map, oid string) (map[string]interface{}, error) {
	snmp_params := make(map[string]string)
	snmp_params["snmp.oid"] = oid
	snmp_params["snmp.action"] = "get"
	e := self.copyParameter(params, snmp_params, "read")
	if nil != e {
		return nil, e
	}
	res := self.drv.Get(snmp_params)
	if res.HasError() {
		return nil, res.Error()
	}

	rv := res.InterfaceValue()
	if nil == rv {
		return nil, commons.ValueIsNil
	}
	values, ok := rv.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("snmp result is not a map[string]interface{}, actual is [%T]%v.", rv, rv)
	}
	return values, nil
}

func (self *snmpBase) GetResult(params commons.Map, oid string, rt result_type) commons.Result {
	values, e := self.Get(params, oid)
	if nil != e {
		return commons.ReturnWithInternalError(e.Error())
	}

	switch rt {
	case RES_STRING:
		s, e := TryGetString(params, values, oid)
		if nil != e {
			return commons.ReturnWithInternalError(e.Error())
		}
		return commons.Return(s)
	case RES_OID:
		s, e := TryGetOid(params, values, oid)
		if nil != e {
			return commons.ReturnWithInternalError(e.Error())
		}
		return commons.Return(s)
	case RES_INT32:
		i32, e := TryGetInt32(params, values, oid, 0)
		if nil != e {
			return commons.ReturnWithInternalError(e.Error())
		}
		return commons.Return(i32)
	case RES_INT64:
		i64, e := TryGetInt64(params, values, oid, 0)
		if nil != e {
			return commons.ReturnWithInternalError(e.Error())
		}
		return commons.Return(i64)
	case RES_UINT32:
		u32, e := TryGetUint32(params, values, oid, 0)
		if nil != e {
			return commons.ReturnWithInternalError(e.Error())
		}
		return commons.Return(u32)
	case RES_UINT64:
		u64, e := TryGetInt64(params, values, oid, 0)
		if nil != e {
			return commons.ReturnWithInternalError(e.Error())
		}
		return commons.Return(u64)
	default:
		return commons.ReturnWithInternalError("unsupported type of snmp result - " + strconv.Itoa(int(rt)))
	}
}

func (self *snmpBase) GetString(params commons.Map, oid string) (string, error) {
	values, e := self.Get(params, oid)
	if nil != e {
		return "", e
	}
	return TryGetString(params, values, oid)
}

func (self *snmpBase) GetOid(params commons.Map, oid string) (string, error) {
	values, e := self.Get(params, oid)
	if nil != e {
		return "", e
	}
	return TryGetOid(params, values, oid)
}

func (self *snmpBase) GetInt32(params commons.Map, oid string) (int32, error) {
	values, e := self.Get(params, oid)
	if nil != e {
		return 0, e
	}
	return TryGetInt32(params, values, oid, 0)
}

func (self *snmpBase) GetInt64(params commons.Map, oid string) (int64, error) {
	values, e := self.Get(params, oid)
	if nil != e {
		return 0, e
	}
	return TryGetInt64(params, values, oid, 0)
}

func (self *snmpBase) GetUint32(params commons.Map, oid string) (uint32, error) {
	values, e := self.Get(params, oid)
	if nil != e {
		return 0, e
	}
	return TryGetUint32(params, values, oid, 0)
}

func (self *snmpBase) GetUint64(params commons.Map, oid string) (uint64, error) {
	values, e := self.Get(params, oid)
	if nil != e {
		return 0, e
	}
	return TryGetUint64(params, values, oid, 0)
}

func (self *snmpBase) GetTable(params commons.Map, oid, columns string,
	cb func(key string, row map[string]interface{}) error) (e error) {
	defer func() {
		if o := recover(); nil != o {
			e = errors.New(fmt.Sprint(o))
		}
	}()

	snmp_params := map[string]string{"snmp.oid": oid,
		"snmp.action":  "table",
		"snmp.columns": columns}

	e = self.copyParameter(params, snmp_params, "read")
	if nil != e {
		return e
	}

	res := self.drv.Get(snmp_params)
	if res.HasError() {
		return res.Error()
	}

	rv := res.InterfaceValue()
	if nil == rv {
		return commons.ValueIsNil
	}
	values, ok := rv.(map[string]interface{})
	if !ok {
		return fmt.Errorf("snmp result must is not a map[string]interface{} - [%T]%v.", rv, rv)
	}

	for key, r := range values {
		row, ok := r.(map[string]interface{})
		if !ok {
			return fmt.Errorf("row with key is '%s' process failed, it is not a map[string]interface{} - [%T]%v.", key, r, r)
		}

		e := cb(key, row)
		if nil != e {
			if commons.InterruptError == e {
				break
			}

			return errors.New("row with key is '" + key + "' process failed, " + e.Error())
		}
	}
	return nil
}

func (self *snmpBase) OneInTable(params commons.Map, oid, columns string,
	cb func(key string, row map[string]interface{}) error) error {
	return self.GetTable(params, oid, columns, func(key string, row map[string]interface{}) error {
		e := cb(key, row)
		if nil == e {
			return commons.InterruptError
		}

		if commons.ContinueError == e {
			return nil
		}

		return e
	})
}

func (self *snmpBase) EachInTable(params commons.Map, oid, columns string,
	cb func(key string, row map[string]interface{}) error) error {
	return self.GetTable(params, oid, columns, cb)
}

func (self *snmpBase) GetOneResult(params commons.Map, oid, columns string,
	cb func(key string, row map[string]interface{}) (map[string]interface{}, error)) commons.Result {
	var err error
	var result map[string]interface{} = nil
	err = self.GetTable(params, oid, columns, func(key string, row map[string]interface{}) error {
		var e error
		result, e = cb(key, row)
		if nil == e {
			return commons.InterruptError
		}

		if commons.ContinueError == e {
			return nil
		}

		return e
	})
	if nil != err {
		return commons.ReturnWithInternalError(err.Error())
	}
	return commons.Return(result)
}

func (self *snmpBase) GetAllResult(params commons.Map, oid, columns string,
	cb func(key string, row map[string]interface{}) (map[string]interface{}, error)) commons.Result {
	var err error
	var results []map[string]interface{} = nil
	err = self.GetTable(params, oid, columns, func(key string, row map[string]interface{}) error {
		result, e := cb(key, row)
		if nil != e {
			return e
		}
		results = append(results, result)
		return nil
	})
	if nil != err {
		return commons.ReturnWithInternalError(err.Error())
	}
	return commons.Return(results)
}

type systemOid struct {
	snmpBase
}

func (self *systemOid) Call(params commons.Map) commons.Result {
	return self.GetResult(params, "1.3.6.1.2.1.1.2.0", RES_OID)
}

type systemDescr struct {
	snmpBase
}

func (self *systemDescr) Call(params commons.Map) commons.Result {
	return self.GetResult(params, "1.3.6.1.2.1.1.1.0", RES_STRING)
}

type systemName struct {
	snmpBase
}

func (self *systemName) Call(params commons.Map) commons.Result {
	return self.GetResult(params, "1.3.6.1.2.1.1.5.0", RES_STRING)
}

type systemUpTime struct {
	snmpBase
}

func (self *systemUpTime) Call(params commons.Map) commons.Result {
	return self.GetResult(params, "1.3.6.1.2.1.1.3.0", RES_UINT64)
}

type systemLocation struct {
	snmpBase
}

func (self *systemLocation) Call(params commons.Map) commons.Result {
	return self.GetResult(params, "1.3.6.1.2.1.1.6.0", RES_STRING)
}

type systemServices struct {
	snmpBase
}

func (self *systemServices) Call(params commons.Map) commons.Result {
	return self.GetResult(params, "1.3.6.1.2.1.1.7.0", RES_INT64)
}

type systemInfo struct {
	snmpBase
}

func (self *systemInfo) Call(params commons.Map) commons.Result {
	return self.GetOneResult(params, "1.3.6.1.2.1.1", "",
		func(key string, old_row map[string]interface{}) (map[string]interface{}, error) {
			oid := GetOid(params, old_row, "2")
			services := GetUint32(params, old_row, "7", 0)

			new_row := map[string]interface{}{}
			new_row["descr"] = GetString(params, old_row, "1")
			new_row["oid"] = oid
			new_row["upTime"] = GetUint32(params, old_row, "3", 0)
			new_row["contact"] = GetString(params, old_row, "4")
			new_row["name"] = GetString(params, old_row, "5")
			new_row["location"] = GetString(params, old_row, "6")
			new_row["services"] = services

			params.Set("sys.oid", oid)
			params.Set("sys.services", strconv.Itoa(int(services)))
			new_row["type"] = params.GetUintWithDefault("!sys.type", 0)
			return new_row, nil
		})
}

type interfaceAll struct {
	snmpBase
}

func (self *interfaceAll) Call(params commons.Map) commons.Result {
	return self.GetAllResult(params, "1.3.6.1.2.1.2.2.1", "1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21",
		func(key string, old_row map[string]interface{}) (map[string]interface{}, error) {
			new_row := map[string]interface{}{}
			new_row["ifIndex"] = GetInt32(params, old_row, "1", -1)
			new_row["ifDescr"] = GetString(params, old_row, "2")
			new_row["ifType"] = GetInt32(params, old_row, "3", -1)
			new_row["ifMtu"] = GetInt32(params, old_row, "4", -1)
			new_row["ifSpeed"] = GetUint64(params, old_row, "5", 0)
			new_row["ifPhysAddress"] = GetHardwareAddress(params, old_row, "6")
			new_row["ifAdminStatus"] = GetInt32(params, old_row, "7", -1)
			new_row["ifOpStatus"] = GetInt32(params, old_row, "8", -1)
			new_row["ifLastChange"] = GetInt32(params, old_row, "9", -1)
			new_row["ifInOctets"] = GetUint64(params, old_row, "10", 0)
			new_row["ifInUcastPkts"] = GetUint64(params, old_row, "11", 0)
			new_row["ifInNUcastPkts"] = GetUint64(params, old_row, "12", 0)
			new_row["ifInDiscards"] = GetUint64(params, old_row, "13", 0)
			new_row["ifInErrors"] = GetUint64(params, old_row, "14", 0)
			new_row["ifInUnknownProtos"] = GetUint64(params, old_row, "15", 0)
			new_row["ifOutOctets"] = GetUint64(params, old_row, "16", 0)
			new_row["ifOutUcastPkts"] = GetUint64(params, old_row, "17", 0)
			new_row["ifOutNUcastPkts"] = GetUint64(params, old_row, "18", 0)
			new_row["ifOutDiscards"] = GetUint64(params, old_row, "19", 0)
			new_row["ifOutErrors"] = GetUint64(params, old_row, "20", 0)
			new_row["ifOutQLen"] = GetUint64(params, old_row, "21", 0)
			return new_row, nil
		})
}

type interfaceDescr struct {
	snmpBase
}

func (self *interfaceDescr) Call(params commons.Map) commons.Result {
	return self.GetAllResult(params, "1.3.6.1.2.1.2.2.1", "1,2,3,4,5,6",
		func(key string, old_row map[string]interface{}) (map[string]interface{}, error) {
			new_row := map[string]interface{}{}
			new_row["ifIndex"] = GetInt32(params, old_row, "1", -1)
			new_row["ifDescr"] = GetString(params, old_row, "2")
			new_row["ifType"] = GetInt32(params, old_row, "3", -1)
			new_row["ifMtu"] = GetInt32(params, old_row, "4", -1)
			new_row["ifSpeed"] = GetUint64(params, old_row, "5", 0)
			new_row["ifPhysAddress"] = GetHardwareAddress(params, old_row, "6")
			return new_row, nil
		})
}

type systemType struct {
	snmpBase
	device2id map[string]int
}

func ErrorIsRestric(msg string, restric bool, log *commons.Logger) error {
	if !restric {
		log.DEBUG.Print(msg)
		return nil
	}
	return errors.New(msg)
}

// func (self *systemType) Init(params map[string]interface{}, drvName string) error {
// 	e := self.snmpBase.Init(params, drvName)
// 	if nil != e {
// 		return e
// 	}
// 	log, ok := params["log"].(*commons.Logger)
// 	if !ok {
// 		log = commons.Log
// 	}

// 	restric := false
// 	v, ok := params["restric"]
// 	if ok {
// 		restric = commons.AsBoolWithDefaultValue(v, restric)
// 	}

// 	dt := commons.SearchFile("etc/device_types.json")
// 	if "" == dt {
// 		return ErrorIsRestric("'etc/device_types.json' is not exists.", restric, log)
// 	}

// 	f, err := ioutil.ReadFile(dt)
// 	if nil != err {
// 		return ErrorIsRestric(fmt.Sprintf("read file '%s' failed, %s", dt, err.Error()), restric, log)
// 	}

// 	self.device2id = make(map[string]int)
// 	err = json.Unmarshal(f, &self.device2id)
// 	if nil != err {
// 		return ErrorIsRestric(fmt.Sprintf("unmarshal json '%s' failed, %s", dt, err.Error()), restric, log)
// 	}

// 	return nil
// }

func (self *systemType) Call(params commons.Map) commons.Result {
	if nil != self.device2id {
		oid := params.GetStringWithDefault("!sys.oid", "")
		if 0 != len(oid) {
			if dt, ok := self.device2id[oid]; ok {
				return commons.Return(dt)
			}
		}
	}

	t := 0
	dt, e := self.GetInt32(params, "1.3.6.1.2.1.4.1.0")
	if nil != e {
		goto SERVICES
	}

	if 1 == dt {
		t += 4
	}
	dt, e = self.GetInt32(params, "1.3.6.1.2.1.17.1.2.0")
	if nil != e {
		goto SERVICES
	}
	if dt > 0 {
		t += 2
	}

	if 0 != t {
		return commons.Return(t >> 1)
	}
SERVICES:
	services, e := params.GetUint32("sys.services")
	if nil != e {
		return commons.ReturnWithInternalError(e.Error())
	}
	return commons.Return((services & 0x7) >> 1)
}

func init() {

	Methods["sys_oid"] = newRouteSpec("sys.oid", "the oid of system", nil,
		func(rs *RouteSpec, params map[string]interface{}) (Method, error) {
			drv := &systemOid{}
			return drv, drv.Init(params)
		})

	Methods["sys_descr"] = newRouteSpec("sys.descr", "the oid of system", nil,
		func(rs *RouteSpec, params map[string]interface{}) (Method, error) {
			drv := &systemDescr{}
			return drv, drv.Init(params)
		})

	Methods["sys_name"] = newRouteSpec("sys.name", "the name of system", nil,
		func(rs *RouteSpec, params map[string]interface{}) (Method, error) {
			drv := &systemName{}
			return drv, drv.Init(params)
		})

	Methods["sys_services"] = newRouteSpec("sys.services", "the name of system", nil,
		func(rs *RouteSpec, params map[string]interface{}) (Method, error) {
			drv := &systemServices{}
			return drv, drv.Init(params)
		})

	Methods["sys_upTime"] = newRouteSpec("sys.upTime", "the upTime of system", nil,
		func(rs *RouteSpec, params map[string]interface{}) (Method, error) {
			drv := &systemUpTime{}
			return drv, drv.Init(params)
		})

	Methods["sys_type"] = newRouteSpec("sys.type", "the type of system", nil,
		func(rs *RouteSpec, params map[string]interface{}) (Method, error) {
			drv := &systemType{}
			return drv, drv.Init(params)
		})

	Methods["sys_location"] = newRouteSpec("sys.location", "the location of system", nil,
		func(rs *RouteSpec, params map[string]interface{}) (Method, error) {
			drv := &systemLocation{}
			return drv, drv.Init(params)
		})

	Methods["sys"] = newRouteSpec("sys", "the system info", nil,
		func(rs *RouteSpec, params map[string]interface{}) (Method, error) {
			drv := &systemInfo{}
			return drv, drv.Init(params)
		})

	Methods["interface"] = newRouteSpec("interface", "the interface info", nil,
		func(rs *RouteSpec, params map[string]interface{}) (Method, error) {
			drv := &interfaceAll{}
			return drv, drv.Init(params)
		})

	Methods["interfaceDescr"] = newRouteSpec("interfaceDescr", "the descr part of interface info", nil,
		func(rs *RouteSpec, params map[string]interface{}) (Method, error) {
			drv := &interfaceDescr{}
			return drv, drv.Init(params)
		})
}

// func splitsystemOid(oid string) (uint, string) {
// 	if !strings.HasPrefix(oid, "1.3.6.1.4.1.") {
// 		return 0, oid
// 	}
// 	oid = oid[12:]
// 	idx := strings.IndexRune(oid, '.')
// 	if -1 == idx {
// 		u, e := strconv.ParseUint(oid, 10, 0)
// 		if nil != e {
// 			panic(e.Error())
// 		}
// 		return uint(u), ""
// 	}

// 	u, e := strconv.ParseUint(oid[:idx], 10, 0)
// 	if nil != e {
// 		panic(e.Error())
// 	}
// 	return uint(u), oid[idx+1:]
// }

// // func (self *dispatcherBase) RegisterGetFunc(oids []string, get DispatchFunc) {
// // 	for _, oid := range oids {
// // 		main, sub := splitsystemOid(oid)
// // 		methods := self.get_methods[main]
// // 		if nil == methods {
// // 			methods = map[string]DispatchFunc{}
// // 			self.get_methods[main] = methods
// // 		}
// // 		methods[sub] = get
// // 	}
// // }

// func findFunc(oid string, funcs map[uint]map[string]DispatchFunc) DispatchFunc {
// 	main, sub := splitsystemOid(oid)
// 	methods := funcs[main]
// 	if nil == methods {
// 		return nil
// 	}
// 	get := methods[sub]
// 	if nil != get {
// 		return get
// 	}
// 	if "" == sub {
// 		return nil
// 	}
// 	return methods[""]
// }

// func findDefaultFunc(oid string, funcs map[uint]map[string]DispatchFunc) DispatchFunc {
// 	main, sub := splitsystemOid(oid)
// 	methods := funcs[main]
// 	if nil == methods {
// 		return nil
// 	}
// 	if "" == sub {
// 		return nil
// 	}
// 	return methods[""]
// }

// func (self *dispatcherBase) invoke(params commons.Map, funcs map[uint]map[string]DispatchFunc) commons.Result {
// 	oid, e := self.GetMetricAsString(params, "sys.oid")
// 	if nil != e {
// 		return commons.ReturnError(e.Code(), "get system oid failed, "+e.Error())
// 	}
// 	f := findFunc(oid, funcs)
// 	if nil != f {
// 		res := f(params)
// 		if !res.HasError() {
// 			return res
// 		}
// 		if commons.ContinueCode != res.ErrorCode() {
// 			return res
// 		}

// 		f = findDefaultFunc(oid, funcs)
// 		if nil != f {
// 			res := f(params)
// 			if !res.HasError() {
// 				return res
// 			}
// 			if commons.ContinueCode != res.ErrorCode() {
// 				return res
// 			}
// 		}
// 	}
// 	if nil != self.get {
// 		return self.get(params)
// 	}
// 	return commons.ReturnError(commons.NotAcceptableCode, "Unsupported device - "+oid)
// }