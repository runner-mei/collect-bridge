package main

// #cgo LDFLAGS: -lm
// #cgo windows CFLAGS: -DLUA_COMPAT_ALL -DLUA_COMPAT_ALL -I ./include
// #cgo windows LDFLAGS: -L ./lib -llua52
// #cgo linux CFLAGS: -DLUA_USE_LINUX -DLUA_COMPAT_ALL
// #cgo linux LDFLAGS: -ldl
// #include <stdlib.h>
// #include "lua.h"
// #include "lualib.h"
// #include "lauxlib.h"
import "C"
import (
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"snmp"
	"sync"
	"time"
	"unsafe"
)

const (
	lua_init_script string = `
function receive ()
    local action, params = coroutine.yield()
    return action, params
end

function send (co, ...)
    local action, params = coroutine.yield(co, ...)
    return action, params
end

function execute_task (action, task)
  return coroutine.create(function()
    return "test ok"
    end)
end

function loop ()
  print("lua enter looping")
  local action, params = receive()  -- get new value
  while "__exit__" ~= action do
    print("lua vm receive - '%s' and '%s' \n", action, params)
    co = execute_task(action, params)
    action, params = send(co)
  end
  print("lua exit looping")
end

print("welcome to lua vm")
loop ()
`

	LUA_YIELD     int = C.LUA_YIELD
	LUA_ERRRUN        = C.LUA_ERRRUN
	LUA_ERRSYNTAX     = C.LUA_ERRSYNTAX
	LUA_ERRMEM        = C.LUA_ERRMEM
	LUA_ERRERR        = C.LUA_ERRERR
	LUA_ERRFILE       = C.LUA_ERRFILE
)

type LUA_CODE int

const (
	LUA_EXECUTE_END      LUA_CODE = 0
	LUA_EXECUTE_CONTINUE LUA_CODE = 1
	LUA_EXECUTE_FAILED   LUA_CODE = 2
)

type LuaDriver struct {
	snmp.Svc
	init_path   string
	ls          *C.lua_State
	init_script string
	waitG       sync.WaitGroup
}

type Continuous struct {
	ls     *C.lua_State
	status LUA_CODE
	drv    string
	action string

	params   map[string]string
	any      interface{}
	err      error
	intValue int
}

// func (svc *Svc) Set(onStart, onStop, onTimeout func()) {
// 	svc.onStart = onStart
// 	svc.onStop = onStop
// 	svc.onTimeout = onTimeout
// }
func NewLuaDriver(init_script string) *LuaDriver {
	driver := &LuaDriver{init_script: init_script}
	driver.Name = "lua_driver"
	driver.Set(func() { driver.atStart() }, func() { driver.atStop() }, nil)
	return driver
}

func (driver *LuaDriver) lua_init(ls *C.lua_State) C.int {
	var cs *C.char
	defer func() {
		if nil != cs {
			C.free(unsafe.Pointer(cs))
		}
	}()

	pushString(ls, runtime.GOARCH)

	cs = C.CString("__mj_arch")
	C.lua_setglobal(ls, cs)
	C.free(unsafe.Pointer(cs))
	cs = nil

	cs = C.CString("__mj_os")
	pushString(ls, runtime.GOOS)
	C.lua_setglobal(ls, cs)
	C.free(unsafe.Pointer(cs))
	cs = nil

	if len(os.Args) > 0 {
		pa := path.Base(os.Args[0])
		if fileExists(pa) {
			pushString(ls, pa)
			cs = C.CString("__mj_execute_directory")
			C.lua_setglobal(ls, cs)
			C.free(unsafe.Pointer(cs))
			cs = nil
		}
	}
	wd, err := os.Getwd()
	if nil == err {
		pushString(ls, wd)
		cs = C.CString("__mj_work_directory")
		C.lua_setglobal(ls, cs)
		C.free(unsafe.Pointer(cs))
		cs = nil

		pa := path.Join(wd, driver.init_path)
		if fileExists(pa) {
			cs = C.CString(pa)
			return C.luaL_loadfilex(ls, cs, nil)
		}
		driver.Logger.Printf("LuaDriver: '%s' is not exist.", pa)
	}

	cs = C.CString(driver.init_script)
	return C.luaL_loadstring(ls, cs)
}

func (driver *LuaDriver) executeTask(schema, action string, params map[string]string) (ret interface{}, err error) {
	drv, ok := Connect(schema)
	if !ok {
		err = fmt.Errorf("driver '%s' is not exists.", schema)
		return
	}

	switch action {
	case "get":
		ret, err = drv.Get(params)
	case "put":
		ret, err = drv.Put(params)
	case "create":
		ret, err = drv.Create(params)
	case "delete":
		ret, err = drv.Delete(params)
	default:
		err = fmt.Errorf("unsupport action '%s'", action)
	}
	return
}

func (driver *LuaDriver) eval(ls, from *C.lua_State, argc C.int) (ret C.int) {

	for {
		ret = C.lua_resume(ls, from, argc)
		if C.LUA_YIELD != ret {
			break
		}
		if 0 == C.lua_gettop(ls) {
			return
		}

		if 0 == C.lua_isstring(ls, 1) {
			return
		}

		var res interface{} = nil
		var err error = nil

		action := toString(ls, 1)
		switch action {
		case "log":
			if nil != driver.Logger {
				driver.Logger.Println(toString(ls, 3))
			}
		case "get", "put", "delete", "create":
			schema := toString(ls, 2)
			params := toParams(ls, 3)
			res, err = driver.executeTask(schema, action, params)
		default:
			err = fmt.Errorf("unsupport action '%s'", action)
		}

		pushAny(ls, res)
		pushError(ls, err)
		argc = 2
	}
	return
}

func (driver *LuaDriver) atStart() {
	ls := C.luaL_newstate()
	defer func() {
		if nil != ls {
			C.lua_close(ls)
		}
	}()
	C.luaL_openlibs(ls)

	if "" == driver.init_path {
		driver.init_path = "lua_init.lua"
	}

	if "" == driver.init_script {
		driver.init_script = lua_init_script
	}
	ret := driver.lua_init(ls)

	if LUA_ERRFILE == ret {
		driver.Logger.Panicf("'" + driver.init_path + "' read fail")
	} else if 0 != ret {
		driver.Logger.Panicf(getError(ls, ret, "load '"+driver.init_path+"' failed").Error())
	}

	ret = driver.eval(ls, nil, 0)
	if C.LUA_YIELD != ret {
		driver.Logger.Panicf(getError(ls, ret, "launch main fiber failed").Error())
	}

	driver.ls = ls
	ls = nil
	driver.Logger.Println("driver is started!")
}

func (driver *LuaDriver) atStop() {
	if nil == driver.ls {
		return
	}

	ret := C.lua_status(driver.ls)
	if C.LUA_YIELD != ret {
		driver.Logger.Panicf(getError(driver.ls, ret, "stop main fiber failed").Error())
	}

	pushString(driver.ls, "__exit__")
	ret = driver.eval(driver.ls, nil, 1)
	if 0 != ret {
		driver.Logger.Panicf(getError(driver.ls, ret, "stop main fiber failed").Error())
	}

	driver.Logger.Println("wait for all fibers to exit!")
	driver.waitG.Wait()
	driver.Logger.Println("all fibers is exited!")

	C.lua_close(driver.ls)
	driver.ls = nil
	driver.Logger.Println("driver is exited!")
}

func (driver *LuaDriver) newContinuous(action string, params map[string]string) *Continuous {
	pushString(driver.ls, action)
	pushParams(driver.ls, params)

	ret := driver.eval(driver.ls, nil, 2)
	if C.LUA_YIELD != ret {
		if 0 == ret {
			return &Continuous{status: LUA_EXECUTE_FAILED,
				err: errors.New("'lua_init.lua' is directly exited.")}
		} else {
			return &Continuous{status: LUA_EXECUTE_FAILED,
				err: getError(driver.ls, ret, "switch to main fiber failed")}
		}
	}

	if C.LUA_TTHREAD != C.lua_type(driver.ls, -1) {
		return &Continuous{status: LUA_EXECUTE_FAILED,
			err: errors.New("main fiber return value by yeild is not 'lua_State' type")}
	}

	new_th := C.lua_tothread(driver.ls, -1)
	if nil == new_th {
		return &Continuous{status: LUA_EXECUTE_FAILED,
			err: errors.New("main fiber return value by yeild is nil")}
	}

	ct := &Continuous{status: LUA_EXECUTE_FAILED, ls: new_th}
	ret = C.lua_resume(new_th, driver.ls, 0)
	ct = driver.executeContinuous(ret, ct)

	if LUA_EXECUTE_CONTINUE == ct.status {
		driver.waitG.Add(1)
	}
	return ct
}

func (driver *LuaDriver) againContinue(ct *Continuous) *Continuous {
	pushAny(ct.ls, ct.any)
	pushError(ct.ls, ct.err)

	ret := C.lua_resume(ct.ls, driver.ls, 2)
	return driver.executeContinuous(ret, ct)
}

func (driver *LuaDriver) executeContinuous(ret C.int, ct *Continuous) *Continuous {
	switch ret {
	case C.LUA_YIELD:
		ct.status = LUA_EXECUTE_CONTINUE
		ct.action = toString(ct.ls, -3)
		if "log" == ct.action {
			ct.intValue = toInteger(ct.ls, -2)
			ct.err = toError(ct.ls, -1)
		} else {
			ct.drv = toString(ct.ls, -2)
			ct.params = toParams(ct.ls, -1)
		}
	case 0:
		ct.status = LUA_EXECUTE_END
		ct.any = toAny(ct.ls, -2)
		ct.err = toError(ct.ls, -1)
		// There is no explicit function to close or to destroy a thread. Threads are
		// subject to garbage collection, like any Lua object. 
	default:
		ct.status = LUA_EXECUTE_FAILED
		ct.err = getError(ct.ls, ret, "script execute failed - ")
		// There is no explicit function to close or to destroy a thread. Threads are
		// subject to garbage collection, like any Lua object. 
	}
	return ct
}

func toContinuous(values []interface{}) (ct *Continuous, err error) {

	if 2 <= len(values) && nil != values[1] {
		err = values[1].(error)
	}

	if nil != values[0] {
		var ok bool = false
		ct, ok = values[0].(*Continuous)
		if !ok {
			err = snmp.NewTwinceError(err, fmt.Errorf("oooooooo! It is not a Continuous - %v", values[0]))
		}
	} else if nil == err {
		err = errors.New("oooooooo! return a nil")
	}
	return
}

func (driver *LuaDriver) invoke(action string, params map[string]string) (interface{}, error) {
	t := 5 * time.Minute
	old := time.Now()

	values := driver.SafelyCall(t, func() *Continuous {
		return driver.newContinuous(action, params)
	})
	ct, err := toContinuous(values)
	if nil != err {
		if nil != ct && LUA_EXECUTE_CONTINUE == ct.status {
			driver.waitG.Done()
		}
		return nil, err
	}

	if LUA_EXECUTE_CONTINUE == ct.status {
		defer func() {
			driver.waitG.Done()
		}()

		for {
			if "log" == ct.action {
				if nil != driver.Logger {
					driver.Logger.Println(ct.err.Error())
				}
				ct.any = nil
				ct.err = nil
			} else {
				ct.any, ct.err = driver.executeTask(ct.drv, ct.action, ct.params)
			}

			seconds := (time.Now().Second() - old.Second())
			t -= (time.Duration(seconds) * time.Second)
			values := driver.SafelyCall(t, func() *Continuous {
				return driver.againContinue(ct)
			})

			ct, err = toContinuous(values)
			if nil != err {
				return nil, err
			}
			if LUA_EXECUTE_CONTINUE != ct.status {
				break
			}
		}
	}
	if LUA_EXECUTE_END == ct.status {
		return ct.any, ct.err
	}
	return nil, ct.err
}

func (driver *LuaDriver) invokeAndReturnBool(action string, params map[string]string) (bool, error) {
	ret, err := driver.invoke(action, params)
	if nil == ret {
		return false, err
	}
	b, ok := ret.(bool)
	if !ok {
		panic(fmt.Sprintf("type of result is not bool type - %v", b))
	}

	return b, err
}
func (driver *LuaDriver) Get(params map[string]string) (interface{}, error) {
	return driver.invoke("get", params)
}

func (driver *LuaDriver) Put(params map[string]string) (interface{}, error) {
	return driver.invoke("put", params)
}

func (driver *LuaDriver) Create(params map[string]string) (bool, error) {
	return driver.invokeAndReturnBool("create", params)
}

func (driver *LuaDriver) Delete(params map[string]string) (bool, error) {
	return driver.invokeAndReturnBool("delete", params)
}
