import { getSetting } from '@/api/login'
import storage from '@/utils/storage'
const state = {
  info: storage.get('app_info')
}

const mutations = {
  SET_INFO: (state, data) => {
    state.info = data
    storage.set('app_info', data)
  }
}

const defaultAppInfo = {
  sys_app_name: 'go-admin',
  // 使用本地 favicon，避免外链证书过期导致控制台 ERR_CERT_DATE_INVALID
  sys_app_logo: '/favicon.ico'
}

/** 数据库 sys_config 里可能仍为旧外链 logo，覆盖为本地资源 */
function sanitizeAppInfo(data) {
  if (!data || typeof data !== 'object') return data
  const out = { ...data }
  const logo = String(out.sys_app_logo || '')
  if (logo.includes('doc-image.zhangwj.com') || logo.includes('zhangwj.com/img/')) {
    out.sys_app_logo = '/favicon.ico'
  }
  return out
}

const actions = {
  settingDetail({ commit }) {
    return new Promise((resolve) => {
      getSetting().then(response => {
        const { data } = response
        const safe = sanitizeAppInfo(data)
        commit('SET_INFO', safe)
        resolve(safe)
      }).catch(() => {
        commit('SET_INFO', defaultAppInfo)
        resolve(defaultAppInfo)
      })
    })
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
