package shadow

// LuaScriptBatchCAS 批量 CAS 更新设备影子 Hash（v1 字段：reported_json / desired_json / version 等）。
// KEYS[1..N] = device:shadow:{SN}
// 每个设备 6 个 ARGV（顺序与 KEYS 一致）：
//   expect_version, reported_json, desired_json, delta_json, metadata_json, updated_ms
// 脚本内两阶段：先校验全部 version，再一次性写入；任一步失败则整批不修改。
const LuaScriptBatchCAS = `
local n = #KEYS
local argc_per = 6
if #ARGV ~= n * argc_per then
  return {err="argv_count_mismatch"}
end

for i = 1, n do
  local key = KEYS[i]
  local base = (i - 1) * argc_per
  local expect_v = tonumber(ARGV[base + 1])
  if redis.call('EXISTS', key) == 0 then
    return {err="shadow_not_found", index=i, key=key}
  end
  local cur_v = tonumber(redis.call('HGET', key, 'version') or '0')
  if cur_v ~= expect_v then
    return {err="version_conflict", index=i, key=key, current_version=cur_v, expect_version=expect_v}
  end
end

for i = 1, n do
  local key = KEYS[i]
  local base = (i - 1) * argc_per
  local new_v = tonumber(ARGV[base + 1]) + 1
  redis.call('HSET', key,
    'version', tostring(new_v),
    'reported_json', ARGV[base + 2],
    'desired_json', ARGV[base + 3],
    'delta_json', ARGV[base + 4],
    'metadata_json', ARGV[base + 5],
    'updated_ms', ARGV[base + 6]
  )
end

return {ok=true, updated=n}
`
