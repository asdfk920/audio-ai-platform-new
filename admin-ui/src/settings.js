module.exports = {
  title: 'go-admin后台管理系统',

  /**
   * @type {boolean} true | false
   * @description Whether show the settings right-panel（右侧悬浮齿轮 / 主题抽屉）
   */
  showSettings: false,

  /**
   * 是否显示顶部导航
   */
  topNav: true,

  /**
   * @type {boolean} true | false
   * @description Whether need tagsView
   */
  tagsView: true,

  /**
   * @type {boolean} true | false
   * @description Whether fix the header
   */
  fixedHeader: true,

  /**
   * @type {boolean} true | false
   * @description Whether show the logo in sidebar
   */
  sidebarLogo: true,

  /**
   * @type {string | array} 'production' | ['production', 'development']
   * @description Need show err logs component.
   * The default is only used in the production env
   * If you want to also use it in dev, you can pass ['production', 'development']
   */
  errorLog: 'production',

  themeStyle: 'dark'
}
