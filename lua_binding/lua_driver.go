package lua_binding

// #cgo windows CFLAGS: -DLUA_COMPAT_ALL -DLUA_COMPAT_ALL -I ./include
// #cgo windows LDFLAGS: -L ./lib -llua52 -lm
// #cgo linux CFLAGS: -DLUA_USE_LINUX -DLUA_COMPAT_ALL
// #cgo linux LDFLAGS: -L. -llua52 -ldl  -lm
// #include <stdlib.h>
// #include "lua.h"
// #include "lualib.h"
// #include "lauxlib.h"
import "C"
import (
	"commons"
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
local mj = {}

mj.DEBUG = 9000
mj.INFO = 6000
mj.WARN = 4000
mj.ERROR = 2000
mj.FATAL = 1000
mj.SYSTEM = 0

function mj.receive ()
    local action, params = coroutine.yield()
    return action, params
end

function mj.send_and_recv ( ...)
    local action, params = coroutine.yield( ...)
    return action, params
end

function mj.log(level, msg)
  if "number" ~= type(level) then
    return nil, "'params' is not a table."
  end

  coroutine.yield("log", level, msg)
end

function mj.execute(schema, action, params)
  if "table" ~= type(params) then
    return nil, "'params' is not a table."
  end
  return coroutine.yield(action, schema, params)
end

function mj.execute_module(module_name, action, params)
  module = require(module_name)
  if nil == module then
    return nil, "module '"..module_name.."' is not exists."
  end
  func = module[action]
  if nil == func then
    return nil, "method '"..action.."' is not implemented in module '"..module_name.."'."
  end

  return func(params)
end

function mj.execute_script(action, script, params)
  if 'string' ~= type(script) then
    return nil, "'script' is not a string."
  end
  local env = {["mj"] = mj,
   ["action"] = action,
   ['params'] = params}
  setmetatable(env, _ENV)
  func = assert(load(script, nil, 'bt', env))
  return func()
end

function mj.execute_task(action, params)
  --if nil == task then
  --  print("params = nil")
  --end

  return coroutine.create(function()
      if nil == params then
        return nil, "'params' is nil."
      end
      if "table" ~= type(params) then
        return nil, "'params' is not a table, actual is '"..type(params).."'." 
      end
      schema = params["schema"]
      if nil == schema then
        return nil, "'schema' is nil"
      elseif "script" == schema then
        return mj.execute_script(action, params["script"], params)
      else
        return mj.execute_module(schema, action, params)
      end
    end)
end


function mj.loop()

  mj.os = __mj_os or "unknown"  -- 386, amd64, or arm.
  mj.arch = __mj_arch or "unknown" -- darwin, freebsd, linux or windows
  mj.execute_directory = __mj_execute_directory or "."
  mj.work_directory = __mj_work_directory or "."

  local ext = ".so"
  local sep = "/"
  if mj.os == "windows" then
    ext = ".dll"
    sep = "\\"
  end

  if nil ~= __mj_execute_directory then
    package.path = package.path .. ";" .. mj.execute_directory .. sep .. "modules"..sep.."?.lua" ..
       ";" .. mj.execute_directory .. sep .. "modules" .. sep .. "?" .. sep .. "init.lua"

    package.cpath = package.cpath .. ";" .. mj.execute_directory .. sep .."modules" .. sep .. "?" .. ext ..
        ";" .. mj.execute_directory .. sep .. "modules" .. sep .. "?" .. sep .. "loadall" .. ext
  end

  if nil ~= __mj_work_directory then
    package.path = package.path .. ";" .. mj.work_directory .. sep .. "modules" .. sep .. "?.lua" ..
       ";" .. mj.work_directory .. sep .. "modules" .. sep .. "?" .. sep .. "init.lua"

    package.cpath = package.cpath .. ";" .. mj.work_directory .. sep .. "modules" .. sep .. "?" .. ext ..
        ";" .. mj.work_directory .. sep .. "modules" .. sep .. "?" .. sep .. "loadall" .. ext
  end


  mj.log(SYSTEM, "lua enter looping")
  local action, params = mj.receive()  -- get new value
  while "__exit__" ~= action do
    mj.log(SYSTEM, "lua vm receive - '"..action.."'")

    co = mj.execute_task(action, params)
    action, params = mj.send_and_recv(co)
  end
  mj.log(SYSTEM, "lua exit looping")
end

_G["mj"] = mj
package.loaded["mj"] = mj
package.preload["mj"] = mj
mj.log(SYSTEM, "welcome to lua vm")
mj.loop ()
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
	LUA_EXECUTE_YIELD    LUA_CODE = 2
	LUA_EXECUTE_FAILED   LUA_CODE = 3
)

type NativeMethod struct {
	Name     string
	Read     func(drv *LuaDriver, ctx *Continuous)
	Write    func(drv *LuaDriver, ctx *Continuous) (int, error)
	Callback func(drv *LuaDriver, ctx *Continuous)
}

var (
	method_init_lua = &NativeMethod{
		Name:     "method_init_lua",
		Read:     nil,
		Write:    nil,
		Callback: nil}
	method_exit_lua = &NativeMethod{
		Name: "method_exit_lua",
		Read: nil,
		Write: func(drv *LuaDriver, ctx *Continuous) (int, error) {
			err := ctx.PushStringParam("__exit__")
			if nil != err {
				return -1, err
			}
			return 1, err
		},
		Callback: nil}

	method_missing = &NativeMethod{
		Name:     "method_missing",
		Read:     nil,
		Write:    writeCallResult,
		Callback: nil}
)

type LuaDriver struct {
	snmp.Svc
	init_path string
	LS        *C.lua_State
	waitG     sync.WaitGroup

	methods        map[string]*NativeMethod
	method_missing *NativeMethod
}

type Continuous struct {
	LS     *C.lua_State
	status LUA_CODE
	method *NativeMethod

	unshift func(drv *LuaDriver, ctx *Continuous)
	push    func(drv *LuaDriver, ctx *Continuous) (int, error)

	Error       error
	IntValue    int
	StringValue string
	Params      map[string]string
	Any         interface{}
}

func (self *Continuous) clear() {
	self.method = nil
	self.Error = nil
	self.IntValue = 0
	self.StringValue = ""
	self.Params = nil
	self.Any = nil
}

func (self *Continuous) ToErrorParam(idx int) error {
	return toError(self.LS, C.int(idx))
}

func (self *Continuous) ToAnyParam(idx int) (interface{}, error) {
	return toAny(self.LS, C.int(idx)), nil
}

func (self *Continuous) ToParamsParam(idx int) (map[string]string, error) {
	return toParams(self.LS, C.int(idx)), nil
}

func (self *Continuous) ToStringParam(idx int) (string, error) {
	return toString(self.LS, C.int(idx)), nil
}

func (self *Continuous) ToIntParam(idx int) (int, error) {
	return toInteger(self.LS, C.int(idx)), nil
}

func (self *Continuous) PushAnyParam(any interface{}) error {
	pushAny(self.LS, any)
	return nil
}

func (self *Continuous) PushParamsParam(params map[string]string) error {
	pushParams(self.LS, params)
	return nil
}

func (self *Continuous) PushStringParam(s string) error {
	pushString(self.LS, s)
	return nil
}

func (self *Continuous) PushErrorParam(e error) error {
	pushError(self.LS, e)
	return nil
}

func readCallArguments(drv *LuaDriver, ctx *Continuous) {
	ctx.StringValue, ctx.Error = ctx.ToStringParam(2)
	if nil != ctx.Error {
		return
	}
	ctx.Params, ctx.Error = ctx.ToParamsParam(3)
}

func writeCallResult(drv *LuaDriver, ctx *Continuous) (int, error) {
	err := ctx.PushAnyParam(ctx.Any)
	if nil != err {
		return -1, err
	}
	err = ctx.PushErrorParam(ctx.Error)
	if nil != err {
		return -1, err
	}
	return 2, nil
}

func readActionResult(drv *LuaDriver, ctx *Continuous) {
	ctx.Any, ctx.Error = ctx.ToAnyParam(2)
	if nil != ctx.Error {
		return
	}
	ctx.Error = ctx.ToErrorParam(3)
}

func writeActionArguments(drv *LuaDriver, ctx *Continuous) (int, error) {

	err := ctx.PushStringParam(ctx.StringValue)
	if nil != err {
		return -1, err
	}
	err = ctx.PushParamsParam(ctx.Params)
	if nil != err {
		return -1, err
	}
	return 2, nil
}

// func (svc *Svc) Set(onStart, onStop, onTimeout func()) {
//	svc.onStart = onStart
//	svc.onStop = onStop
//	svc.onTimeout = onTimeout
// }
func NewLuaDriver() *LuaDriver {
	driver := &LuaDriver{}
	driver.Name = "lua_driver"
	driver.methods = make(map[string]*NativeMethod)
	driver.Set(func() { driver.atStart() }, func() { driver.atStop() }, nil)
	driver.CallbackWith(&NativeMethod{
		Name:  "get",
		Read:  readCallArguments,
		Write: writeCallResult,
		Callback: func(lua *LuaDriver, ctx *Continuous) {
			drv, ok := commons.Connect(ctx.StringValue)
			if !ok {
				ctx.Error = fmt.Errorf("driver '%s' is not exists.", ctx.StringValue)
				return
			}

			ctx.Any, ctx.Error = drv.Get(ctx.Params)
		}}, &NativeMethod{
		Name:  "put",
		Read:  readCallArguments,
		Write: writeCallResult,
		Callback: func(lua *LuaDriver, ctx *Continuous) {
			drv, ok := commons.Connect(ctx.StringValue)
			if !ok {
				ctx.Error = fmt.Errorf("driver '%s' is not exists.", ctx.StringValue)
				return
			}

			ctx.Any, ctx.Error = drv.Put(ctx.Params)
		}}, &NativeMethod{
		Name:  "create",
		Read:  readCallArguments,
		Write: writeCallResult,
		Callback: func(lua *LuaDriver, ctx *Continuous) {
			drv, ok := commons.Connect(ctx.StringValue)
			if !ok {
				ctx.Error = fmt.Errorf("driver '%s' is not exists.", ctx.StringValue)
				return
			}

			ctx.Any, ctx.Error = drv.Create(ctx.Params)
		}}, &NativeMethod{
		Name:  "delete",
		Read:  readCallArguments,
		Write: writeCallResult,
		Callback: func(lua *LuaDriver, ctx *Continuous) {
			drv, ok := commons.Connect(ctx.StringValue)
			if !ok {
				ctx.Error = fmt.Errorf("driver '%s' is not exists.", ctx.StringValue)
				return
			}

			ctx.Any, ctx.Error = drv.Delete(ctx.Params)
		}}, &NativeMethod{
		Name: "log",
		Read: func(drv *LuaDriver, ctx *Continuous) {
			ctx.IntValue, _ = ctx.ToIntParam(2)
			ctx.StringValue, _ = ctx.ToStringParam(3)
		},
		Write: func(drv *LuaDriver, ctx *Continuous) (int, error) {
			return 0, nil
		},
		Callback: func(drv *LuaDriver, ctx *Continuous) {
			drv.Logger.Println(ctx.StringValue)
		}})
	return driver
}

func (self *LuaDriver) CallbackWith(methods ...*NativeMethod) error {
	for _, m := range methods {
		if nil == m {
			return nil
		}
		if "" == m.Name {
			return errors.New("'name' is empty.")
		}
		if nil == m.Callback {
			return errors.New("'callback' of '" + m.Name + "' is nil.")
		}
		if _, ok := self.methods[m.Name]; ok {
			return errors.New("'" + m.Name + "' is already exists.")
		}
		self.methods[m.Name] = m
	}
	return nil
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

	if nil != cs {
		C.free(unsafe.Pointer(cs))
	}
	cs = C.CString(lua_init_script)
	return C.luaL_loadstring(ls, cs)
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
		driver.init_path = "core.lua"
	}

	ret := driver.lua_init(ls)

	if LUA_ERRFILE == ret {
		driver.Logger.Panicf("'" + driver.init_path + "' read fail")
	} else if 0 != ret {
		driver.Logger.Panicf(getError(ls, ret, "load '"+driver.init_path+"' failed").Error())
	}

	ctx := &Continuous{LS: ls, method: method_init_lua}

	ctx = driver.eval(ctx)
	for LUA_EXECUTE_CONTINUE == ctx.status {
		if nil != ctx.method && nil != ctx.method.Callback {
			ctx.method.Callback(driver, ctx)
		}
		ctx = driver.eval(ctx)
	}

	if LUA_EXECUTE_YIELD != ctx.status {
		driver.Logger.Panicf("launch main fiber failed, " + ctx.Error.Error())
	}

	driver.LS = ls
	ls = nil
	driver.Logger.Println("driver is started!")
}

func (driver *LuaDriver) atStop() {
	if nil == driver.LS {
		return
	}

	ret := C.lua_status(driver.LS)
	if C.LUA_YIELD != ret {
		driver.Logger.Panicf(getError(driver.LS, ret, "stop main fiber failed, status is error").Error())
	}

	ctx := &Continuous{LS: driver.LS, method: method_exit_lua}

	ctx = driver.eval(ctx)
	for LUA_EXECUTE_CONTINUE == ctx.status {
		if nil != ctx.method && nil != ctx.method.Callback {
			ctx.method.Callback(driver, ctx)
		}
		ctx = driver.eval(ctx)
	}

	if LUA_EXECUTE_END != ctx.status {
		driver.Logger.Panicf("stop main fiber failed," + ctx.Error.Error())
	}

	driver.Logger.Println("wait for all fibers to exit!")
	driver.waitG.Wait()
	driver.Logger.Println("all fibers is exited!")

	C.lua_close(driver.LS)
	driver.LS = nil
	driver.Logger.Println("driver is exited!")
}

func (self *LuaDriver) eval(ctx *Continuous) *Continuous {
	var ret C.int = 0
	var argc int = -1
	var ok bool = false
	var from *C.lua_State = nil

	if nil == ctx.LS {
		panic("aaaaa")
	}
	ls := ctx.LS
	if ls != self.LS {
		from = self.LS
	}

	for {
		if nil != ctx.method && nil != ctx.method.Write {
			argc, ctx.Error = ctx.method.Write(self, ctx)
			if nil != ctx.Error {
				ctx.status = LUA_EXECUTE_FAILED
				ctx.IntValue = C.LUA_ERRERR
				ctx.Error = errors.New("push arguments failed - " + ctx.Error.Error())
				return ctx
			}
		} else {
			argc = 0
		}
		self.Logger.Println(ctx.method.Name, ls, from, argc)
		ret = C.lua_resume(ls, from, C.int(argc))

		ctx.clear()

		switch ret {
		case 0:
			ctx.status = LUA_EXECUTE_END
			if nil != ctx.unshift {
				ctx.unshift(self, ctx)
			}
			// There is no explicit function to close or to destroy a thread. Threads are
			// subject to garbage collection, like any Lua object. 
			return ctx
		case C.LUA_YIELD:
			if 0 == C.lua_gettop(ls) {
				ctx.status = LUA_EXECUTE_YIELD
				ctx.IntValue = int(ret)
				ctx.Error = errors.New("script execute failed - return arguments is empty.")
				return ctx
			}

			if 0 == C.lua_isstring(ls, 1) {
				ctx.status = LUA_EXECUTE_YIELD
				ctx.IntValue = int(ret)
				ctx.Error = errors.New("script execute failed - return first argument is not string.")
				return ctx
			}

			action := toString(ls, 1)
			ctx.method, ok = self.methods[action]
			if !ok {
				ctx.Error = fmt.Errorf("unsupport action '%s'", action)
				ctx.Any = nil
				ctx.method = method_missing
			}

			if nil != ctx.method.Read {
				ctx.method.Read(self, ctx)
			}

			if nil != ctx.method.Callback {
				ctx.status = LUA_EXECUTE_CONTINUE
				return ctx
			}
		default:
			ctx.status = LUA_EXECUTE_FAILED
			ctx.IntValue = int(ret)
			ctx.Error = getError(ls, ret, "script execute failed")
			// There is no explicit function to close or to destroy a thread. Threads are
			// subject to garbage collection, like any Lua object. 
			return ctx
		}
	}
	return ctx
}

func toContinuous(values []interface{}) (ctx *Continuous, err error) {

	if 2 <= len(values) && nil != values[1] {
		err = values[1].(error)
	}

	if nil != values[0] {
		var ok bool = false
		ctx, ok = values[0].(*Continuous)
		if !ok {
			if nil != err {
				err = snmp.NewTwinceError(err, fmt.Errorf("oooooooo! It is not a Continuous - %v", values[0]))
			} else {
				err = fmt.Errorf("oooooooo! It is not a Continuous - %v", values[0])
			}
		}
	} else if nil == err {
		err = errors.New("oooooooo! return a nil")
	}
	return
}

func (self *LuaDriver) newContinuous(action string, params map[string]string) *Continuous {
	if nil == self.LS {
		return &Continuous{status: LUA_EXECUTE_FAILED,
			Error: errors.New("lua status is nil.")}
	}

	method := &NativeMethod{
		Name:  "get",
		Write: writeActionArguments}

	ctx := &Continuous{
		LS:          self.LS,
		status:      LUA_EXECUTE_END,
		StringValue: action,
		Params:      params,
		push:        writeActionArguments,
		unshift:     readActionResult,
		method:      method}

	ctx = self.eval(ctx)
	if LUA_EXECUTE_YIELD != ctx.status {
		if LUA_EXECUTE_END == ctx.status {
			ctx.status = LUA_EXECUTE_FAILED
			ctx.Error = errors.New("'core.lua' is directly exited.")
			return ctx
		} else {
			ctx.status = LUA_EXECUTE_FAILED
			if nil == ctx.Error {
				ctx.Error = errors.New("switch to main fiber failed.")
			} else {
				ctx.Error = errors.New("switch to main fiber failed, " + ctx.Error.Error())
			}
			return ctx
		}
	}

	if nil == self.LS { // check for muti-thread
		ctx.status = LUA_EXECUTE_FAILED
		ctx.Error = errors.New("lua status is nil, exited?")
		return ctx
	}

	if C.LUA_TTHREAD != C.lua_type(self.LS, -1) {
		ctx.status = LUA_EXECUTE_FAILED
		ctx.Error = errors.New("main fiber return value by yeild is not 'lua_State' type")
		return ctx
	}

	new_th := C.lua_tothread(self.LS, -1)
	if nil == new_th {
		ctx.status = LUA_EXECUTE_FAILED
		ctx.Error = errors.New("main fiber return value by yeild is nil")
		return ctx
	}

	ctx.LS = new_th
	ctx.method = method
	ctx.method.Write = nil
	ctx = self.eval(ctx)
	if LUA_EXECUTE_CONTINUE == ctx.status {
		self.waitG.Add(1)
	}
	return ctx
}

func (driver *LuaDriver) invoke(action string, params map[string]string) (interface{}, error) {
	t := 5 * time.Minute
	old := time.Now()

	values := driver.SafelyCall(t, func() *Continuous {
		return driver.newContinuous(action, params)
	})
	ctx, err := toContinuous(values)
	if nil != err {
		if nil != ctx && LUA_EXECUTE_CONTINUE == ctx.status {
			driver.waitG.Done()
		}
		return nil, err
	}

	if LUA_EXECUTE_CONTINUE == ctx.status {
		defer func() {
			driver.waitG.Done()
		}()

		for LUA_EXECUTE_CONTINUE == ctx.status {
			if nil != ctx.method && nil != ctx.method.Callback {
				ctx.method.Callback(driver, ctx)
			}

			seconds := (time.Now().Second() - old.Second())
			t -= (time.Duration(seconds) * time.Second)
			values := driver.SafelyCall(t, func() *Continuous {
				return driver.eval(ctx)
			})

			ctx, err = toContinuous(values)
			if nil != err {
				return nil, err
			}
		}
	}

	if LUA_EXECUTE_END == ctx.status {
		return ctx.Any, ctx.Error
	}
	return nil, ctx.Error
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
