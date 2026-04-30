import request from '@/utils/request'

const base = '/api/v1/platform-rbac'

export function getPlatformRbacMatrix() {
  return request({
    url: base + '/matrix',
    method: 'get'
  })
}

export function listPlatformRbacRoles() {
  return request({
    url: base + '/roles',
    method: 'get'
  })
}

export function updatePlatformRbacRole(roleKey, modules) {
  return request({
    url: `${base}/roles/${roleKey}`,
    method: 'put',
    data: { modules }
  })
}

