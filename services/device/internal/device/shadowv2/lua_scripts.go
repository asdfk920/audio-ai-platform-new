package shadowv2

const (
	LuaScriptInit = `
local key = KEYS[1]
local reported = cjson.decode(ARGV[1] or '{}')
local desired = cjson.decode(ARGV[2] or '{}')
local version = tonumber(ARGV[3] or 1)
local update_time = tonumber(ARGV[4])
local status = ARGV[5] or 'online'

if redis.call('EXISTS', key) == 1 then
    return {err="Shadow already exists"}
end

redis.call('HSET', key,
    'reported', cjson.encode(reported),
    'desired', cjson.encode(desired),
    'version', version,
    'update_time', update_time,
    'status', status
)

return {ok=true, version=version}
`

	LuaScriptUpdateReported = `
local key = KEYS[1]
local new_reported = cjson.decode(ARGV[1] or '{}')
local update_time = tonumber(ARGV[2])

local exists = redis.call('EXISTS', key)
if exists == 0 then
    return {err="Shadow not found"}
end

local current_reported_json = redis.call('HGET', key, 'reported') or '{}'
local current_reported = cjson.decode(current_reported_json)

for k, v in pairs(new_reported) do
    current_reported[k] = v
end

local current_version = tonumber(redis.call('HGET', key, 'version') or 0)
local new_version = current_version + 1
local current_status = redis.call('HGET', key, 'status') or 'offline'

redis.call('HSET', key,
    'reported', cjson.encode(current_reported),
    'version', new_version,
    'update_time', update_time
)

return {
    ok=true,
    version=new_version,
    reported=cjson.encode(current_reported),
    status=current_status
}
`

	LuaScriptUpdateDesired = `
local key = KEYS[1]
local new_desired = cjson.decode(ARGV[1] or '{}')
local update_time = tonumber(ARGV[2])

local exists = redis.call('EXISTS', key)
if exists == 0 then
    return {err="Shadow not found"}
end

local current_desired_json = redis.call('HGET', key, 'desired') or '{}'
local current_desired = cjson.decode(current_desired_json)

for k, v in pairs(new_desired) do
    current_desired[k] = v
end

local current_version = tonumber(redis.call('HGET', key, 'version') or 0)
local new_version = current_version + 1
local current_status = redis.call('HGET', key, 'status') or 'offline'

redis.call('HSET', key,
    'desired', cjson.encode(current_desired),
    'version', new_version,
    'update_time', update_time
)

return {
    ok=true,
    version=new_version,
    desired=cjson.encode(current_desired),
    status=current_status
}
`

	LuaScriptCASUpdate = `
local key = KEYS[1]
local expect_version = tonumber(ARGV[1])
local new_reported_str = ARGV[2] or '{}'
local new_desired_str = ARGV[3] or '{}'
local new_status = ARGV[4]
local update_time = tonumber(ARGV[5])

local exists = redis.call('EXISTS', key)
if exists == 0 then
    return {err="Shadow not found", conflict=false}
end

local current_version = tonumber(redis.call('HGET', key, 'version') or 0)
if current_version ~= expect_version then
    return {
        err="Version conflict",
        conflict=true,
        current_version=current_version,
        expect_version=expect_version
    }
end

local updates = {}
updates['version'] = current_version + 1
updates['update_time'] = update_time

if new_reported_str and new_reported_str ~= '{}' and new_reported_str ~= '' then
    local new_reported = cjson.decode(new_reported_str)
    local current_reported_json = redis.call('HGET', key, 'reported') or '{}'
    local current_reported = cjson.decode(current_reported_json)
    
    for k, v in pairs(new_reported) do
        current_reported[k] = v
    end
    
    updates['reported'] = cjson.encode(current_reported)
end

if new_desired_str and new_desired_str ~= '{}' and new_desired_str ~= '' then
    local new_desired = cjson.decode(new_desired_str)
    local current_desired_json = redis.call('HGET', key, 'desired') or '{}'
    local current_desired = cjson.decode(current_desired_json)
    
    for k, v in pairs(new_desired) do
        current_desired[k] = v
    end
    
    updates['desired'] = cjson.encode(current_desired)
end

if new_status and new_status ~= '' then
    updates['status'] = new_status
end

redis.call('HSET', key, unpack(updates))

local result = {
    ok=true,
    version=updates['version'],
    conflict=false
}

if updates['reported'] then
    result['reported'] = updates['reported']
end
if updates['desired'] then
    result['desired'] = updates['desired']
end
if updates['status'] then
    result['status'] = updates['status']
end

return result
`

	LuaScriptUpdateStatus = `
local key = KEYS[1]
local new_status = ARGV[1]
local update_time = tonumber(ARGV[2])

local exists = redis.call('EXISTS', key)
if exists == 0 then
    return {err="Shadow not found"}
end

local current_version = tonumber(redis.call('HGET', key, 'version') or 0)
local new_version = current_version + 1

redis.call('HSET', key,
    'status', new_status,
    'update_time', update_time,
    'version', new_version
)

return {
    ok=true,
    version=new_version,
    status=new_status
}
`
)
