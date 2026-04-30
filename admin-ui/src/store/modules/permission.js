import { constantRoutes } from '@/router'
import { getRoutes } from '@/api/admin/sys-role'
import { getPlatformRbacMatrix } from '@/api/admin/platform-rbac'
import Layout from '@/layout'
// import sysuserindex from '@/views/sysuser/index'

/** router/index 中已声明的 /platform-* 静态路由，避免与后端菜单重复注册导致 vue-router 同名告警 */
const STATIC_PLATFORM_PATHS = new Set(
  constantRoutes
    .filter((r) => r && typeof r.path === 'string' && r.path.startsWith('/platform-'))
    .map((r) => r.path)
)

function filterDuplicatePlatformMenus(menus) {
  if (!Array.isArray(menus)) return []
  const out = []
  for (const item of menus) {
    if (!item) continue
    if (item.path && STATIC_PLATFORM_PATHS.has(item.path)) {
      continue
    }
    const next = { ...item }
    if (item.children && item.children.length) {
      next.children = filterDuplicatePlatformMenus(item.children)
    }
    out.push(next)
  }
  return out
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
        location: 'admin-ui/src/store/modules/permission.js',
        message,
        data: data || {},
        timestamp: Date.now()
      })
    }).catch(() => { /* ignore */ })
  } catch (e) { /* ignore */ }
}
// #endregion

/**
 * topNav=true 时顶部菜单来自 topbarRouters，仅含后端 sys_menu；
 * 未写入库的平台页（如 /platform-device）不会出现在顶部。
 * 将 router/index 里已声明的 /platform-* 静态路由按 path 去重合并到后端菜单之后。
 */
/**
 * 按平台矩阵 modules 隐藏无权的 /platform-* 静态路由（meta.platformModule）
 */
function filterPlatformStaticRoutesByModules(routes, modules) {
  if (!modules || typeof modules !== 'object') {
    return routes
  }
  return routes.map((route) => {
    if (!route || !route.path || typeof route.path !== 'string' || !route.path.startsWith('/platform-')) {
      return route
    }
    const child = route.children && route.children[0]
    const mod = child && child.meta && child.meta.platformModule
    if (!mod) {
      return route
    }
    const v = modules[mod]
    if (v === 'none') {
      return { ...route, hidden: true }
    }
    return route
  })
}

function mergePlatformStaticTopRoutes(dbRoutes, filteredConstantRoutes) {
  const base = filteredConstantRoutes || constantRoutes
  const staticPlatform = base.filter(
    (r) => r && r.path && typeof r.path === 'string' && r.path.startsWith('/platform-') && !r.hidden
  )
  const seen = new Set()
  const out = []
  for (const r of dbRoutes) {
    if (r && r.path) {
      seen.add(r.path)
    }
    out.push(r)
  }
  for (const r of staticPlatform) {
    if (r.path && !seen.has(r.path)) {
      out.push(r)
      seen.add(r.path)
    }
  }
  return out
}

/**
 * Use meta.role to determine if the current user has permission
 * @param roles
 * @param route
 */
function hasPermission(roles, route) {
  if (route.meta && route.meta.roles) {
    return roles.some(role => route.meta.roles.includes(role))
  } else {
    return true
  }
}

/**
 * Use names to determine if the current user has permission
 * @param names
 * @param route
 */
function hasPathPermission(paths, route) {
  if (route.path) {
    return paths.some(path => route.path === path.path)
  } else {
    return true
  }
}

/**
  * 后台查询的菜单数据拼装成路由格式的数据
  * @param routes
  */
export function generaMenu(routes, data) {
  data.forEach(item => {
    const menu = {
      path: item.path,
      component: item.component === 'Layout' ? Layout : loadView(item.component),
      hidden: item.visible !== '0',
      children: [],
      name: item.menuName,
      meta: {
        title: item.title,
        icon: item.icon,
        noCache: item.noCache
      }
    }
    if (item.children) {
      generaMenu(menu.children, item.children)
    }
    routes.push(menu)
  })
}

export const loadView = (view) => { // 路由懒加载
  return (resolve) => require([`@/views${view}`], resolve)
}

/**
 * Filter asynchronous routing tables by recursion
 * @param routes asyncRoutes
 * @param roles
 */
export function filterAsyncRoutes(routes, roles) {
  const res = []

  routes.forEach(route => {
    const tmp = { ...route }
    if (hasPermission(roles, tmp)) {
      if (tmp.children) {
        tmp.children = filterAsyncRoutes(tmp.children, roles)
      }
      res.push(tmp)
    }
  })

  return res
}

/**
 * Filter asynchronous routing tables by recursion
 * @param routes asyncRoutes
 * @param components
 */
export function filterAsyncPathRoutes(routes, paths) {
  const res = []

  routes.forEach(route => {
    const tmp = { ...route }
    if (hasPathPermission(paths, tmp)) {
      if (tmp.children) {
        tmp.children = filterAsyncPathRoutes(tmp.children, paths)
      }
      res.push(tmp)
    }
  })

  return res
}

const state = {
  routes: [],
  addRoutes: [],
  defaultRoutes: [],
  topbarRouters: [],
  sidebarRouters: [],
  /** 当前登录角色在矩阵中的 modules（来自 /platform-rbac/matrix） */
  platformModules: {}
}

const mutations = {
  SET_ROUTES: (state, routes) => {
    state.addRoutes = routes
    state.routes = constantRoutes.concat(routes)
  },
  SET_DEFAULT_ROUTES: (state, routes) => {
    state.defaultRoutes = constantRoutes.concat(routes)
  },
  SET_TOPBAR_ROUTES: (state, routes) => {
    // 顶部导航菜单默认添加统计报表栏指向首页
    // const index = [{
    //   path: 'dashboard',
    //   meta: { title: '统计报表', icon: 'dashboard' }
    // }]
    state.topbarRouters = routes // .concat(index)
  },
  SET_SIDEBAR_ROUTERS: (state, routes) => {
    state.sidebarRouters = routes
  },
  SET_PLATFORM_MODULES: (state, modules) => {
    state.platformModules = modules && typeof modules === 'object' ? modules : {}
  }
}

const actions = {
  generateRoutes({ commit }) {
    return new Promise((resolve) => {
      const applyFallback = () => {
        const empty = []
        commit('SET_ROUTES', empty)
        commit('SET_PLATFORM_MODULES', {})
        commit('SET_SIDEBAR_ROUTERS', constantRoutes.concat(empty))
        commit('SET_DEFAULT_ROUTES', empty)
        commit('SET_TOPBAR_ROUTES', mergePlatformStaticTopRoutes([], constantRoutes))
        resolve(empty)
      }

      // #region agent log
      const t0 = Date.now()
      agentDebugLog('H7', 'menurole_request_start', {})
      // #endregion
      getRoutes()
        .then(async(response) => {
          // #region agent log
          agentDebugLog('H7', 'menurole_request_done', {
            ms: Date.now() - t0,
            ok: Boolean(response && response.code === 200),
            topLen: Array.isArray(response && response.data) ? response.data.length : -1
          })
          // #endregion
          let loadMenuData = Array.isArray(response.data) ? response.data : []
          loadMenuData = filterDuplicatePlatformMenus(loadMenuData)
          if (!response || response.code !== 200) {
            applyFallback()
            return
          }
          const dynamicRoutes = []
          generaMenu(dynamicRoutes, loadMenuData)
          dynamicRoutes.push({ path: '*', redirect: '/', hidden: true })
          commit('SET_ROUTES', dynamicRoutes)
          const sidebarRoutes = []
          generaMenu(sidebarRoutes, loadMenuData)
          let platformMods = {}
          try {
            const mx = await getPlatformRbacMatrix()
            const d = mx && mx.data
            if (d && d.modules && typeof d.modules === 'object') {
              platformMods = d.modules
            }
          } catch (e) {
            console.warn('platform-rbac matrix load failed', e)
          }
          commit('SET_PLATFORM_MODULES', platformMods)
          const filteredConstants = filterPlatformStaticRoutesByModules(constantRoutes, platformMods)
          commit('SET_SIDEBAR_ROUTERS', filteredConstants.concat(sidebarRoutes))
          commit('SET_DEFAULT_ROUTES', sidebarRoutes)
          commit('SET_TOPBAR_ROUTES', mergePlatformStaticTopRoutes(sidebarRoutes, filteredConstants))
          resolve(dynamicRoutes)
        })
        .catch((err) => {
          console.error('menurole load failed', err)
          applyFallback()
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
