module("routes",  package.seeall)

local pcall = pcall
mj.routes = {}

local route = ml.class()
function route:_init()
  self.filters = {}
  return self
end
function route:equal(key, value, ignoreCase)

  local m = "equal"
  if nil ~= ignoreCase then
    if type(ignoreCase) ~= "boolean"  then
      error("ignoreCase must is a boolean value")
    end

    if ignoreCase then
      m = "equal_with_ignore_case"
    end
  end

  table.insert(self.filters, {method= m, arguments={key, value}})
  return self
end
function route:in_with(key, ...)
  table.insert(self.filters, {method= "in", arguments={key, {...}}})
  return self
end
function route:start_with(key, prefix)
  table.insert(self.filters, {method= "start_with", arguments={key, prefix}})
  return self
end
function route:end_with(key, suffix)
  table.insert(self.filters, {method= "end_with", arguments={key, suffix}})
  return self
end
function route:contains(key, sub)
  table.insert(self.filters, {method= "contains", arguments={key, sub}})
  return self
end
function route:match(key, pat)
  table.insert(self.filters, {method= "match", arguments={key, pat}})
  return self
end
function route:and_with(f)
  error("and_with is not implemented")
  return self
end

local actions = {get= true, put=true, create=true, delete=true}
function check_table_params_of_action(opts)
  method = opts["method"]
  if nil == method or not actions[method] then
    error("please use 'get, put, create, delete' to create the action object")
  end
  schema = opts["schema"]
  if nil == schema then
    error("'schema' is required.")
  end
end

function check_action_options(opts)
  if type(opts) ~= "table" then
    error("argument must is a table[string,string]")
  end
  if nil ~= opts["method"] then
    error("'method' is remain, user can`t use it.")
  end
end

function load_routefile(file)
  local res, rt = nil, {}
  rt.__index = _G
  ml.update(rt, _G)
  rt.route= route
  rt.get = function(opts)
    check_action_options(opts)
    opts["method"] = "get"
    return opts
  end
  rt.put = function(opts)
    check_action_options(opts)
    opts["method"] = "put"
    return opts
  end
  rt.create = function(opts)
    check_action_options(opts)
    opts["method"] = "create"
    return opts
  end
  rt.delete = function(opts)
    check_action_options(opts)
    opts["method"] = "delete"
    return opts
  end



  if type(file) ~= "string" then
    error("argument 'file' must is a string")
  end
  local f, message = loadfile(file, "bt", rt)
  if nil == f then
    error(message or "load '" .. file .. "'failed")
  end
  f()

  local name = rt.name
  local method = rt.method
  local level = rt.level
  local categories = rt.categories
  local match = rt.match
  local action = rt.action

  if nil == method or "" == method then
    error("load '" .. file .."' failed, method is required")
  end
  if not actions[method] then
    error("please use 'get, put, create, delete' to set method var")
  end
  if nil == name or "" == name then
    error("load '" .. file .."' failed, name is required")
  end
  if nil == level or "" == level then
    error("load '" .. file .."' failed, level is required")
  end 
  if nil == categories or "" == categories then
    error("load '" .. file .."' failed, categories is required")
  end 
  if nil == match or "" == match then
    error("load '" .. file .."' failed, match is required")
  end
  if not route.classof(match) then
    error("load '" .. file .."' failed, match must is a route object")
  end
  rt.match = match.filters
  if nil == action or "" == action then
    error("load '" .. file .."' failed, action is required")
  end

  if type(action) == "table" then
    check_table_params_of_action(action)
  elseif type(action) == "function" then
    error("oooooops, I`m sorry, it is not implemented.")
  else
    error("argument must is a table[string,string] or function.")
  end
  

  return {name = rt.name, method= rt.method, level = rt.level, categories = rt.categories, match = rt.match, action = rt.action }
end