import request from '@/utils/request'

export function getDashboardData() {
  return request({
    url: '/api/v1/dashboard',
    method: 'get'
  })
}
