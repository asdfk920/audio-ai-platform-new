import request from '@/utils/request'

const base = '/api/v1/platform-member'

export function listMemberLevels() {
  return request({ url: base + '/levels', method: 'get' })
}

export function upsertMemberLevel(data) {
  const hasId = data && data.id
  return request({ url: base + '/levels', method: hasId ? 'put' : 'post', data })
}

export function deleteMemberLevel(id) {
  return request({ url: `${base}/levels/${id}`, method: 'delete' })
}

/** action: set_status，需传 status: 0 | 1 */
export function batchMemberLevels(data) {
  return request({ url: `${base}/levels/batch`, method: 'post', data })
}

/** ids 为全表等级 id 按目标顺序排列 */
export function reorderMemberLevels(data) {
  return request({ url: `${base}/levels/reorder`, method: 'post', data })
}

export function listMemberBenefits() {
  return request({ url: base + '/benefits', method: 'get' })
}

export function upsertMemberBenefit(data) {
  const hasId = data && data.id
  return request({ url: base + '/benefits', method: hasId ? 'put' : 'post', data })
}

export function deleteMemberBenefit(id) {
  return request({ url: `${base}/benefits/${id}`, method: 'delete' })
}

/** 批量设置生命周期或类型：action = set_lifecycle | set_benefit_type */
export function batchMemberBenefits(data) {
  return request({ url: `${base}/benefits/batch`, method: 'post', data })
}

export function getLevelBenefits(levelCode) {
  return request({ url: base + '/level-benefits', method: 'get', params: { level_code: levelCode }})
}

export function setLevelBenefits(levelCode, benefitCodes) {
  return request({ url: `${base}/level-benefits/${encodeURIComponent(levelCode)}`, method: 'put', data: { benefit_codes: benefitCodes || [] }})
}

export function getUserMember(userId) {
  return request({ url: base + '/user-member', method: 'get', params: { user_id: userId }})
}

export function upsertUserMember(data) {
  return request({ url: base + '/user-member', method: 'put', data })
}

export function listUserMembers(params) {
  return request({ url: `${base}/user-members`, method: 'get', params })
}

export function userMemberSummary() {
  return request({ url: `${base}/user-members/summary`, method: 'get' })
}

export function getUserMemberDetail(userId) {
  return request({ url: `${base}/user-member/detail`, method: 'get', params: { user_id: userId }})
}

/** action: set_level | set_status | renew_days | downgrade */
export function batchUserMembers(data) {
  return request({ url: `${base}/user-member/batch`, method: 'post', data })
}

