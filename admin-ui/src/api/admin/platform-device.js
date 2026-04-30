import request from '@/utils/request'

// go-admin 本地路由前缀（已注册：/api/v1/platform-device/*）
const base = '/api/v1/platform-device'

export function listPlatformDevices(query) {
  return request({
    url: base + '/list',
    method: 'get',
    params: query
  })
}

export function getPlatformDeviceSummary() {
  return request({
    // 后端路由：GET /api/v1/platform-device/stats
    url: base + '/stats',
    method: 'get'
  })
}

export function getPlatformDeviceEnum() {
  return request({
    // 后端路由：GET /api/v1/platform-device/enums
    url: base + '/enums',
    method: 'get'
  })
}

export function getPlatformDeviceDetail(sn, deviceId) {
  const idNum = deviceId != null && deviceId !== '' ? Number(deviceId) : 0
  const hasId = Number.isFinite(idNum) && idNum > 0
  let snStr = sn != null && sn !== '' ? String(sn).trim() : ''
  // 避免误把路径段「detail」当 SN；真实 SN 为 detail 时用仅 device_id 或 query
  if (snStr === 'detail') {
    snStr = ''
  }
  const params = {}
  if (hasId) {
    params.device_id = idNum
  }
  // 有 SN 时走 /platform-device/{sn}，避免固定路径 /detail 在部分环境下被当成 /:sn=detail
  const url = snStr ? `${base}/${encodeURIComponent(snStr)}` : `${base}/detail`
  return request({
    url,
    method: 'get',
    params,
    silent: true
  })
}

export function createPlatformDevice(data) {
  return request({
    // 后端路由：POST /api/v1/platform-device
    url: base,
    method: 'post',
    data
  })
}

/** POST /api/v1/platform-device/activate-cloud — 设备端带密钥+HMAC 的激活（开放平台/脚本） */
export function activatePlatformDeviceCloud(data) {
  return request({
    url: base + '/activate-cloud',
    method: 'post',
    data
  })
}

/** POST /api/v1/platform-device/activate-cloud-admin — 后台已登录管理员一键激活（无需密钥与签名） */
export function activatePlatformDeviceCloudAdmin(data) {
  return request({
    url: base + '/activate-cloud-admin',
    method: 'post',
    data
  })
}

export function importPlatformDevices(data) {
  return request({
    url: base + '/import',
    method: 'post',
    data
  })
}

/** GET /api/v1/platform-device/import/template — xlsx 模板 */
export function downloadDeviceImportTemplate() {
  return request({
    url: base + '/import/template',
    method: 'get',
    responseType: 'blob'
  })
}

/** POST /api/v1/platform-device/import/jobs — multipart field: file */
export function createDeviceImportJob(formData) {
  return request({
    url: base + '/import/jobs',
    method: 'post',
    data: formData,
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

/** GET /api/v1/platform-device/import/jobs/:id */
export function getDeviceImportJob(id) {
  return request({
    url: `${base}/import/jobs/${id}`,
    method: 'get'
  })
}

/** GET /api/v1/platform-device/import/jobs/:id/download — CSV 含明文密钥 */
export function downloadDeviceImportJobResult(id) {
  return request({
    url: `${base}/import/jobs/${id}/download`,
    method: 'get',
    responseType: 'blob'
  })
}

export function batchPlatformDeviceStatus(data) {
  return request({
    // 后端路由：PUT /api/v1/platform-device/status
    url: base + '/status',
    method: 'put',
    data
  })
}

export function setPlatformDeviceStatus(data) {
  const sn = data && data.sn
  return request({
    // 后端路由：PUT /api/v1/platform-device/:sn/status
    url: `${base}/${encodeURIComponent(sn)}/status`,
    method: 'put',
    data
  })
}

export function unbindPlatformDevice(data) {
  const sn = data && data.sn
  return request({
    // 后端路由：POST /api/v1/platform-device/:sn/unbind
    url: `${base}/${encodeURIComponent(sn)}/unbind`,
    method: 'post',
    data
  })
}

export function sendPlatformDeviceCommand(data) {
  const sn = data && data.sn
  return request({
    // 后端路由：POST /api/v1/platform-device/:sn/command
    url: `${base}/${encodeURIComponent(sn)}/command`,
    method: 'post',
    data
  })
}

export function pushPlatformDeviceOTA(data) {
  const sn = data && data.sn
  return request({
    // 后端路由：POST /api/v1/platform-device/:sn/ota
    url: `${base}/${encodeURIComponent(sn)}/ota`,
    method: 'post',
    data
  })
}

/** GET 状态上报历史：始终 /status-logs + query（device_id、sn），避免 /:sn/status-logs 在旧后端未注册时 HTTP 404 */
export function listPlatformDeviceStatusLogs(sn, params) {
  const p = Object.assign({}, params || {})
  const idNum = p.device_id != null && p.device_id !== '' ? Number(p.device_id) : 0
  const hasId = Number.isFinite(idNum) && idNum > 0
  let snStr = sn != null && sn !== '' ? String(sn).trim() : ''
  if (snStr === 'status-logs' || snStr === 'detail') {
    snStr = ''
  }
  if (hasId) {
    p.device_id = idNum
  } else {
    delete p.device_id
  }
  if (snStr) {
    p.sn = snStr
  } else {
    delete p.sn
  }
  return request({
    url: `${base}/status-logs`,
    method: 'get',
    params: p,
    silent: true
  })
}

/** POST /api/v1/platform-device/remote-command — 通用远程指令 */
export function remotePlatformDeviceCommand(data) {
  return request({
    url: `${base}/remote-command`,
    method: 'post',
    data: data || {},
    silent: true
  })
}

/** POST 立即触发设备上报（单段 path，旧后端可仅注册 /trigger-report-status） */
export function triggerPlatformDeviceReportStatus(data) {
  return request({
    url: `${base}/trigger-report-status`,
    method: 'post',
    data: data || {},
    silent: true
  })
}

/** POST 手动填报状态（优先单段 path，与旧后端 /status-logs/manual 兼容） */
export function manualPlatformDeviceStatusReport(data) {
  return request({
    url: `${base}/manual-status-report`,
    method: 'post',
    data: data || {},
    silent: true
  })
}

/** POST /api/v1/platform-device/info/update — 管理员更新设备扩展信息 */
export function updatePlatformDeviceInfo(data) {
  return request({
    url: `${base}/info/update`,
    method: 'post',
    data: data || {},
    silent: true
  })
}

/** POST /api/v1/platform-device/info/update-batch — 批量更新 */
export function updatePlatformDeviceInfoBatch(data) {
  return request({
    url: `${base}/info/update-batch`,
    method: 'post',
    data: data || {},
    silent: true
  })
}
