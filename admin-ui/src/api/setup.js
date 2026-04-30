import request from '@/utils/request'

// 后台冷启动 setup 状态/注册：统一使用 /api/admin/setup/*，与 C 端 /api/v1/* 彻底分离。
// 旧 /api/v1/setup/* 已从后端下线，不再做兼容回退。
export function getSetupStatus() {
  return request({
    url: '/api/admin/setup/status',
    method: 'get',
    silent: true
  })
}

export function postSetupRegister(data) {
  return request({
    url: '/api/admin/setup/register',
    method: 'post',
    data,
    silent: true
  })
}

// 管理员账号注册（后台独立模块，仅写入 public.sys_admin，与 C 端 users 隔离）
// 与 postSetupRegister 的区别：不依赖 needsSetup=true，随时可注册新管理员。
export function postAdminAccountRegister(data) {
  return request({
    url: '/api/admin/account/register',
    method: 'post',
    data
  })
}
