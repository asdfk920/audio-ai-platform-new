import request from '@/utils/request'

const base = '/api/v1/user-realname'

function normalizeListQuery(query) {
  if (!query) return {}
  const q = { ...query }
  if (q.pageSize != null && q.page_size == null) {
    q.page_size = q.pageSize
    delete q.pageSize
  }
  return q
}

/** 实名列表（与后端 GET /user-realname/list 一致） */
export function listPendingRealname(query) {
  return request({
    url: base + '/list',
    method: 'get',
    params: normalizeListQuery(query)
  })
}

export function getRealnameDetail(userId) {
  return request({
    url: `${base}/detail/${userId}`,
    method: 'get'
  })
}

/** 单条审核：body 为 user_id、audit_result(1 通过 2 驳回)、audit_remark */
export function reviewRealname(data) {
  const user_id = data.user_id
  const audit_result = data.action === 'approve' ? 1 : 2
  let audit_remark = ''
  if (audit_result === 2) {
    audit_remark = (data.reject_reason || '').trim()
  } else {
    audit_remark = (data.comment || '').trim() || '审核通过'
  }
  return request({
    url: base + '/audit',
    method: 'post',
    data: { user_id, audit_result, audit_remark }
  })
}

/** 批量审核：对 items 中每条 user_id 调用审核接口 */
export function batchReviewRealname(data) {
  const audit_result = data.action === 'approve' ? 1 : 2
  const audit_remark = audit_result === 2
    ? String(data.reject_reason || '').trim()
    : String(data.comment || '').trim() || '批量通过'
  const items = data.items || []
  return Promise.all(
    items.map((it) =>
      request({
        url: base + '/audit',
        method: 'post',
        data: {
          user_id: it.user_id,
          audit_result,
          audit_remark
        }
      })
    )
  )
}
