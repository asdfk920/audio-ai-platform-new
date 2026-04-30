function sanitizeAppLogo(info) {
  if (!info || typeof info !== 'object') return info
  const logo = String(info.sys_app_logo || '')
  if (logo.includes('doc-image.zhangwj.com') || logo.includes('zhangwj.com/img/')) {
    return { ...info, sys_app_logo: '/favicon.ico' }
  }
  return info
}

const getters = {
  sidebar: state => state.app.sidebar,
  size: state => state.app.size,
  device: state => state.app.device,
  visitedViews: state => state.tagsView.visitedViews,
  cachedViews: state => state.tagsView.cachedViews,
  token: state => state.user.token,
  avatar: state => state.user.avatar,
  name: state => state.user.name,
  introduction: state => state.user.introduction,
  roles: state => state.user.roles,
  permisaction: state => state.user.permisaction,
  permission_routes: state => state.permission.routes,
  topbarRouters: state => state.permission.topbarRouters,
  defaultRoutes: state => state.permission.defaultRoutes,
  sidebarRouters: state => state.permission.sidebarRouters,
  errorLogs: state => state.errorLog.logs,
  appInfo: state => sanitizeAppLogo(state.system.info)
}
export default getters
