import request from '@/utils/request'

const base = '/api/v1/platform-device'

/** GET /api/v1/platform-device/shadow/list — 设备影子分页列表 */
export function listDeviceShadows(query) {
  const req = (path) =>
    request({
      url: base + path,
      method: 'get',
      params: query
    })
  return req('/shadow/list').catch((err) => {
    const status = err && err.response && err.response.status
    if (status === 404) {
      return req('/shadow-list')
    }
    return Promise.reject(err)
  })
}

/** GET /api/v1/platform-device/shadow?sn= — 影子详情（reported/desired/delta） */
export function getDeviceShadow(sn) {
  return request({
    url: base + '/shadow',
    method: 'get',
    params: { sn },
    silent: true
  })
}
