<template>
  <BasicLayout>
    <template #wrapper>
      <el-card class="box-card filter-card">
        <el-form :model="queryParams" label-width="88px" class="filter-form" @submit.native.prevent="handleQuery">
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8">
              <el-form-item label="设备 SN">
                <div class="sn-row">
                  <el-input
                    v-model="queryParams.sn"
                    clearable
                    size="small"
                    placeholder="SN"
                    @keyup.enter.native="handleQuery"
                  />
                  <el-select v-model="queryParams.sn_mode" size="small" class="sn-mode">
                    <el-option label="精确" value="exact" />
                    <el-option label="模糊" value="fuzzy" />
                  </el-select>
                </div>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8">
              <el-form-item label="在线状态">
                <el-select v-model="queryParams.online_status" placeholder="全部" clearable size="small" style="width:100%">
                  <el-option label="全部" value="" />
                  <el-option label="在线" :value="1" />
                  <el-option label="离线" :value="0" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8">
              <el-form-item label="影子记录">
                <el-select v-model="queryParams.has_shadow" placeholder="全部" clearable size="small" style="width:100%">
                  <el-option label="全部" value="" />
                  <el-option label="已有影子" value="1" />
                  <el-option label="无影子" value="0" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item>
            <el-button type="primary" size="small" icon="el-icon-search" @click="handleQuery">查询</el-button>
            <el-button size="small" icon="el-icon-refresh" @click="resetQuery">重置</el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <el-card class="box-card table-card">
        <el-table v-loading="loading" :data="list" size="small" stripe>
          <el-table-column prop="sn" label="设备 SN" min-width="140" show-overflow-tooltip />
          <el-table-column label="在线" width="80" align="center">
            <template slot-scope="scope">
              <el-tag :type="scope.row.display_online === 1 ? 'success' : 'info'" size="mini">
                {{ scope.row.display_online === 1 ? '在线' : '离线' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="firmware_version" label="固件" width="100" show-overflow-tooltip />
          <el-table-column label="电量" width="72" align="center">
            <template slot-scope="scope">
              {{ scope.row.battery != null ? scope.row.battery + '%' : '—' }}
            </template>
          </el-table-column>
          <el-table-column prop="run_state" label="运行状态" width="100" show-overflow-tooltip />
          <el-table-column label="Reported" width="88" align="center">
            <template slot-scope="scope">
              <el-tag :type="scope.row.has_reported ? 'success' : 'info'" size="mini">
                {{ scope.row.has_reported ? '有' : '无' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="Desired" width="88" align="center">
            <template slot-scope="scope">
              <el-tag :type="scope.row.has_desired ? 'warning' : 'info'" size="mini">
                {{ scope.row.has_desired ? '有' : '无' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="最后上报" width="168" align="center">
            <template slot-scope="scope">
              {{ scope.row.last_report_time ? parseTime(scope.row.last_report_time) : '—' }}
            </template>
          </el-table-column>
          <el-table-column label="影子更新" width="168" align="center">
            <template slot-scope="scope">
              {{ scope.row.shadow_updated_at ? parseTime(scope.row.shadow_updated_at) : '—' }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" align="center" fixed="right">
            <template slot-scope="scope">
              <el-button type="primary" size="mini" plain @click="openDetail(scope.row)">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
        <pagination
          v-show="total > 0"
          :total="total"
          :page.sync="queryParams.page"
          :limit.sync="queryParams.pageSize"
          @pagination="getList"
        />
      </el-card>

      <el-dialog title="设备影子详情" :visible.sync="detailOpen" width="920px" append-to-body @open="onDetailOpen">
        <div v-loading="detailLoading">
          <template v-if="shadowDetail">
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item label="SN">{{ shadowDetail.sn }}</el-descriptions-item>
              <el-descriptions-item label="设备 ID">{{ shadowDetail.device_id }}</el-descriptions-item>
              <el-descriptions-item label="在线">
                <el-tag :type="shadowDetail.online ? 'success' : 'info'" size="mini">
                  {{ shadowDetail.online ? '在线' : '离线' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="Redis 快照">
                <el-tag :type="shadowDetail.redis_present ? 'success' : 'info'" size="mini">
                  {{ shadowDetail.redis_present ? '有' : '无' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="固件">{{ shadowDetail.firmware_version || '—' }}</el-descriptions-item>
              <el-descriptions-item label="电量">
                {{ shadowDetail.battery != null ? shadowDetail.battery + '%' : '—' }}
              </el-descriptions-item>
              <el-descriptions-item label="运行状态">{{ shadowDetail.power_status || '—' }}</el-descriptions-item>
              <el-descriptions-item label="音量">
                {{ shadowDetail.volume != null ? shadowDetail.volume : '—' }}
              </el-descriptions-item>
              <el-descriptions-item label="最后上报">
                {{ shadowDetail.last_report_time ? parseTime(shadowDetail.last_report_time) : '—' }}
              </el-descriptions-item>
              <el-descriptions-item label="最后在线">
                {{ shadowDetail.last_online_time ? parseTime(shadowDetail.last_online_time) : '—' }}
              </el-descriptions-item>
            </el-descriptions>

            <el-tabs v-model="detailTab" style="margin-top:12px">
              <el-tab-pane label="Reported（上报）" name="reported">
                <pre class="json-pre">{{ formatJson(shadowDetail.reported) }}</pre>
              </el-tab-pane>
              <el-tab-pane label="Desired（期望）" name="desired">
                <pre class="json-pre">{{ formatJson(shadowDetail.desired) }}</pre>
              </el-tab-pane>
              <el-tab-pane label="Delta（差异）" name="delta">
                <pre class="json-pre">{{ formatJson(shadowDetail.delta) }}</pre>
              </el-tab-pane>
            </el-tabs>
          </template>
          <div v-else-if="!detailLoading" class="empty-text">暂无影子数据</div>
        </div>
        <span slot="footer" class="dialog-footer">
          <el-button size="small" :loading="detailLoading" @click="refreshDetail">刷新</el-button>
          <el-button @click="detailOpen = false">关闭</el-button>
        </span>
      </el-dialog>
    </template>
  </BasicLayout>
</template>

<script>
import { listDeviceShadows, getDeviceShadow } from '@/api/admin/platform-device-shadow'
import { parseTime } from '@/utils'
import BasicLayout from '@/layout/BasicLayout'
import Pagination from '@/components/Pagination'

export default {
  name: 'PlatformDeviceShadow',
  components: { BasicLayout, Pagination },
  data() {
    return {
      loading: false,
      list: [],
      total: 0,
      queryParams: {
        page: 1,
        pageSize: 20,
        sn: '',
        sn_mode: 'fuzzy',
        online_status: '',
        has_shadow: ''
      },
      detailOpen: false,
      detailLoading: false,
      detailTab: 'reported',
      detailSn: '',
      shadowDetail: null
    }
  },
  created() {
    this.queryParams.page = 1
    this.getList()
  },
  activated() {
    this.getList()
  },
  methods: {
    parseTime,
    getList() {
      this.loading = true
      const params = {
        page: this.queryParams.page,
        pageSize: this.queryParams.pageSize,
        sn: this.queryParams.sn || undefined,
        sn_mode: this.queryParams.sn_mode,
        online_status: this.queryParams.online_status === '' ? undefined : this.queryParams.online_status,
        has_shadow: this.queryParams.has_shadow === '' ? undefined : this.queryParams.has_shadow
      }
      listDeviceShadows(params)
        .then((res) => {
          console.log('[ShadowList] 📥 API完整响应:', JSON.stringify(res, null, 2))
          // 与 platform-device 一致：兼容 { code, data: { list, count } } 或直接 { list, count }
          const body = res || {}
          const page =
            body.data && typeof body.data === 'object' && !Array.isArray(body.data)
              ? body.data
              : body
          console.log('[ShadowList] 🔍 解析后的page对象:', page)
          const rawList = page.list != null ? page.list : page.rows
          console.log('[ShadowList] 📋 rawList:', rawList, '类型:', typeof rawList, Array.isArray(rawList) ? '数组' : '非数组')
          this.list = Array.isArray(rawList) ? rawList : []
          const cnt = page.count != null ? page.count : page.total
          this.total = Number(cnt) || 0
          console.log('[ShadowList] ✅ 最终结果: list长度=', this.list.length, ', total=', this.total)
          if (this.list.length > 0) {
            console.log('[ShadowList] 📌 第一条数据:', this.list[0])
          }
          // 页码超出（如 keep-alive 残留 page=2）时 count 有值但 list 为空
          if (this.list.length === 0 && this.total > 0 && this.queryParams.page > 1) {
            this.queryParams.page = 1
            this.getList()
          }
        })
        .catch((e) => {
          // 非 200 时 request 拦截器已弹 msg，此处仅处理 404 等需额外说明的场景
          if (e && e.response && e.response.status === 404) {
            this.$message.error('接口未找到：请重启 go-admin 后端（需包含设备影子路由）')
          }
          this.list = []
          this.total = 0
        })
        .finally(() => {
          this.loading = false
        })
    },
    handleQuery() {
      this.queryParams.page = 1
      this.getList()
    },
    resetQuery() {
      this.queryParams = {
        page: 1,
        pageSize: 20,
        sn: '',
        sn_mode: 'fuzzy',
        online_status: '',
        has_shadow: ''
      }
      this.getList()
    },
    openDetail(row) {
      this.detailSn = row.sn
      this.detailTab = 'reported'
      this.shadowDetail = null
      this.detailOpen = true
    },
    onDetailOpen() {
      if (this.detailSn) {
        this.fetchDetail()
      }
    },
    refreshDetail() {
      this.fetchDetail()
    },
    fetchDetail() {
      if (!this.detailSn) return
      this.detailLoading = true
      getDeviceShadow(this.detailSn)
        .then((res) => {
          this.shadowDetail = (res && res.data) != null ? res.data : res
        })
        .catch((e) => {
          this.shadowDetail = null
          this.$message.error((e && e.message) || '加载影子详情失败')
        })
        .finally(() => {
          this.detailLoading = false
        })
    },
    formatJson(raw) {
      if (raw == null || raw === '') return '{}'
      try {
        const obj = typeof raw === 'string' ? JSON.parse(raw) : raw
        return JSON.stringify(obj, null, 2)
      } catch (e) {
        return String(raw)
      }
    }
  }
}
</script>

<style scoped>
.filter-card {
  margin-bottom: 12px;
}
.table-card {
  margin-bottom: 16px;
}
.sn-row {
  display: flex;
  gap: 8px;
}
.sn-mode {
  width: 88px;
  flex-shrink: 0;
}
.json-pre {
  font-size: 12px;
  margin: 0;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 360px;
  overflow: auto;
}
.empty-text {
  color: #909399;
  text-align: center;
  padding: 24px;
}
</style>
