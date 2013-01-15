package lua_binding

const (
	lua_init_script string = `
local mj = {}

mj.DEBUG = 9000
mj.INFO = 6000
mj.WARN = 4000
mj.ERROR = 2000
mj.FATAL = 1000
mj.SYSTEM = 0

mj.os = __mj_os or "unknown"  -- 386, amd64, or arm.
mj.arch = __mj_arch or "unknown" -- darwin, freebsd, linux or windows
mj.execute_directory = __mj_execute_directory or "."
mj.work_directory = __mj_work_directory or "."

mj.path_separator = "/"
mj.execute_ext = ".so"

if mj.os == "windows" then
  mj.path_separator  = "\\"
  mj.execute_ext = ".dll"
end

local join_path_with_sep = function(pa, sep, ...)
    for i = 1,select('#',...) do
        pa = pa .. sep .. select(i,...)
    end
    return pa
end

local ml_paths = { join_path_with_sep(mj.work_directory, mj.path_separator, 'microlight', 'ml.lua'),
    join_path_with_sep(mj.work_directory, mj.path_separator, '..', 'lua_binding', 'microlight', 'ml.lua'),
    join_path_with_sep(mj.execute_directory, mj.path_separator, 'microlight', 'ml.lua'),
    join_path_with_sep(mj.execute_directory, mj.path_separator, '..', 'lua_binding', 'microlight', 'ml.lua') }

local buffer = { '"microlight" load failed' }
local ok, ml = nil, nil
for i, pa in ipairs(ml_paths) do
  ok, ml = pcall(dofile, pa)
  if ok and nil ~= ml then
    break
  end

  if nil ~= ml then
    table.insert(buffer, "load '".. pa .. "' failed -- " .. ml)
  else
    table.insert(buffer, "load '".. pa .. "' failed")
  end
end

if (not ok) or nil == ml then
  error(table.concat(buffer, "\n"))
end

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

function mj.invoke_native(action, ...)
  return coroutine.yield(action, ...)
end

function mj.clean_path(pa)
  return mj.invoke_native("io_ext.clean", pa)
end

function mj.join_path(pa,...)
  return mj.clean_path(join_path_with_sep(pa, mj.path_separator, ...))
end

function mj.enumerate_files(pa)
  return mj.invoke_native("io_ext.enumerate_files", pa)
end


function mj.enumerate_scripts(pat)
  local modules_files = mj.enumerate_files(mj.join_path(mj.execute_directory, "modules"))
  if(mj.work_directory ~= mj.execute_directory) then
    local files = mj.enumerate_files(mj.join_path(mj.work_directory, "modules"))
    modules_files = ml.extend(modules_files, files)
  end
  if nil == pat then
    return modules_files
  end

  return ml.ifilter(modules_files, function(v)
    return nil ~= string.match(v, pat)
  end)
end

function mj.file_exists(pa)
  return mj.invoke_native("io_ext.file_exists", pa)
end

function mj.directory_exists(pa)
  return mj.invoke_native("io_ext.directory_exists", pa)
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
  mj.log(mj.SYSTEM, "lua enter looping")
  local action, params = mj.receive()  -- get new value
  while "__exit__" ~= action do
    mj.log(mj.SYSTEM, "lua vm receive - '"..action.."'")

    co = mj.execute_task(action, params)
    action, params = mj.send_and_recv(co)
  end
  mj.log(mj.SYSTEM, "lua exit looping")
end

_G["mj"] = mj
package.loaded["mj"] = mj
package.preload["mj"] = mj

_G["ml"] = ml
package.loaded["ml"] = ml
package.preload["ml"] = ml

mj.log(mj.SYSTEM, "welcome to lua vm on " .. mj.os .. "(" .. mj.arch .. ")")
mj.log(mj.SYSTEM, "work_directory is '" .. mj.work_directory .. "'")
mj.log(mj.SYSTEM, "execute_directory is '" .. mj.execute_directory .. "'")
mj.log(mj.SYSTEM, "load init script file")

do 
  local sep = mj.path_separator
  local ext = mj.execute_ext
  local paths_sep = ":"

  if mj.os == "windows" then
    paths_sep = ";"
  end

  if nil ~= mj.execute_directory then
    package.path = package.path .. paths_sep .. mj.execute_directory .. sep .. "modules"..sep.."?.lua" ..
        paths_sep .. mj.execute_directory .. sep .. "modules" .. sep .. "?" .. sep .. "init.lua"

    package.cpath = package.cpath .. paths_sep .. mj.execute_directory .. sep .."modules" .. sep .. "?" .. ext ..
        paths_sep .. mj.execute_directory .. sep .. "modules" .. sep .. "?" .. sep .. "loadall" .. ext
  end

  if nil ~= mj.work_directory then
    package.path = package.path .. paths_sep .. mj.work_directory .. sep .. "modules" .. sep .. "?.lua" ..
        paths_sep .. mj.work_directory .. sep .. "modules" .. sep .. "?" .. sep .. "init.lua"

    package.cpath = package.cpath .. paths_sep .. mj.work_directory .. sep .. "modules" .. sep .. "?" .. ext ..
        paths_sep .. mj.work_directory .. sep .. "modules" .. sep .. "?" .. sep .. "loadall" .. ext
  end
end

for i, x in ipairs(mj.enumerate_scripts(".*_init%.lua$")) do
    mj.log(mj.SYSTEM, "load '" .. x .. "'")
    dofile(x)
end

mj.log(mj.SYSTEM, "==================================")

mj.loop ()`
)