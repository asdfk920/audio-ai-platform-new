import axios from 'axios'
import { MessageBox, Message } from 'element-ui'
import store from '@/store'
import { getToken } from '@/utils/auth'
import { resolveApiBaseURL } from '@/utils/env-api'

// 登录页 bootstrap 接口：部分 axios 版本会丢自定义字段 silent，按 URL 兜底静默，避免误弹「查询失败」
function isSilentRequest(config) {
  if (!config) return false
  if (config.silent === true) return true
  const raw = String(config.baseURL || '') + String(config.url || '')
  return (
    raw.includes('/api/v1/app-config') ||
    raw.includes('/api/v1/captcha') ||
    raw.includes('/api/admin/setup/')
  )
}

function requestUrl(config) {
  if (!config) return ''
  return String(config.baseURL || '') + String(config.url || '')
}

// create an axios instance
const service = axios.create({
  // 开发环境通常为空，走 9527 同源 + devServer 代理到 go-admin:8000
  baseURL: resolveApiBaseURL(),
  // withCredentials: true, // send cookies when cross-domain requests
  // 部分接口（如首次加载菜单/权限、导入导出等）在 Windows + Docker 环境可能超过 10s；放宽超时避免频繁误报
  timeout: 30000 // request timeout
})

// request interceptor
service.interceptors.request.use(
  config => {
    // do something before request is sent
    if (config.data instanceof FormData) {
      const h = config.headers
      if (h) {
        if (typeof h.delete === 'function') {
          h.delete('Content-Type')
          h.delete('content-type')
        } else {
          delete h['Content-Type']
          delete h['content-type']
        }
      }
    } else {
      const hasCT = config.headers['Content-Type'] || config.headers['content-type']
      if (!hasCT) {
        config.headers['Content-Type'] = 'application/json'
      }
    }

    if (store.getters.token) {
      // let each request carry token
      // ['X-Token'] is a custom headers key
      // please modify it according to the actual situation
      config.headers['Authorization'] = 'Bearer ' + getToken()
    }
    return config
  },
  error => {
    // do something with request error
    console.log(error) // for debug
    return Promise.reject(error)
  }
)

// response interceptor
service.interceptors.response.use(
  /**
   * If you want to get http information such as headers or status
   * Please return  response => response
  */

  /**
   * Determine the request status by custom code
   * Here is just an example
   * You can also judge the status by HTTP Status Code
   */
  response => {
    const silent = isSilentRequest(response.config)
    const data = response.data || {}
    const code = data.code
    // 兼容后端未包裹 {code,msg,data} 的旧/非标准响应：HTTP 200 直接放行
    if (code === undefined && response && response.status === 200) {
      return response.data
    }
    if (code === 401) {
      store.dispatch('user/resetToken')
      // 登录成功后立刻 router.push 时，地址栏可能仍是 /login；此时守卫里 getInfo 若返回 401，
      // 下面按「在登录页」整页 reload 会打断 SPA 跳转（表现为登录成功却不进首页 / redirect）。
      const isGetInfo = /getinfo/i.test(requestUrl(response.config))
      if (!isGetInfo) {
        if (location.href.indexOf('login') !== -1) {
          location.reload() // 为了重新实例化vue-router对象 避免bug
        } else {
          MessageBox.confirm(
            '登录状态已过期，您可以继续留在该页面，或者重新登录',
            '系统提示',
            {
              confirmButtonText: '重新登录',
              cancelButtonText: '取消',
              type: 'warning'
            }
          ).then(() => {
            location.reload() // 为了重新实例化vue-router对象 避免bug
          })
        }
      }
      return Promise.reject(data)
    } else if (code === 6401) {
      store.dispatch('user/resetToken')
      MessageBox.confirm(
        '登录状态已过期，您可以继续留在该页面，或者重新登录',
        '系统提示',
        {
          confirmButtonText: '重新登录',
          cancelButtonText: '取消',
          type: 'warning'
        }
      ).then(() => {
        location.reload() // 为了重新实例化vue-router对象 避免bug
      })
      return Promise.reject(data)
    } else if (code === 400 || code === 403) {
      if (!silent) {
        Message({
          message: data.msg || data.message || '请求失败',
          type: 'error',
          duration: 5 * 1000
        })
      }
      return Promise.reject(data)
    } else if (code !== 200) {
      // Notification.error({
      //   title: response.data.msg
      // })
      if (!silent) {
        Message({
          message: data.msg || data.message || '请求失败',
          type: 'error'
        })
      }
      return Promise.reject(data)
    } else {
      return response.data
    }
  },
  error => {
    const silent = isSilentRequest(error.config)
    if (error.message === 'Network Error') {
      if (!silent) {
        Message({
          message: '服务器连接异常，请检查服务器！',
          type: 'error',
          duration: 5 * 1000
        })
      }
      return Promise.reject(error)
    }
    console.log('err' + error) // for debug

    if (!silent) {
      const ed = error.response && error.response.data
      const srvMsg = ed && (ed.msg || ed.message)
      Message({
        message: srvMsg || error.message,
        type: 'error',
        duration: 5 * 1000
      })
    }

    return Promise.reject(error)
  }
)

export default service
