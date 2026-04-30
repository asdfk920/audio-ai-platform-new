<template>
  <div class="dashboard-editor-container">
    <el-row :gutter="12" class="stat-row">
      <el-col :sm="12" :xs="24" :md="8" :xl="8" :lg="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-label">总用户数</div>
          <div class="stat-value">{{ formatInt(stats.total_users) }}</div>
        </el-card>
      </el-col>
      <el-col :sm="12" :xs="24" :md="8" :xl="8" :lg="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-label">日活用户</div>
          <div class="stat-value">{{ formatInt(stats.active_users) }}</div>
          <div class="stat-hint">当日有登录记录的用户</div>
        </el-card>
      </el-col>
      <el-col :sm="12" :xs="24" :md="8" :xl="8" :lg="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-label">总音频内容</div>
          <div class="stat-value">{{ formatInt(stats.total_contents) }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12" class="stat-row">
      <el-col :sm="12" :xs="24" :md="8" :xl="8" :lg="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-label">内容处理成功率</div>
          <div class="stat-value accent">{{ processedPercent }}</div>
          <div class="stat-hint">processed_contents 中 status=completed 占比</div>
        </el-card>
      </el-col>
      <el-col :sm="12" :xs="24" :md="8" :xl="8" :lg="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-label">今日播放量</div>
          <div class="stat-value">{{ formatInt(stats.today_play_count) }}</div>
        </el-card>
      </el-col>
      <el-col :sm="12" :xs="24" :md="8" :xl="8" :lg="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-label">在线设备</div>
          <div class="stat-value">{{ formatInt(stats.online_devices) }}</div>
          <div class="stat-hint">devices.status = online</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card :bordered="false" class="trend-card" :body-style="{ padding: '0' }">
      <div v-loading="loading" class="salesCard">
        <el-tabs v-model="activeTab">
          <el-tab-pane label="用户注册趋势" name="reg">
            <el-row>
              <el-col :span="24">
                <bar :key="'reg-' + chartKey" :list="regBarData" title="近 14 日新注册用户" />
              </el-col>
            </el-row>
          </el-tab-pane>
          <el-tab-pane label="内容播放趋势" name="play">
            <el-row>
              <el-col :span="24">
                <bar :key="'play-' + chartKey" :list="playBarData" title="近 14 日播放次数" />
              </el-col>
            </el-row>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-card>
  </div>
</template>

<script>
import { getDashboardData } from '@/api/admin/dashboard'
import Bar from '@/components/Bar.vue'

const emptyStats = () => ({
  total_users: 0,
  active_users: 0,
  total_contents: 0,
  processed_rate: 0,
  today_play_count: 0,
  online_devices: 0,
  user_reg_trend: [],
  playback_trend: []
})

export default {
  name: 'DashboardAdmin',
  components: { Bar },
  data() {
    return {
      loading: false,
      stats: emptyStats(),
      activeTab: 'reg',
      chartKey: 0
    }
  },
  computed: {
    processedPercent() {
      const r = Number(this.stats.processed_rate)
      if (!Number.isFinite(r)) return '—'
      return (r * 100).toFixed(1) + '%'
    },
    regBarData() {
      return (this.stats.user_reg_trend || []).map((p) => ({
        x: p.date,
        y: Number(p.count) || 0
      }))
    },
    playBarData() {
      return (this.stats.playback_trend || []).map((p) => ({
        x: p.date,
        y: Number(p.count) || 0
      }))
    }
  },
  created() {
    this.fetchData()
  },
  methods: {
    formatInt(n) {
      if (n === null || n === undefined) return '0'
      return String(n)
    },
    fetchData() {
      this.loading = true
      getDashboardData()
        .then((res) => {
          const d = res && res.data ? res.data : {}
          this.stats = { ...emptyStats(), ...d }
          this.chartKey += 1
        })
        .catch(() => {
          this.stats = emptyStats()
        })
        .finally(() => {
          this.loading = false
        })
    }
  }
}
</script>

<style lang="scss" scoped>
.dashboard-editor-container {
  padding: 12px;
  background-color: rgb(240, 242, 245);
  min-height: 360px;
}

.stat-row {
  margin-bottom: 12px;
}

.stat-card {
  margin-bottom: 12px;
  border-radius: 8px;
}

.stat-label {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.45);
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
  line-height: 1.2;

  &.accent {
    color: #13c2c2;
  }
}

.stat-hint {
  margin-top: 8px;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.35);
}

.trend-card {
  border-radius: 8px;
}

.salesCard {
  background: #fff;
}

::v-deep .el-tabs__item {
  padding-left: 16px !important;
  height: 50px;
  line-height: 50px;
}
</style>
