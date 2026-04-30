import request from '@/utils/request'

const base = '/api/v1/platform-content'

/** 获取内容列表 */
export function listContent(query) {
  return request({
    url: base + '/list',
    method: 'get',
    params: query
  })
}

/** 获取内容详情 */
export function getContentDetail(contentId) {
  return request({
    url: base + '/detail',
    method: 'get',
    params: { content_id: contentId }
  })
}

function appendSpatialFields(formData, data) {
  const keys = ['pos_x', 'pos_y', 'pos_z', 'yaw', 'pitch', 'roll', 'render_distance', 'render_gain', 'render_filter']
  for (const k of keys) {
    if (data[k] === undefined || data[k] === null || data[k] === '') continue
    formData.append(k, String(data[k]))
  }
}

/** 新增内容 */
export function addContent(data) {
  const formData = new FormData()
  formData.append('title', data.title)
  formData.append('artist', data.artist)
  formData.append('duration', String(data.duration_sec))
  formData.append('vip_level', String(data.vip_level))
  if (data.status !== undefined && data.status !== null && data.status !== '') {
    formData.append('status', String(data.status))
  }
  appendSpatialFields(formData, data)
  if (data.cover_url) formData.append('cover_url', data.cover_url)
  if (data.audio_url) formData.append('audio_url', data.audio_url)
  if (data.audio_key) formData.append('audio_key', data.audio_key)
  if (data.cover_file instanceof File) formData.append('cover', data.cover_file)
  if (data.audio_file instanceof File) formData.append('audio', data.audio_file)
  formData.append('audio_validity_mode', data.audio_validity_mode || 'none')
  formData.append('audio_valid_from', data.audio_valid_from || '')
  formData.append('audio_valid_until', data.audio_valid_until || '')
  if (data.lyrics) formData.append('lyrics', data.lyrics)
  if (data.description) formData.append('description', data.description)

  // #region agent log
  const fdKeys = []
  try {
    formData.forEach((_, k) => {
      fdKeys.push(k)
    })
  } catch (e) { /* ignore */ }
  fetch('http://127.0.0.1:7774/ingest/97f27dd3-163b-4d88-a4e7-e2b55236be17', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Debug-Session-Id': '167bb8' },
    body: JSON.stringify({
      sessionId: '167bb8',
      hypothesisId: 'H1',
      location: 'platform-content.js:addContent',
      message: 'formdata_built',
      data: { keys: fdKeys, titleLen: data.title ? String(data.title).length : 0 },
      timestamp: Date.now()
    })
  }).catch(() => {})
  // #endregion

  // 勿手写 Content-Type: multipart/form-data（缺少 boundary 会导致服务端解析不到字段）
  const req = request({
    url: base + '/add',
    method: 'post',
    data: formData
  })
  return req
    .then((res) => {
      // #region agent log
      fetch('http://127.0.0.1:7774/ingest/97f27dd3-163b-4d88-a4e7-e2b55236be17', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Debug-Session-Id': '167bb8' },
        body: JSON.stringify({
          sessionId: '167bb8',
          hypothesisId: 'H2',
          location: 'platform-content.js:addContent:then',
          message: 'add_response_ok',
          data: { code: res && res.code },
          timestamp: Date.now()
        })
      }).catch(() => {})
      // #endregion
      return res
    })
    .catch((err) => {
      // #region agent log
      fetch('http://127.0.0.1:7774/ingest/97f27dd3-163b-4d88-a4e7-e2b55236be17', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Debug-Session-Id': '167bb8' },
        body: JSON.stringify({
          sessionId: '167bb8',
          hypothesisId: 'H3',
          location: 'platform-content.js:addContent:catch',
          message: 'add_request_failed',
          data: {
            code: err && err.code,
            msg: err && err.msg,
            httpStatus: err && err.response && err.response.status,
            axiosMsg: err && err.message
          },
          timestamp: Date.now()
        })
      }).catch(() => {})
      // #endregion
      return Promise.reject(err)
    })
}

/** 更新内容（后端仅解析 application/x-www-form-urlencoded / multipart，不用 JSON） */
export function updateContent(contentId, data) {
  const params = new URLSearchParams()
  params.append('content_id', String(contentId))
  const set = (k, v) => {
    if (v === undefined || v === null || v === '') return
    params.append(k, String(v))
  }
  set('title', data.title)
  set('artist', data.artist)
  set('vip_level', data.vip_level)
  set('status', data.status)
  set('cover_url', data.cover_url)
  set('audio_url', data.audio_url)
  if (data.duration_sec != null && data.duration_sec !== '') {
    params.append('duration', String(data.duration_sec))
  }
  const spatialKeys = ['pos_x', 'pos_y', 'pos_z', 'yaw', 'pitch', 'roll', 'render_distance', 'render_gain', 'render_filter']
  for (const k of spatialKeys) {
    if (data[k] === undefined || data[k] === null || data[k] === '') continue
    params.append(k, String(data[k]))
  }
  params.append('audio_validity_mode', data.audio_validity_mode || 'none')
  params.append('audio_valid_from', data.audio_valid_from || '')
  params.append('audio_valid_until', data.audio_valid_until || '')
  return request({
    url: base + '/update',
    method: 'post',
    data: params.toString(),
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}

/** 上架内容 */
export function onlineContent(contentId) {
  return request({
    url: base + '/online',
    method: 'post',
    data: { content_id: contentId }
  })
}

/** 下架内容 */
export function offlineContent(contentId) {
  return request({
    url: base + '/offline',
    method: 'post',
    data: { content_id: contentId }
  })
}

/** 上传文件（通用接口，所有文件转为私有格式） */
export function uploadFile(file) {
  const formData = new FormData()
  formData.append('file', file)

  return request({
    url: '/api/admin/file/upload',
    method: 'post',
    data: formData,
    timeout: 300000
  })
}

/** 删除内容 */
export function deleteContent(contentId) {
  return request({
    url: base + '/delete',
    method: 'post',
    data: { content_id: contentId }
  })
}
