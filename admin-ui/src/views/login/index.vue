<template>
  <div class="login-container">
    <div id="particles-js">
      <!-- <vue-particles
        v-if="refreshParticles"
        color="#dedede"
        :particle-opacity="0.7"
        :particles-number="80"
        shape-type="circle"
        :particle-size="4"
        lines-color="#dedede"
        :lines-width="1"
        :line-linked="true"
        :line-opacity="0.4"
        :lines-distance="150"
        :move-speed="3"
        :hover-effect="true"
        hover-mode="grab"
        :click-effect="true"
        click-mode="push"
      /> -->
    </div>

    <div class="login-weaper animated bounceInDown">
      <div class="login-left">
        <div class="login-time" v-text="currentTime" />
        <img :src="sysInfo.sys_app_logo" alt="" class="img">
        <p class="title" v-text="sysInfo.sys_app_name" />
      </div>
      <div class="login-border">
        <div class="login-main">
          <div class="auth-mode-tabs">
            <el-radio-group v-model="authMode" size="medium" @change="onAuthModeChange">
              <el-radio-button label="login">登录</el-radio-button>
              <el-radio-button label="register">注册</el-radio-button>
            </el-radio-group>
          </div>
          <el-form
            v-show="authMode === 'login'"
            ref="loginForm"
            :model="loginForm"
            :rules="loginRules"
            class="login-form"
            autocomplete="on"
            label-position="left"
          >
            <el-form-item prop="username">
              <span class="svg-container">
                <i class="el-icon-user" />
              </span>
              <el-input
                ref="username"
                v-model="loginForm.username"
                placeholder="用户名"
                name="username"
                type="text"
                tabindex="1"
                autocomplete="on"
              />
            </el-form-item>

            <el-tooltip
              v-model="capsTooltip"
              content="Caps lock is On"
              placement="right"
              manual
            >
              <el-form-item prop="password">
                <span class="svg-container">
                  <svg-icon icon-class="password" />
                </span>
                <el-input
                  :key="passwordType"
                  ref="password"
                  v-model="loginForm.password"
                  :type="passwordType"
                  placeholder="密码"
                  name="password"
                  tabindex="2"
                  autocomplete="on"
                  @keyup.native="checkCapslock"
                  @blur="capsTooltip = false"
                  @keyup.enter.native="handleLogin"
                />
                <span class="show-pwd" @click="showPwd">
                  <svg-icon
                    :icon-class="
                      passwordType === 'password' ? 'eye' : 'eye-open'
                    "
                  />
                </span>
              </el-form-item>
            </el-tooltip>
            <el-form-item prop="code" style="width: 66%; float: left">
              <span class="svg-container">
                <svg-icon icon-class="validCode" />
              </span>
              <el-input
                ref="username"
                v-model="loginForm.code"
                placeholder="验证码"
                name="username"
                type="text"
                tabindex="3"
                maxlength="5"
                autocomplete="off"
                style="width: 75%"
                @keyup.enter.native="handleLogin"
              />
            </el-form-item>
            <div
              class="login-code"
              style="
                cursor: pointer;
                width: 30%;
                height: 48px;
                float: right;
                background-color: #f0f1f5;
              "
            >
              <img
                style="
                  height: 48px;
                  width: 100%;
                  border: 1px solid rgba(0, 0, 0, 0.1);
                  border-radius: 5px;
                "
                :src="codeUrl"
                alt="验证码"
                @click="getCode"
                @error="onCaptchaImgError"
              >
            </div>

            <el-button
              :loading="loading"
              type="primary"
              style="width: 100%; padding: 12px 20px; margin-bottom: 30px"
              @click.native.prevent="handleLogin"
            >
              <span v-if="!loading">登 录</span>
              <span v-else>登 录 中...</span>
            </el-button>
          </el-form>

          <template v-if="authMode === 'register'">
            <el-form
              ref="setupFormRef"
              :model="setupForm"
              :rules="setupRules"
              class="login-form"
              label-position="left"
            >
            <el-alert
              :title="needsSetup ? '数据库中尚无后台管理员，请创建首个账号并选择身份' : '新建后台管理员账号（仅写入 sys_admin，与前台用户互不影响）'"
              type="info"
              :closable="false"
              show-icon
              style="margin-bottom: 16px"
            />
            <el-form-item prop="roleSlug" class="setup-role-row">
              <div class="setup-role-label">管理员身份</div>
              <el-radio-group v-model="setupForm.roleSlug" size="small">
                <el-radio
                  v-for="opt in setupRoleOptions"
                  :key="opt.slug"
                  :label="opt.slug"
                  border
                >
                  {{ opt.label }}
                </el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item prop="username">
              <span class="svg-container">
                <i class="el-icon-user" />
              </span>
              <el-input
                v-model="setupForm.username"
                placeholder="用户名（字母开头，3–64 位）"
                autocomplete="off"
              />
            </el-form-item>
            <el-form-item prop="password">
              <span class="svg-container">
                <svg-icon icon-class="password" />
              </span>
              <el-input
                :key="setupPasswordType"
                v-model="setupForm.password"
                :type="setupPasswordType"
                placeholder="密码（≥8 位，含大小写、数字、特殊字符）"
                autocomplete="new-password"
              />
              <span class="show-pwd" @click="toggleSetupPwd('password')">
                <svg-icon :icon-class="setupPasswordType === 'password' ? 'eye' : 'eye-open'" />
              </span>
            </el-form-item>
            <el-form-item prop="password2">
              <span class="svg-container">
                <svg-icon icon-class="password" />
              </span>
              <el-input
                :key="setupPassword2Type"
                v-model="setupForm.password2"
                :type="setupPassword2Type"
                placeholder="确认密码"
                autocomplete="new-password"
              />
              <span class="show-pwd" @click="toggleSetupPwd('password2')">
                <svg-icon :icon-class="setupPassword2Type === 'password' ? 'eye' : 'eye-open'" />
              </span>
            </el-form-item>
            <el-form-item prop="nickname">
              <span class="svg-container">
                <i class="el-icon-postcard" />
              </span>
              <el-input v-model="setupForm.nickname" placeholder="昵称（可选）" />
            </el-form-item>
            <el-form-item prop="code" style="width: 66%; float: left">
              <span class="svg-container">
                <svg-icon icon-class="validCode" />
              </span>
              <el-input
                v-model="loginForm.code"
                placeholder="验证码"
                maxlength="5"
                style="width: 75%"
                @keyup.enter.native="handleSetupRegister"
              />
            </el-form-item>
            <div
              class="login-code"
              style="
                cursor: pointer;
                width: 30%;
                height: 48px;
                float: right;
                background-color: #f0f1f5;
              "
            >
              <img
                style="
                  height: 48px;
                  width: 100%;
                  border: 1px solid rgba(0, 0, 0, 0.1);
                  border-radius: 5px;
                "
                :src="codeUrl"
                alt="验证码"
                @click="getCode"
                @error="onCaptchaImgError"
              >
            </div>
            <el-button
              :loading="loading"
              type="primary"
              style="width: 100%; padding: 12px 20px; margin-bottom: 30px"
              @click.native.prevent="handleSetupRegister"
            >
              <span v-if="!loading">创建并前往登录</span>
              <span v-else>提交中...</span>
            </el-button>
          </el-form>
          </template>
        </div>
      </div>
    </div>

    <el-dialog title="Or connect with" :visible.sync="showDialog" :close-on-click-modal="false">
      Can not be simulated on local, so please combine you own business
      simulation! ! !
      <br>
      <br>
      <br>
      <social-sign />
    </el-dialog>
    <div
      id="bottom_layer"
      class="s-bottom-layer s-isindex-wrap"
      style="visibility: visible; width: 100%"
    >
      <div class="s-bottom-layer-content">

        <div class="lh">
          <a class="text-color" href="https://beian.miit.gov.cn" target="_blank">
            沪ICP备XXXXXXXXX号-1
          </a>
        </div>
        <div class="open-content-info">
          <div class="tip-hover-panel" style="top: -18px; right: -12px">
            <div class="rest_info_tip">
              <div class="tip-wrapper">
                <div class="lh tip-item" style="display: none">
                  <a
                    class="text-color"
                    href="https://beian.miit.gov.cn"
                    target="_blank"
                  >
                    沪ICP备XXXXXXXXX号-1
                  </a>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { getCodeImg } from '@/api/login'
import { getSetupStatus, postSetupRegister, postAdminAccountRegister } from '@/api/setup'
import moment from 'moment'
import SocialSign from './components/SocialSignin'

export default {
  name: 'Login',
  components: { SocialSign },
  data() {
    return {
      codeUrl: '',
      cookiePassword: '',
      refreshParticles: true,
      loginForm: {
        username: 'admin',
        password: 'admin123',
        rememberMe: false,
        code: '',
        uuid: ''
      },
      loginRules: {
        username: [
          { required: true, trigger: 'blur', message: '用户名不能为空' }
        ],
        password: [
          { required: true, trigger: 'blur', message: '密码不能为空' }
        ],
        code: [
          { required: true, trigger: 'change', message: '验证码不能为空' }
        ]
      },
      needsSetup: false,
      /** login：账号登录；register：冷启动首账号注册（仅 needsSetup 时可用表单） */
      authMode: 'login',
      /** 与后端 GET /setup/status 的 setupRoles 对齐；接口失败时用默认三项 */
      setupRoleOptions: [
        { slug: 'super_admin', label: '超级管理员' },
        { slug: 'operator', label: '运营管理员' },
        { slug: 'finance', label: '财务管理员' }
      ],
      setupForm: {
        roleSlug: 'super_admin',
        username: '',
        password: '',
        password2: '',
        nickname: ''
      },
      setupRules: {
        roleSlug: [{ required: true, trigger: 'change', message: '请选择管理员身份' }],
        username: [{ required: true, trigger: 'blur', message: '用户名不能为空' }],
        password: [{ required: true, trigger: 'blur', message: '密码不能为空' }],
        password2: [{ required: true, trigger: 'blur', message: '请再次输入密码' }]
      },
      passwordType: 'password',
      setupPasswordType: 'password',
      setupPassword2Type: 'password',
      capsTooltip: false,
      loading: false,
      showDialog: false,
      redirect: undefined,
      otherQuery: {},
      currentTime: null,
      sysInfo: {
        sys_app_name: 'go-admin',
        sys_app_logo: '/favicon.ico'
      }
    }
  },
  watch: {
    $route: {
      handler: function(route) {
        const query = route.query
        if (query) {
          this.redirect = query.redirect
          this.otherQuery = this.getOtherQuery(query)
        }
      },
      immediate: true
    }
  },
  created() {
    getSetupStatus()
      .then((res) => {
        const d = res && res.data !== undefined ? res.data : res
        // #region agent log
        fetch('http://127.0.0.1:7281/ingest/098c2c1f-e0e8-4ec4-a6e0-ba69d53941cd',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'a35f07'},body:JSON.stringify({sessionId:'a35f07',runId:'setup-status',hypothesisId:'H13',location:'admin-ui/src/views/login/index.vue:created.then',message:'setup status response parsed in frontend',data:{hasData:!!d,needsSetup:!!(d&&d.needsSetup),keys:d&&typeof d==='object'?Object.keys(d):[]},timestamp:Date.now()})}).catch(()=>{})
        // #endregion
        this.needsSetup = !!(d && d.needsSetup)
        if (d && Array.isArray(d.setupRoles) && d.setupRoles.length) {
          this.setupRoleOptions = d.setupRoles.map((r) => ({
            slug: r.slug,
            label: r.label || r.slug
          }))
        }
        const qMode = this.$route && this.$route.query && this.$route.query.mode
        if (qMode === 'register' || qMode === 'login') {
          this.authMode = qMode
        } else if (this.needsSetup) {
          this.authMode = 'register'
        } else {
          this.authMode = 'login'
        }
      })
      .catch((err) => {
        // #region agent log
        fetch('http://127.0.0.1:7281/ingest/098c2c1f-e0e8-4ec4-a6e0-ba69d53941cd',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'a35f07'},body:JSON.stringify({sessionId:'a35f07',runId:'setup-status',hypothesisId:'H14',location:'admin-ui/src/views/login/index.vue:created.catch',message:'setup status request failed in frontend',data:{message:err&&err.message?err.message:'',status:err&&err.response&&err.response.status?err.response.status:null},timestamp:Date.now()})}).catch(()=>{})
        // #endregion
      })
    this.getCode()
    // window.addEventListener('storage', this.afterQRScan)
    this.getCurrentTime()
    this.getSystemSetting()
  },
  mounted() {
    if (this.authMode === 'login') {
      if (this.loginForm.username === '' && this.$refs.username) {
        this.$refs.username.focus()
      } else if (this.loginForm.password === '' && this.$refs.password) {
        this.$refs.password.focus()
      }
    }
    window.addEventListener('resize', () => {
      this.refreshParticles = false
      this.$nextTick(() => (this.refreshParticles = true))
    })
  },
  destroyed() {
    clearInterval(this.timer)
    window.removeEventListener('resize', () => {})
    // window.removeEventListener('storage', this.afterQRScan)
  },
  methods: {
    onCaptchaImgError() {
      this.codeUrl = ''
      this.getCode()
    },
    onAuthModeChange() {
      this.$nextTick(() => {
        if (this.authMode === 'login' && this.$refs.username) {
          this.$refs.username.focus()
        }
        this.getCode()
      })
    },
    getSystemSetting() {
      this.$store.dispatch('system/settingDetail').then((ret) => {
        // 安全检查：确保返回值存在
        if (ret && typeof ret === 'object') {
          this.sysInfo = ret
          if (ret.sys_app_name) {
            document.title = ret.sys_app_name
          }
        }
      }).catch(() => {
        // 失败时使用默认值
        if (this.sysInfo && this.sysInfo.sys_app_name) {
          document.title = this.sysInfo.sys_app_name
        } else {
          document.title = 'go-admin'
        }
      })
    },
    getCurrentTime() {
      this.timer = setInterval((_) => {
        this.currentTime = moment().format('YYYY-MM-DD HH时mm分ss秒')
      }, 1000)
    },
    getCode() {
      getCodeImg().then((res) => {
        console.log('验证码响应:', res)
        // 安全检查：确保 res 和 res.data 存在
        if (!res || !res.data) {
          console.warn('验证码返回数据为空，使用默认值')
          if (process.env.NODE_ENV === 'development') {
            this.loginForm.code = '0'
            this.loginForm.uuid = '0'
          }
          return
        }

        // 后端返回格式：{ captchaId: 'xxx', image: 'data:image/png;base64,...' }
        const captchaData = res.data
        const captchaId = captchaData.captchaId || captchaData.id || res.id || ''
        let captchaImage = captchaData.image || captchaData

        console.log('验证码 ID:', captchaId)
        console.log('验证码图片:', captchaImage ? '有数据' : '空')
        console.log('图片前 50 字符:', captchaImage ? captchaImage.substring(0, 50) : '无')

        // 确保 base64 有正确的图片前缀
        if (captchaImage && typeof captchaImage === 'string' && !captchaImage.startsWith('data:image')) {
          captchaImage = 'data:image/png;base64,' + captchaImage
        }

        this.codeUrl = captchaImage
        this.loginForm.uuid = captchaId

        console.log('最终验证码 URL:', this.codeUrl ? '设置成功' : '设置失败')

        // 不在此处把 code 写成占位符：本地前端常为 development，而后端仍可能是 prod（会校验验证码），
        // 强制 '0' 会导致与图片不一致、登录失败且看起来像「验证码没了」。
      }).catch((err) => {
        console.error('验证码请求失败:', err)
        if (process.env.NODE_ENV === 'development') {
          this.loginForm.code = '0'
          this.loginForm.uuid = '0'
        }
      })
    },
    checkCapslock({ shiftKey, key } = {}) {
      if (key && key.length === 1) {
        if (
          (shiftKey && key >= 'a' && key <= 'z') ||
          (!shiftKey && key >= 'A' && key <= 'Z')
        ) {
          this.capsTooltip = true
        } else {
          this.capsTooltip = false
        }
      }
      if (key === 'CapsLock' && this.capsTooltip === true) {
        this.capsTooltip = false
      }
    },
    showPwd() {
      if (this.passwordType === 'password') {
        this.passwordType = ''
      } else {
        this.passwordType = 'password'
      }
      this.$nextTick(() => {
        this.$refs.password.focus()
      })
    },
    toggleSetupPwd(field) {
      if (field === 'password') {
        this.setupPasswordType = this.setupPasswordType === 'password' ? '' : 'password'
        return
      }
      this.setupPassword2Type = this.setupPassword2Type === 'password' ? '' : 'password'
    },
    handleSetupRegister() {
      if (process.env.NODE_ENV === 'development') {
        if (!this.loginForm.code) this.loginForm.code = '0'
        if (!this.loginForm.uuid) this.loginForm.uuid = '0'
      }
      if (!this.loginForm.code) {
        this.$message.warning('请填写验证码')
        return
      }
      this.$refs.setupFormRef.validate((valid) => {
        if (!valid) return
        if (this.setupForm.password !== this.setupForm.password2) {
          this.$message.error('两次输入的密码不一致')
          return
        }
        this.loading = true
        // needsSetup=true 时仍走冷启动专用接口（校验「首次安装」语义）；
        // 其余情况走通用管理员账号注册接口，仅写入 sys_admin。
        const payload = {
          roleSlug: this.setupForm.roleSlug,
          username: this.setupForm.username.trim(),
          password: this.setupForm.password,
          nickname: (this.setupForm.nickname || '').trim(),
          code: this.loginForm.code,
          uuid: this.loginForm.uuid
        }
        const reqFn = this.needsSetup ? postSetupRegister : postAdminAccountRegister
        reqFn(payload)
          .then(() => {
            this.$message.success('创建成功，请使用新账号登录')
            this.needsSetup = false
            this.authMode = 'login'
            this.loginForm.username = this.setupForm.username.trim()
            this.loginForm.password = ''
            this.setupForm.password = ''
            this.setupForm.password2 = ''
            this.setupPasswordType = 'password'
            this.setupPassword2Type = 'password'
            this.getCode()
            this.$nextTick(() => {
              if (this.$refs.username) this.$refs.username.focus()
            })
          })
          .catch(() => {
            this.getCode()
          })
          .finally(() => {
            this.loading = false
          })
      })
    },
    handleLogin() {
      if (process.env.NODE_ENV === 'development') {
        if (!this.loginForm.code) this.loginForm.code = '0'
        if (!this.loginForm.uuid) this.loginForm.uuid = '0'
      }
      this.$refs.loginForm.validate((valid) => {
        if (valid) {
          this.loading = true
          this.$store
            .dispatch('user/login', this.loginForm)
            .then(() => {
              // 先结束按钮 loading：router.push 会等待全局守卫里 getInfo 完成，若后端慢或卡住，否则会一直显示「登录中…」
              this.loading = false
              this.$router
                .push({ path: this.redirect || '/', query: this.otherQuery })
                .catch(() => {})
            })
            .catch((err) => {
              console.error(err)
            })
        } else {
          console.log('error submit!!')
          return false
        }
      })
    },
    getOtherQuery(query) {
      return Object.keys(query).reduce((acc, cur) => {
        if (cur !== 'redirect') {
          acc[cur] = query[cur]
        }
        return acc
      }, {})
    }
  }
}
</script>

<style lang="scss" scoped>
/* 修复input 背景不协调 和光标变色 */
/* Detail see https://github.com/PanJiaChen/vue-element-admin/pull/927 */

.setup-role-row {
  clear: both;
  width: 100%;
  margin-bottom: 8px;
  float: none;
}
.setup-role-row ::v-deep .el-form-item__content {
  margin-left: 0 !important;
  display: block;
}
.setup-role-label {
  font-size: 13px;
  color: #606266;
  margin-bottom: 8px;
  text-align: left;
}
.setup-role-row .el-radio-group {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
}

.auth-mode-tabs {
  text-align: center;
  margin-bottom: 18px;
  margin-top: 4px;
}
.auth-mode-tabs .el-radio-group {
  display: inline-flex;
}
$bg: #283443;
$light_gray: #fff;
$cursor: #fff;

#bottom_layer {
  visibility: hidden;
  width: 3000px;
  position: fixed;
  z-index: 302;
  bottom: 0;
  left: 0;
  height: 39px;
  padding-top: 1px;
  zoom: 1;
  margin: 0;
  line-height: 39px;
  // background: #0e6cff;
}
#bottom_layer .lh {
  display: inline-block;
  margin-right: 14px;
}
#bottom_layer .lh .emphasize {
  text-decoration: underline;
  font-weight: 700;
}
#bottom_layer .lh:last-child {
  margin-left: -2px;
  margin-right: 0;
}
#bottom_layer .lh.activity {
  font-weight: 700;
  text-decoration: underline;
}
#bottom_layer a {
  font-size: 12px;
  text-decoration: none;
}
#bottom_layer .text-color {
  color: #bbb;
}
#bottom_layer .aria-img {
  width: 49px;
  height: 20px;
  margin-bottom: -5px;
}
#bottom_layer a:hover {
  color: #fff;
}
#bottom_layer .s-bottom-layer-content {
  margin: 0 17px;
  text-align: center;
}
#bottom_layer .s-bottom-layer-content .auto-transform-line {
  display: inline;
}
#bottom_layer .s-bottom-layer-content .auto-transform-line:first-child {
  margin-right: 14px;
}
.s-bottom-space {
  position: static;
  width: 100%;
  height: 40px;
  margin: 23px auto 12px;
}
#bottom_layer .open-content-info a:hover {
  color: #fff;
}
#bottom_layer .open-content-info .text-color {
  color: #626675;
}
.open-content-info {
  position: relative;
  display: inline-block;
  width: 20px;
}
.open-content-info > span {
  cursor: pointer;
  font-size: 14px;
}
.open-content-info > span:hover {
  color: #fff;
}
.open-content-info .tip-hover-panel {
  position: absolute;
  display: none;
  padding-bottom: 18px;
}
.open-content-info .tip-hover-panel .rest_info_tip {
  max-width: 560px;
  padding: 8px 12px 8px 12px;
  background: #fff;
  border-radius: 6px;
  border: 1px solid rgba(0, 0, 0, 0.05);
  box-shadow: 0 2px 4px 0 rgba(0, 0, 0, 0.1);
  text-align: left;
}
.open-content-info .tip-hover-panel .rest_info_tip .tip-wrapper {
  white-space: nowrap;
  line-height: 20px;
}
.open-content-info .tip-hover-panel .rest_info_tip .tip-wrapper .tip-item {
  height: 20px;
  line-height: 20px;
}
.open-content-info
  .tip-hover-panel
  .rest_info_tip
  .tip-wrapper
  .tip-item:last-child {
  margin-right: 0;
}
@media screen and (max-width: 515px) {
  .open-content-info {
    width: 16px;
  }
  .open-content-info .tip-hover-panel {
    right: -16px !important;
  }
}
.footer {
  background-color: #0e6cff;
  margin-bottom: -20px;
}

.login-container {
  display: -webkit-box;
  display: -ms-flexbox;
  display: flex;
  -webkit-box-align: center;
  -ms-flex-align: center;
  align-items: center;
  width: 100%;
  height: 100%;
  margin: 0 auto;
  background: url("../../assets/login.png") no-repeat;
  background-color: #0e6cff;
  position: relative;
  background-size: cover;
  height: 100vh;
  background-position: 50%;
}

#particles-js {
  z-index: 1;
  width: 100%;
  height: 100%;
  position: absolute;
}

.login-weaper {
  margin: 0 auto;
  width: 1000px;
  -webkit-box-shadow: -4px 5px 10px rgba(0, 0, 0, 0.4);
  box-shadow: -4px 5px 10px rgba(0, 0, 0, 0.4);
  z-index: 1000;
}

.login-left {
  border-top-left-radius: 5px;
  border-bottom-left-radius: 5px;
  -webkit-box-pack: center;
  -ms-flex-pack: center;
  justify-content: center;
  -webkit-box-orient: vertical;
  -webkit-box-direction: normal;
  -ms-flex-direction: column;
  flex-direction: column;
  background-color: rgba(64, 158, 255, 0);
  color: #fff;
  float: left;
  width: 50%;
  position: relative;
  min-height: 500px;
  -webkit-box-align: center;
  -ms-flex-align: center;
  align-items: center;
  display: -webkit-box;
  display: -ms-flexbox;
  display: flex;
  .login-time {
    position: absolute;
    left: 25px;
    top: 25px;
    width: 100%;
    color: #fff;
    opacity: 0.9;
    font-size: 18px;
    overflow: hidden;
    font-weight: 500;
  }
}

.login-left .img {
  width: 90px;
  height: 90px;
  border-radius: 3px;
}

.login-left .title {
  text-align: center;
  color: #fff;
  letter-spacing: 2px;
  font-size: 16px;
  font-weight: 600;
}

.login-border {
  position: relative;
  min-height: 500px;
  -webkit-box-align: center;
  -ms-flex-align: center;
  align-items: center;
  display: -webkit-box;
  display: -ms-flexbox;
  display: flex;
  border-left: none;
  border-top-right-radius: 5px;
  border-bottom-right-radius: 5px;
  color: #fff;
  background-color: hsla(0, 0%, 100%, 0.9);
  width: 50%;
  float: left;
}

.login-main {
  margin: 0 auto;
  width: 65%;
}

.login-title {
  color: #333;
  margin-bottom: 40px;
  font-weight: 500;
  font-size: 22px;
  text-align: center;
  letter-spacing: 4px;
}

@supports (-webkit-mask: none) and (not (cater-color: $cursor)) {
  .login-container .el-input input {
    color: $cursor;
  }
}

/* reset element-ui css */
.login-container {
  ::v-deep .el-input {
    display: inline-block;
    height: 47px;
    width: 85%;

    input {
      background: transparent;
      border: 0px;
      -webkit-appearance: none;
      border-radius: 0px;
      padding: 12px 5px 12px 15px;
      color: #333;
      height: 47px;
      caret-color: #333;

      &:-webkit-autofill {
        box-shadow: 0 0 0px 1000px $bg inset !important;
        -webkit-text-fill-color: $cursor !important;
      }
    }
  }

  .el-form-item {
    border: 1px solid rgba(0, 0, 0, 0.1);
    background: rgba(255, 255, 255, 0.8);
    border-radius: 5px;
    color: #454545;
  }
}
$bg: #2d3a4b;
$dark_gray: #889aa4;
$light_gray: #eee;

.login-container {
  .tips {
    font-size: 14px;
    color: #fff;
    margin-bottom: 10px;

    span {
      &:first-of-type {
        margin-right: 16px;
      }
    }
  }

  .svg-container {
    padding: 6px 5px 6px 15px;
    color: $dark_gray;
    vertical-align: middle;
    width: 30px;
    display: inline-block;
  }

  .title-container {
    position: relative;

    .title {
      font-size: 26px;
      color: $light_gray;
      margin: 0px auto 40px auto;
      text-align: center;
      font-weight: bold;
    }
  }

  .show-pwd {
    position: absolute;
    right: 10px;
    top: 7px;
    font-size: 16px;
    color: $dark_gray;
    cursor: pointer;
    user-select: none;
  }

  .thirdparty-button {
    position: absolute;
    right: 0;
    bottom: 6px;
  }

  @media only screen and (max-width: 470px) {
    .thirdparty-button {
      display: none;
    }
    .login-weaper {
      width: 100%;
      padding: 0 30px;
      box-sizing: border-box;
      box-shadow: none;
    }
    .login-main {
      width: 80%;
    }
    .login-left {
      display: none !important;
    }
    .login-border {
      width: 100%;
    }
  }
}
</style>
