import request from '@/utils/request'

const base = '/api/v1/platform-device'

export function listIotProducts(query) {
  return request({
    url: base + '/products',
    method: 'get',
    params: query
  })
}

export function getIotProduct(id) {
  return request({
    url: `${base}/products/${id}`,
    method: 'get'
  })
}

export function createIotProduct(data) {
  return request({
    url: base + '/products',
    method: 'post',
    data
  })
}

export function updateIotProduct(id, data) {
  return request({
    url: `${base}/products/${id}`,
    method: 'put',
    data
  })
}

export function publishIotProduct(id) {
  return request({
    url: `${base}/products/${id}/publish`,
    method: 'post'
  })
}

export function disableIotProduct(id) {
  return request({
    url: `${base}/products/${id}/disable`,
    method: 'post'
  })
}

/** 固件列表（沿用平台设备接口） */
export function listFirmware(params) {
  return request({
    url: base + '/firmware/list',
    method: 'get',
    params
  })
}

/** multipart: product_key, version, file, description, ... */
export function uploadFirmware(formData) {
  return request({
    url: base + '/firmware/upload',
    method: 'post',
    data: formData,
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 120000
  })
}
