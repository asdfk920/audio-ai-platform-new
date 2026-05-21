<template>
  <div class="offline-page">
    <div class="offline-card">
      <div class="offline-icon-wrap">
        <i class="el-icon-warning-outline offline-icon" />
      </div>
      <h1 class="offline-title">网络连接已中断</h1>
      <p class="offline-desc">
        无法连接到管理后台服务，请检查本机网络、代理或后端服务是否正常。
      </p>
      <p v-if="fromHint" class="offline-from">
        恢复后将尝试返回：<code>{{ fromHint }}</code>
      </p>
      <div class="offline-actions">
        <el-button type="primary" :loading="reconnecting" @click="retry">
          重新连接
        </el-button>
        <el-button @click="goHome">
          进入首页
        </el-button>
      </div>
      <p class="offline-tip">
        提示：网络恢复时页面也可自动检测（部分浏览器支持）。
      </p>
    </div>
  </div>
</template>

<script>
import request from '@/utils/request'

export default {
  name: 'OfflineReconnect',
  data() {
    return {
      reconnecting: false,
      onlineHandler: null
    }
  },
  computed: {
    fromHint() {
      const q = this.$route.query.from
      if (!q) return ''
      try {
        return decodeURIComponent(String(q))
      } catch (e) {
        return String(q)
      }
    }
  },
  mounted() {
    this.onlineHandler = () => {
      this.$message.success('网络已恢复，可点击「重新连接」加载页面')
    }
    if (typeof window !== 'undefined') {
      window.addEventListener('online', this.onlineHandler)
    }
  },
  beforeDestroy() {
    if (typeof window !== 'undefined' && this.onlineHandler) {
      window.removeEventListener('online', this.onlineHandler)
    }
  },
  methods: {
    async retry() {
      this.reconnecting = true
      try {
        await request({
          url: '/api/v1/app-config',
          method: 'get',
          silent: true,
          timeout: 8000
        })
        const raw = this.$route.query.from
        let target = '/home/index'
        if (raw) {
          try {
            target = decodeURIComponent(String(raw))
          } catch (e) {
            target = String(raw)
          }
        }
        if (target && target !== this.$route.fullPath) {
          await this.$router.replace(target)
        } else {
          await this.$router.replace('/home/index')
        }
      } catch (e) {
        this.$message.error('仍无法连接服务器，请稍后再试')
      } finally {
        this.reconnecting = false
      }
    },
    goHome() {
      this.$router.replace('/home/index')
    }
  }
}
</script>

<style lang="scss" scoped>
.offline-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(160deg, #f5f7fa 0%, #e8ecf1 100%);
  padding: 24px;
  box-sizing: border-box;
}

.offline-card {
  max-width: 440px;
  width: 100%;
  padding: 40px 36px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.08);
  text-align: center;
}

.offline-icon-wrap {
  margin-bottom: 20px;
}

.offline-icon {
  font-size: 64px;
  color: #e6a23c;
}

.offline-title {
  margin: 0 0 12px;
  font-size: 22px;
  font-weight: 600;
  color: #303133;
}

.offline-desc {
  margin: 0 0 16px;
  font-size: 14px;
  line-height: 1.6;
  color: #606266;
}

.offline-from {
  margin: 0 0 24px;
  font-size: 13px;
  color: #909399;
  word-break: break-all;

  code {
    font-size: 12px;
    background: #f4f4f5;
    padding: 2px 6px;
    border-radius: 4px;
  }
}

.offline-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: center;
  margin-bottom: 16px;
}

.offline-tip {
  margin: 0;
  font-size: 12px;
  color: #c0c4cc;
  line-height: 1.5;
}
</style>
