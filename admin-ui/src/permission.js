import router, { resetRouter } from './router'
import store from './store'
import { Message } from 'element-ui'
import NProgress from 'nprogress' // progress bar
import 'nprogress/nprogress.css' // progress bar style
import { getToken } from '@/utils/auth' // get token from cookie
import getPageTitle from '@/utils/get-page-title'

NProgress.configure({ showSpinner: false }) // NProgress Configuration

const whiteList = ['/login', '/auth-redirect'] // no redirect whitelist

/** axios 拒绝体常为 { code, msg } 对象；直接 Message.error(对象) 会显示空白红条 */
function formatGuardErrorMessage(err, fallback = '加载失败，请重新登录') {
  if (err == null || err === '') return fallback
  if (typeof err === 'string') return err
  if (err instanceof Error && err.message) return err.message
  if (typeof err === 'object') {
    const m = err.msg || err.message
    if (m != null && String(m).trim() !== '') return String(m)
    if (err.code != null) return `${fallback}（${err.code}）`
  }
  return fallback
}

// #region agent log
function agentDebugLog(hypothesisId, message, data) {
  // 默认关闭：避免本机未启动 ingest 服务时报 ERR_CONNECTION_REFUSED 刷屏
  // 需要开启时，在浏览器控制台执行：window.__ENABLE_AGENT_DEBUG__ = true
  if (typeof window !== 'undefined' && window.__ENABLE_AGENT_DEBUG__ !== true) return
  try {
    fetch('http://127.0.0.1:7559/ingest/97f27dd3-163b-4d88-a4e7-e2b55236be17', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Debug-Session-Id': 'e66055' },
      body: JSON.stringify({
        sessionId: 'e66055',
        runId: 'ui-lag',
        hypothesisId,
        location: 'admin-ui/src/permission.js',
        message,
        data: data || {},
        timestamp: Date.now()
      })
    }).catch(() => { /* ignore */ })
  } catch (e) { /* ignore */ }
}
// #endregion

router.beforeEach(async(to, from, next) => {
  // start progress bar
  NProgress.start()

  // set page title
  document.title = getPageTitle(to.meta.title)

  // determine whether the user has logged in
  const hasToken = getToken()
  // sidebarRouters 可能被 TopNav 交互清空，不能用来判断“是否已注入动态路由”
  const hasAccessRoutes = Boolean(store.state.permission && store.state.permission.addRoutes && store.state.permission.addRoutes.length > 0)

  // #region agent log
  agentDebugLog('H1', 'route_guard_enter', {
    to: to && to.path,
    from: from && from.path,
    hasToken: Boolean(hasToken),
    hasRoles: Boolean(store.getters.roles && store.getters.roles.length > 0),
    hasSidebar: Boolean(store.getters.sidebarRouters && store.getters.sidebarRouters.length > 0),
    hasAccessRoutes
  })
  // #endregion

  if (hasToken) {
    if (to.path === '/login') {
      // if is logged in, redirect to the home page
      next({ path: '/' })
      NProgress.done()
    } else {
      // determine whether the user has obtained his permission roles through getInfo
      const hasRoles = store.getters.roles && store.getters.roles.length > 0
      // 注意：sidebarRouters 会被 TopNav 切换覆盖为空数组；用 addRoutes 判断是否已注入动态路由更可靠
      if (hasRoles && hasAccessRoutes) {
        // #region agent log
        agentDebugLog('H2', 'route_guard_fastpath_next', { to: to && to.path })
        // #endregion
        next()
      } else {
        try {
          if (!hasRoles) {
            // #region agent log
            const t0 = Date.now()
            agentDebugLog('H3', 'getInfo_start', {})
            // #endregion
            await store.dispatch('user/getInfo')
            // #region agent log
            agentDebugLog('H3', 'getInfo_done', { ms: Date.now() - t0 })
            // #endregion
          }
          // 只有在尚未注入动态路由时才重置并重建，避免每次跳转都触发 generateRoutes 循环
          if (!hasAccessRoutes) {
            resetRouter()
          }
          // #region agent log
          const t1 = Date.now()
          agentDebugLog('H4', 'generateRoutes_start', {})
          // #endregion
          const accessRoutes = !hasAccessRoutes
            ? await store.dispatch('permission/generateRoutes', store.getters.roles)
            : store.state.permission.addRoutes
          // #region agent log
          agentDebugLog('H4', 'generateRoutes_done', { ms: Date.now() - t1, routes: Array.isArray(accessRoutes) ? accessRoutes.length : -1 })
          // #endregion
          if (!hasAccessRoutes) {
            router.addRoutes(accessRoutes)
            next({ ...to, replace: true })
          } else {
            next()
          }
        } catch (error) {
          // #region agent log
          agentDebugLog('H5', 'route_guard_error', {
            to: to && to.path,
            err: error ? String(error).slice(0, 220) : null
          })
          // #endregion
          Message.error(formatGuardErrorMessage(error))
          next(`/login?redirect=${to.path}`)
          NProgress.done()
        }
      }
    }
  } else {
    /* has no token*/

    if (whiteList.indexOf(to.path) !== -1) {
      // in the free login whitelist, go directly
      // #region agent log
      agentDebugLog('H6', 'route_guard_whitelist_next', { to: to && to.path })
      // #endregion
      next()
    } else {
      // other pages that do not have permission to access are redirected to the login page.
      // #region agent log
      agentDebugLog('H6', 'route_guard_redirect_login', { to: to && to.path })
      // #endregion
      next(`/login?redirect=${to.path}`)
      NProgress.done()
    }
  }
})

router.afterEach(() => {
  // finish progress bar
  NProgress.done()
})
