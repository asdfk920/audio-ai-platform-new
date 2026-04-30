<template>
  <div class="home-container">
    <el-row :gutter="12" class="row">
      <el-col :xs="24" :sm="12" :md="8">
        <el-card shadow="hover" class="card">
          <div class="card-title">快速入口</div>
          <div class="quick-actions">
            <el-button type="primary" size="mini" @click="go('/platform-user/index')">平台用户</el-button>
            <el-button type="info" size="mini" @click="go('/platform-admin/index')">管理员管理</el-button>
            <el-button type="warning" size="mini" @click="go('/platform-rbac/index')">权限管理</el-button>
          </div>
          <div class="hint">平台用户来自用户服务（C 端账号）；管理员为 go-admin 的 sys_user（后台账号）。建议先配置权限矩阵。</div>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="12" :md="8">
        <el-card shadow="hover" class="card">
          <div class="card-title">服务状态</div>
          <div class="status-line">
            <span class="label">go-admin</span>
            <el-tag :type="healthOk ? 'success' : 'danger'" size="mini">{{ healthOk ? 'OK' : 'FAIL' }}</el-tag>
          </div>
          <div class="hint">此处仅探测后端健康检查接口。</div>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="24" :md="8">
        <el-card shadow="hover" class="card">
          <div class="card-title">数据概览</div>
          <div class="hint">下方仪表盘会展示用户/内容/播放等趋势数据。</div>
        </el-card>
      </el-col>
    </el-row>

    <DashboardAdmin />
  </div>
</template>

<script>
import DashboardAdmin from '@/views/dashboard/admin/index'
import request from '@/utils/request'

export default {
  name: 'HomeIndex',
  components: { DashboardAdmin },
  data() {
    return {
      healthOk: false
    }
  },
  created() {
    this.probeHealth()
  },
  methods: {
    go(path) {
      this.$router.push(path)
    },
    probeHealth() {
      request({
        url: '/api/v1/health',
        method: 'get',
        silent: true
      })
        .then(() => {
          this.healthOk = true
        })
        .catch(() => {
          this.healthOk = false
        })
    }
  }
}
</script>

<style scoped>
.home-container {
  padding: 12px;
}
.row {
  margin-bottom: 12px;
}
.card {
  border-radius: 8px;
  margin-bottom: 12px;
}
.card-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 10px;
}
.quick-actions {
  display: flex;
  gap: 8px;
  margin-bottom: 10px;
}
.status-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}
.label {
  color: #303133;
  font-weight: 500;
}
.hint {
  color: #909399;
  font-size: 12px;
  line-height: 18px;
}
</style>

