import request from '@/utils/request'

// 获取验证码（失败时静默：登录页不弹全局「查询失败」，开发环境可继续填占位码）
export function getCodeImg() {
  return request({
    url: '/api/v1/captcha',
    method: 'get',
    silent: true
  })
}

// 查询 此接口不在验证数据权限（失败时静默：登录页已有默认 sysInfo，避免全局拦截器弹「查询失败」）
export function getSetting() {
  return request({
    url: '/api/v1/app-config',
    method: 'get',
    silent: true
  })
}
