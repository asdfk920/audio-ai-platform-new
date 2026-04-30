import request from '@/utils/request'

// 绑定管理 API 前缀
const base = '/api/v1/platform-user-device-bind'

/**
 * 获取绑定列表
 * @param {Object} query - 查询参数
 * @param {string} query.userId - 用户 ID
 * @param {string} query.deviceSn - 设备 SN
 * @param {number} query.status - 绑定状态
 * @param {number} query.page - 页码
 * @param {number} query.pageSize - 每页数量
 */
export function getBindList(query) {
  return request({
    url: base + '/list',
    method: 'get',
    params: query
  })
}

/**
 * 获取绑定详情
 * @param {number} id - 绑定 ID
 */
export function getBindDetail(id) {
  return request({
    url: base + '/' + id,
    method: 'get'
  })
}

/**
 * 新增绑定
 * @param {Object} data - 绑定数据
 * @param {number} data.userId - 用户 ID
 * @param {string} data.deviceSn - 设备 SN
 */
export function createBind(data) {
  return request({
    url: base,
    method: 'post',
    data
  })
}

/**
 * 解绑
 * @param {number} id - 绑定 ID
 */
export function unbind(id) {
  return request({
    url: base + '/' + id + '/unbind',
    method: 'post'
  })
}

/**
 * 批量解绑
 * @param {Array<number>} ids - 绑定 ID 列表
 */
export function batchUnbind(ids) {
  return request({
    url: base + '/batch-unbind',
    method: 'post',
    data: { ids }
  })
}

/**
 * 获取用户已绑定设备列表
 * @param {number} userId - 用户 ID
 */
export function getUserBindDevices(userId) {
  return request({
    url: base + '/user/devices',
    method: 'get',
    params: { userId }
  })
}

/**
 * 获取用户可绑定设备列表
 * @param {number} userId - 用户 ID
 */
export function getAvailableDevices(userId) {
  return request({
    url: base + '/available/devices',
    method: 'get',
    params: { userId }
  })
}
