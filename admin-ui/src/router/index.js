import Vue from 'vue'
import Router from 'vue-router'

Vue.use(Router)

/* Layout */
import Layout from '@/layout'

/* Router Modules */
// import componentsRouter from './modules/components'
// import chartsRouter from './modules/charts'
// import tableRouter from './modules/table'
// import nestedRouter from './modules/nested'

/**
 * Note: sub-menu only appear when route children.length >= 1
 * Detail see: https://panjiachen.github.io/vue-element-admin-site/guide/essentials/router-and-nav.html
 *
 * hidden: true                   if set true, item will not show in the sidebar(default is false)
 * alwaysShow: true               if set true, will always show the root menu
 *                                if not set alwaysShow, when item has more than one children route,
 *                                it will becomes nested mode, otherwise not show the root menu
 * redirect: noRedirect           if set noRedirect will no redirect in the breadcrumb
 * name:'router-name'             the name is used by <keep-alive> (must set!!!)
 * meta : {
    roles: ['admin','editor']    control the page roles (you can set multiple roles)
    title: 'title'               the name show in sidebar and breadcrumb (recommend set)
    icon: 'svg-name'             the icon show in the sidebar
    noCache: true                if set true, the page will no be cached(default is false)
    affix: true                  if set true, the tag will affix in the tags-view
    breadcrumb: false            if set false, the item will hidden in breadcrumb(default is true)
    activeMenu: '/example/list'  if set path, the sidebar will highlight the path you set
  }
 */

/**
 * constantRoutes
 * a base page that does not have permission requirements
 * all roles can be accessed
 */
export const constantRoutes = [
  {
    path: '/redirect',
    component: Layout,
    hidden: true,
    children: [
      {
        path: '/redirect/:path*',
        component: () => import('@/views/redirect/index')
      }
    ]
  },
  {
    path: '/login',
    component: () => import('@/views/login/index'),
    hidden: true
  },
  {
    path: '/auth-redirect',
    component: () => import('@/views/login/auth-redirect'),
    hidden: true
  },
  {
    path: '/404',
    component: () => import('@/views/error-page/404'),
    hidden: true
  },
  {
    path: '/401',
    component: () => import('@/views/error-page/401'),
    hidden: true
  },
  {
    path: '/',
    component: Layout,
    // 默认进入新首页（登录后落地页）
    redirect: '/home/index',
    children: [
      {
        path: 'dashboard',
        hidden: true,
        component: () => import('@/views/dashboard/index'),
        name: 'Dashboard',
        meta: { title: '首页', icon: 'dashboard' }
      }
    ]
  },
  {
    path: '/home',
    component: Layout,
    redirect: '/home/index',
    children: [
      {
        path: 'index',
        component: () => import('@/views/home/index.vue'),
        name: 'Home',
        meta: { title: '首页', icon: 'dashboard', noCache: true, affix: true }
      }
    ]
  },
  {
    path: '/platform-user',
    component: Layout,
    redirect: '/platform-user/index',
    children: [
      {
        path: 'index',
        component: () => import('@/views/admin/platform-user/index'),
        name: 'PlatformUser',
        meta: { title: '平台用户', icon: 'user', noCache: true, affix: true, platformModule: 'user_mgmt' }
      }
    ]
  },
  {
    path: '/platform-admin',
    component: Layout,
    redirect: '/platform-admin/index',
    children: [
      {
        path: 'index',
        // 指向新的 sys-admin 页面：直接对接 /api/v1/sys-admin/* 接口，涵盖
        // CRUD / 批量删除 / 重置密码 / 安全策略 / 强制改密 / 自助改密
        component: () => import('@/views/admin/sys-admin/index'),
        name: 'PlatformAdmin',
        meta: { title: '管理员管理', icon: 'peoples', noCache: true, platformModule: 'sys_config' }
      }
    ]
  },
  {
    path: '/platform-content',
    component: Layout,
    redirect: '/platform-content/index',
    children: [
      {
        path: 'index',
        component: () => import('@/views/admin/platform-content/index'),
        name: 'PlatformContent',
        meta: { title: '内容管理', icon: 'documentation', noCache: true, breadcrumb: false, platformModule: 'content_mgmt' }
      }
    ]
  },
  {
    path: '/platform-rbac',
    component: Layout,
    redirect: '/platform-rbac/index',
    children: [
      {
        path: 'index',
        component: () => import('@/views/admin/platform-rbac/index'),
        name: 'PlatformRbac',
        meta: { title: '权限管理', icon: 'lock', noCache: true, platformModule: 'sys_config' }
      }
    ]
  },
  {
    path: '/platform-member',
    component: Layout,
    redirect: '/platform-member/index',
    children: [
      {
        path: 'index',
        component: () => import('@/views/admin/platform-member/index'),
        name: 'PlatformMember',
        meta: { title: '会员管理', icon: 'peoples', noCache: true, platformModule: 'member_mgmt' }
      }
    ]
  },
  {
    path: '/platform-realname',
    component: Layout,
    redirect: '/platform-realname/index',
    children: [
      {
        path: 'index',
        component: () => import('@/views/admin/platform-realname/index'),
        name: 'PlatformRealName',
        meta: { title: '实名审核', icon: 'eye', noCache: true, platformModule: 'user_mgmt' }
      }
    ]
  },
  {
    path: '/platform-device',
    component: Layout,
    redirect: '/platform-device/index',
    children: [
      {
        path: 'index',
        component: () => import('@/views/admin/platform-device/index'),
        name: 'PlatformDevice',
        meta: { title: '设备管理', icon: 'component', noCache: true, platformModule: 'device_mgmt' }
      }
    ]
  },
  {
    path: '/platform-product',
    component: Layout,
    redirect: '/platform-product/index',
    children: [
      {
        path: 'index',
        component: () => import('@/views/admin/platform-product/index'),
        name: 'PlatformProduct',
        meta: { title: '产品管理', icon: 'nested', noCache: true, platformModule: 'device_mgmt' }
      }
    ]
  },
  {
    path: '/profile',
    component: Layout,
    redirect: '/profile/index',
    hidden: true,
    children: [
      {
        path: 'index',
        component: () => import('@/views/profile/index'),
        name: 'Profile',
        meta: { title: '个人中心', icon: 'user', noCache: true }
      }
    ]
  }
]

/**
 * asyncRoutes
 * the routes that need to be dynamically loaded based on user roles
 */
export const asyncRoutes = [

]

const createRouter = () => new Router({
  mode: 'history', // require service support
  scrollBehavior: () => ({ y: 0 }),
  routes: constantRoutes
})

const router = createRouter()

// Detail see: https://github.com/vuejs/vue-router/issues/1234#issuecomment-357941465
export function resetRouter() {
  const newRouter = createRouter()
  router.matcher = newRouter.matcher // reset router
}

export default router
