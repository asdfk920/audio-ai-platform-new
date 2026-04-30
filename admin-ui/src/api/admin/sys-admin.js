import request from '@/utils/request'

// 管理员模块前端 API
// 后端：go-admin/admin/app/admin/apis/sys_admin.go
// 路由：/api/v1/sys-admin/*（受 JWT + Casbin 约束，仅 super_admin / admin 角色可用）
//
// 字段命名遵循后端 DTO 中的 snake_case（见 service/dto/sys_admin.go）。

const base = '/api/v1/sys-admin'

// #region agent log
function agentDebugLog(hypothesisId, message, data) {
  try {
    fetch('http://127.0.0.1:7281/ingest/098c2c1f-e0e8-4ec4-a6e0-ba69d53941cd', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Debug-Session-Id': 'a35f07' },
      body: JSON.stringify({
        sessionId: 'a35f07',
        runId: 'sys-admin-ui',
        hypothesisId,
        location: 'admin-ui/src/api/admin/sys-admin.js',
        message,
        data: data || {},
        timestamp: Date.now()
      })
    }).catch(() => {})
  } catch (e) {
    console.log('placeholder')
  }
}
// #endregion

/** 分页列表
 * query: { pageIndex, pageSize, keyword, role_id, status, last_login_from, last_login_to, sort_by, sort_order }
 */
export function listSysAdmin(query) {
  return request({
    url: base,
    method: 'get',
    params: query
  })
}

/** 详情 */
export function getSysAdmin(id) {
  return request({
    url: `${base}/${id}`,
    method: 'get'
  })
}

/** 新增（body 字段见 SysAdminCreateReq：username/password/nickname/real_name/email/phone/avatar/dept_id/role_ids/status/remark/must_change_password） */
export function addSysAdmin(data) {
  return request({
    url: base,
    method: 'post',
    data
  })
}

/** 更新（body 字段见 SysAdminUpdateReq） */
export function updateSysAdmin(id, data) {
  // H1: 前端未在 body 里带 user_id，导致后端 JSON bind 校验 required 失败
  agentDebugLog('H1', 'updateSysAdmin_request_shape', {
    id: Number(id),
    hasUserId: Boolean(data && (data.user_id || data.user_id === 0)),
    keys: data ? Object.keys(data).slice(0, 30) : [],
    roleIdsLen: data && Array.isArray(data.role_ids) ? data.role_ids.length : -1,
    hasAvatar: Boolean(data && data.avatar)
  })
  return request({
    url: `${base}/${id}`,
    method: 'put',
    data
  })
}

/** 单条删除（软删除） */
export function delSysAdmin(id) {
  return request({
    url: `${base}/${id}`,
    method: 'delete'
  })
}

/** 批量删除：body = { user_ids: number[], reason?: string } */
export function batchDelSysAdmin(userIds, reason) {
  return request({
    url: `${base}/batch-delete`,
    method: 'post',
    data: { user_ids: userIds, reason: reason || '' }
  })
}

/** 启用/禁用：status = '1' 禁用 / '2' 正常 */
export function changeSysAdminStatus(id, status) {
  return request({
    url: `${base}/${id}/status`,
    method: 'put',
    data: { user_id: Number(id), status: String(status) }
  })
}

/** 重置密码（超管重置他人密码）
 * requireChangeOnLogin 默认为 true，要求对方下次登录强制改密。
 */
export function resetSysAdminPassword(id, newPassword, requireChangeOnLogin = true) {
  return request({
    url: `${base}/${id}/password`,
    method: 'put',
    data: {
      user_id: Number(id),
      new_password: newPassword,
      require_change_on_login: !!requireChangeOnLogin
    }
  })
}

/** 设置安全策略：IP 白名单 / 登录时间窗 */
export function setSysAdminSecurity(id, { allowedIps, allowedLoginStart, allowedLoginEnd }) {
  return request({
    url: `${base}/${id}/security`,
    method: 'put',
    data: {
      user_id: Number(id),
      allowed_ips: allowedIps || '',
      allowed_login_start: allowedLoginStart || '',
      allowed_login_end: allowedLoginEnd || ''
    }
  })
}

/** 切换「下次登录强制改密」标志 */
export function setSysAdminMustChange(id, must) {
  return request({
    url: `${base}/${id}/must-change-password`,
    method: 'put',
    data: { user_id: Number(id), must: !!must }
  })
}

/** 自助改密（已登录管理员本人调用） */
export function changeSelfPassword(oldPassword, newPassword) {
  return request({
    url: '/api/admin/account/change-password',
    method: 'post',
    data: { old_password: oldPassword, new_password: newPassword }
  })
}
