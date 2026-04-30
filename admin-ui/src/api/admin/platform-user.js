import axios from 'axios'
import request from '@/utils/request'
import { getToken } from '@/utils/auth'
import { resolveBlob } from '@/utils/zipdownload'
import { resolveApiBaseURL } from '@/utils/env-api'

const base = '/api/v1/platform-user'

/** 导出 CSV（与列表相同筛选条件） */
export function exportPlatformUser(query) {
  const baseURL = resolveApiBaseURL()
  return axios({
    method: 'get',
    url: baseURL + base + '/export',
    params: query,
    responseType: 'blob',
    headers: { Authorization: 'Bearer ' + getToken() }
  }).then((res) => {
    resolveBlob(res, 'text/csv;charset=utf-8')
  })
}

/** 从 CSV 批量导入用户 */
export function importPlatformUser(file) {
  const fd = new FormData()
  fd.append('file', file)
  return request({
    url: base + '/import',
    method: 'post',
    data: fd
  })
}

export function listPlatformUser(query) {
  return request({
    url: base + '/list',
    method: 'get',
    params: query
  })
}

export function getPlatformUser(userId) {
  return request({
    url: `${base}/${userId}`,
    method: 'get'
  })
}

export function addPlatformUser(data) {
  return request({
    url: base,
    method: 'post',
    data
  })
}

export function updatePlatformUser(userId, data) {
  return request({
    url: `${base}/${userId}`,
    method: 'put',
    data
  })
}

export function uploadPlatformUserAvatar(userId, file) {
  const fd = new FormData()
  fd.append('file', file)
  return request({
    url: `${base}/${userId}/avatar`,
    method: 'post',
    data: fd
  })
}

export function updatePlatformUserStatus(userId, status) {
  const uid = Number(userId)
  const st = Number(status)
  return request({
    url: base + '/status',
    method: 'put',
    data: { userId: uid, user_id: uid, status: st }
  })
}

export function resetPlatformUserPassword(userId, password) {
  return request({
    url: `${base}/${userId}/password`,
    method: 'put',
    data: { password }
  })
}

export function setPlatformUserRoles(userId, roleIds) {
  return request({
    url: `${base}/${userId}/roles`,
    method: 'put',
    data: { role_ids: roleIds }
  })
}

export function delPlatformUser(userId) {
  return request({
    url: `${base}/${userId}`,
    method: 'delete'
  })
}
