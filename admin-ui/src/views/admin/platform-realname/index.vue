<template>
  <BasicLayout>
    <template #wrapper>
      <div class="realname-page">
        <el-row :gutter="16" class="stat-row">
          <el-col :xs="12" :sm="6">
            <el-card shadow="hover" class="stat-card stat-pending">
              <div class="stat-label">待审核</div>
              <div class="stat-num">{{ summary.pending_manual }}</div>
            </el-card>
          </el-col>
          <el-col :xs="12" :sm="6">
            <el-card shadow="hover" class="stat-card stat-pass">
              <div class="stat-label">已通过</div>
              <div class="stat-num">{{ summary.manual_pass }}</div>
            </el-card>
          </el-col>
          <el-col :xs="12" :sm="6">
            <el-card shadow="hover" class="stat-card stat-reject">
              <div class="stat-label">已驳回</div>
              <div class="stat-num">{{ summary.manual_reject }}</div>
            </el-card>
          </el-col>
          <el-col :xs="12" :sm="6">
            <el-card shadow="hover" class="stat-card stat-today">
              <div class="stat-label">今日申请</div>
              <div class="stat-num">{{ summary.today_submit }}</div>
            </el-card>
          </el-col>
        </el-row>

        <el-card shadow="never" class="filter-card">
          <el-alert
            title="合规提示"
            type="info"
            :closable="false"
            show-icon
            class="compliance-alert"
          >
            列表仅展示脱敏信息；证件影像仅供审核使用，请勿截屏外传。操作日志由系统留存，便于审计追溯。
          </el-alert>

          <el-form :model="queryParams" label-width="88px" class="filter-form">
            <el-row :gutter="12">
              <el-col :xs="24" :sm="12" :md="6">
                <el-form-item label="用户ID">
                  <el-input v-model="queryParams.user_id" placeholder="精确匹配" clearable size="small" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="12" :md="6">
                <el-form-item label="审核状态">
                  <el-select v-model="queryParams.auth_status" placeholder="全部" clearable size="small" style="width:100%">
                    <el-option label="全部" value="" />
                    <el-option label="待审核" :value="20" />
                    <el-option label="已通过" :value="21" />
                    <el-option label="已驳回" :value="22" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="12" :md="6">
                <el-form-item label="证件类型">
                  <el-select v-model="queryParams.cert_type" placeholder="全部" clearable size="small" style="width:100%">
                    <el-option label="全部" value="" />
                    <el-option label="个人身份证" :value="1" />
                    <el-option label="企业证照" :value="2" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="12" :md="6">
                <el-form-item label="渠道">
                  <el-select
                    v-model="queryParams.channel"
                    filterable
                    allow-create
                    default-first-option
                    placeholder="APP / 小程序 / H5"
                    clearable
                    size="small"
                    style="width:100%"
                  >
                    <el-option v-for="c in channelPresets" :key="c" :label="c" :value="c" />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="12" align="middle">
              <el-col :xs="24" :sm="12" :md="10">
                <el-form-item label="提交时间">
                  <el-date-picker
                    v-model="queryParams.dateRange"
                    type="daterange"
                    range-separator="至"
                    start-placeholder="开始"
                    end-placeholder="结束"
                    value-format="yyyy-MM-dd"
                    size="small"
                    style="width:100%"
                    clearable
                  />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="24" :md="14" class="filter-actions">
                <el-button type="primary" icon="el-icon-search" size="small" @click="handleQuery">搜索</el-button>
                <el-button icon="el-icon-refresh" size="small" @click="resetQuery">重置</el-button>
                <el-button
                  type="success"
                  plain
                  icon="el-icon-check"
                  size="small"
                  :disabled="!canBatchApprove"
                  @click="batchApprove"
                >批量通过</el-button>
                <el-button
                  type="danger"
                  plain
                  icon="el-icon-close"
                  size="small"
                  :disabled="!canBatchReject"
                  @click="batchReject"
                >批量驳回</el-button>
                <el-button icon="el-icon-download" size="small" @click="exportCsv">导出</el-button>
              </el-col>
            </el-row>
          </el-form>
        </el-card>

        <el-card shadow="never" class="table-card">
          <div slot="header" class="table-header">
            <span>实名申请列表</span>
          </div>
          <el-table
            v-loading="loading"
            :data="list"
            border
            stripe
            :row-class-name="tableRowClassName"
            @selection-change="handleSelectionChange"
          >
            <el-table-column type="selection" width="48" align="center" :selectable="rowSelectable" />
            <el-table-column label="流水ID" prop="id" width="88" align="center" />
            <el-table-column label="用户ID" prop="user_id" width="88" align="center" />
            <el-table-column label="证件类型" prop="cert_type" width="108" align="center">
              <template slot-scope="scope">
                <span>{{ certTypeLabel(scope.row.cert_type) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="姓名(脱敏)" prop="real_name_masked" min-width="120" show-overflow-tooltip />
            <el-table-column label="证件后4位" prop="id_number_last4" width="100" align="center" />
            <el-table-column label="三方流水号" prop="third_party_flow_no" min-width="130" show-overflow-tooltip />
            <el-table-column label="渠道" prop="third_party_channel" width="100" align="center" show-overflow-tooltip />
            <el-table-column label="审核状态" width="108" align="center">
              <template slot-scope="scope">
                <el-tag :type="authStatusTagType(scope.row.auth_status)" size="small" effect="plain">
                  <i :class="authStatusIcon(scope.row.auth_status)" /> {{ authStatusText(scope.row.auth_status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="提交时间" prop="created_at" width="168" align="center">
              <template slot-scope="scope">
                <span>{{ parseTime(scope.row.created_at) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="审核时间" width="168" align="center">
              <template slot-scope="scope">
                <span>{{ scope.row.reviewed_at ? parseTime(scope.row.reviewed_at) : '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="审核人" prop="reviewed_by" width="100" align="center" show-overflow-tooltip>
              <template slot-scope="scope">
                <span>{{ scope.row.reviewed_by || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="驳回原因" min-width="140" show-overflow-tooltip>
              <template slot-scope="scope">
                <span v-if="scope.row.auth_status === 22">{{ scope.row.fail_reason || '—' }}</span>
                <span v-else>—</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="220" align="center" fixed="right">
              <template slot-scope="scope">
                <template v-if="scope.row.auth_status === 20">
                  <el-button type="success" size="mini" plain @click="openReview(scope.row, 'approve')">通过</el-button>
                  <el-button type="danger" size="mini" plain @click="openReview(scope.row, 'reject')">驳回</el-button>
                  <el-button type="primary" size="mini" plain icon="el-icon-view" @click="openDetail(scope.row)">详情</el-button>
                </template>
                <template v-else>
                  <el-button type="primary" size="mini" plain icon="el-icon-view" @click="openDetail(scope.row)">查看详情</el-button>
                </template>
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
      </div>

      <el-dialog title="实名审核详情" :visible.sync="detailOpen" width="720px" append-to-body class="detail-dialog">
        <template v-if="detail">
          <el-descriptions :column="2" border size="small" title="用户基础信息">
            <el-descriptions-item label="用户ID">{{ detail.user_id }}</el-descriptions-item>
            <el-descriptions-item label="昵称">{{ detail.nickname || '—' }}</el-descriptions-item>
            <el-descriptions-item label="手机号">{{ detail.mobile_masked || '—' }}</el-descriptions-item>
            <el-descriptions-item label="注册时间">{{ detail.user_registered_at ? parseTime(detail.user_registered_at) : '—' }}</el-descriptions-item>
          </el-descriptions>

          <el-descriptions :column="2" border size="small" title="实名信息" style="margin-top:12px">
            <el-descriptions-item label="流水ID">{{ detail.id }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="authStatusTagType(detail.auth_status)" size="small">{{ authStatusText(detail.auth_status) }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="证件类型">{{ certTypeLabel(detail.cert_type) }}</el-descriptions-item>
            <el-descriptions-item label="姓名(脱敏)">{{ detail.real_name_masked }}</el-descriptions-item>
            <el-descriptions-item label="证件后4位">{{ detail.id_number_last4 }}</el-descriptions-item>
            <el-descriptions-item label="完整证件号">仅列表后四位可见，密文存库不展示</el-descriptions-item>
            <el-descriptions-item label="三方流水号">{{ detail.third_party_flow_no || '—' }}</el-descriptions-item>
            <el-descriptions-item label="渠道">{{ detail.third_party_channel || '—' }}</el-descriptions-item>
            <el-descriptions-item label="提交时间">{{ parseTime(detail.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="审核时间">{{ detail.reviewed_at ? parseTime(detail.reviewed_at) : '—' }}</el-descriptions-item>
            <el-descriptions-item label="审核人">{{ detail.reviewed_by || '—' }}</el-descriptions-item>
            <el-descriptions-item label="驳回原因" :span="2">{{ detail.fail_reason || '—' }}</el-descriptions-item>
          </el-descriptions>

          <div class="photo-block">
            <div class="photo-title">证件影像（含水印示意）</div>
            <el-row :gutter="12">
              <el-col :span="8">
                <div class="photo-wrap">
                  <span class="wm">审核专用</span>
                  <div class="photo-placeholder">{{ detail.id_card_front_ref || '人像面未上传' }}</div>
                </div>
              </el-col>
              <el-col :span="8">
                <div class="photo-wrap">
                  <span class="wm">审核专用</span>
                  <div class="photo-placeholder">{{ detail.id_card_back_ref || '国徽面未上传' }}</div>
                </div>
              </el-col>
              <el-col :span="8">
                <div class="photo-wrap">
                  <span class="wm">审核专用</span>
                  <div class="photo-placeholder">{{ detail.face_data_ref || '人脸未上传' }}</div>
                </div>
              </el-col>
            </el-row>
          </div>

          <el-alert title="操作日志" type="warning" :closable="false" show-icon style="margin-top:12px">
            系统已记录审核操作（操作人、时间、结果）。如需完整审计流水，请对接日志/审计平台查询。
          </el-alert>
        </template>
        <span slot="footer" class="dialog-footer">
          <el-button @click="detailOpen=false">关闭</el-button>
          <template v-if="detail && detail.auth_status === 20">
            <el-button type="danger" plain @click="openReviewFromDetail('reject')">驳回</el-button>
            <el-button type="success" @click="openReviewFromDetail('approve')">通过</el-button>
          </template>
        </span>
      </el-dialog>

      <el-dialog :title="reviewTitle" :visible.sync="reviewOpen" width="520px" append-to-body :close-on-click-modal="false">
        <el-form ref="reviewForm" :model="reviewForm" label-width="96px">
          <el-form-item label="流水ID">
            <el-input v-model="reviewForm.auth_record_id" disabled />
          </el-form-item>
          <el-form-item v-if="reviewForm.action==='reject'" label="驳回原因" required>
            <el-input v-model="reviewForm.reject_reason" type="textarea" :rows="3" placeholder="至少 4 个字，将展示给用户" />
          </el-form-item>
          <el-form-item label="备注">
            <el-input v-model="reviewForm.comment" type="textarea" :rows="2" placeholder="选填，写入审核备注" />
          </el-form-item>
        </el-form>
        <span slot="footer" class="dialog-footer">
          <el-button @click="reviewOpen=false">取消</el-button>
          <el-button type="primary" :loading="reviewLoading" @click="submitReview">确定</el-button>
        </span>
      </el-dialog>
    </template>
  </BasicLayout>
</template>

<script>
import {
  listPendingRealname,
  getRealnameDetail,
  reviewRealname,
  batchReviewRealname
} from '@/api/admin/platform-realname'

export default {
  name: 'PlatformRealName',
  data() {
    return {
      loading: false,
      list: [],
      total: 0,
      channelPresets: ['APP', '小程序', 'H5', 'Web'],
      summary: {
        pending_manual: 0,
        manual_pass: 0,
        manual_reject: 0,
        today_submit: 0
      },
      queryParams: {
        page: 1,
        pageSize: 20,
        user_id: '',
        auth_status: '',
        cert_type: '',
        channel: '',
        dateRange: null
      },
      selectedRows: [],
      detailOpen: false,
      detail: null,
      reviewOpen: false,
      reviewLoading: false,
      reviewForm: {
        auth_record_id: 0,
        action: 'approve',
        comment: '',
        reject_reason: ''
      }
    }
  },
  computed: {
    reviewTitle() {
      return this.reviewForm.action === 'approve' ? '通过实名审核' : '驳回实名审核'
    },
    canBatchApprove() {
      return this.selectedRows.length > 0 && this.selectedRows.every(r => r.auth_status === 20)
    },
    canBatchReject() {
      return this.canBatchApprove
    }
  },
  created() {
    this.getList()
  },
  methods: {
    certTypeLabel(ct) {
      if (ct === 1) return '个人身份证'
      if (ct === 2) return '企业证照'
      return String(ct ?? '-')
    },
    authStatusText(v) {
      switch (v) {
        case 20: return '待审核'
        case 21: return '已通过'
        case 22: return '已驳回'
        default: return String(v || '-')
      }
    },
    authStatusTagType(v) {
      switch (v) {
        case 20: return 'warning'
        case 21: return 'success'
        case 22: return 'danger'
        default: return 'info'
      }
    },
    authStatusIcon(v) {
      switch (v) {
        case 20: return 'el-icon-time'
        case 21: return 'el-icon-circle-check'
        case 22: return 'el-icon-circle-close'
        default: return 'el-icon-info'
      }
    },
    isOverdue(row) {
      if (row.auth_status !== 20 || !row.created_at) return false
      const t = new Date(row.created_at).getTime()
      return Date.now() - t > 24 * 60 * 60 * 1000
    },
    tableRowClassName({ row }) {
      if (this.isOverdue(row)) return 'realname-row-overdue'
      return ''
    },
    rowSelectable(row) {
      return row.auth_status === 20
    },
    handleSelectionChange(rows) {
      this.selectedRows = rows
    },
    buildQuery() {
      const q = {
        page: this.queryParams.page,
        page_size: this.queryParams.pageSize
      }
      const uid = String(this.queryParams.user_id || '').trim()
      if (uid) q.user_id = uid
      if (this.queryParams.auth_status !== '' && this.queryParams.auth_status != null) {
        q.auth_status = this.queryParams.auth_status
      }
      if (this.queryParams.cert_type !== '' && this.queryParams.cert_type != null) {
        q.cert_type = this.queryParams.cert_type
      }
      const ch = String(this.queryParams.channel || '').trim()
      if (ch) q.channel = ch
      if (this.queryParams.dateRange && this.queryParams.dateRange.length === 2) {
        q.created_from = this.queryParams.dateRange[0]
        q.created_to = this.queryParams.dateRange[1]
      }
      q.include_summary = 1
      return q
    },
    handleQuery() {
      this.queryParams.page = 1
      this.getList()
    },
    resetQuery() {
      this.queryParams = {
        page: 1,
        pageSize: 20,
        user_id: '',
        auth_status: '',
        cert_type: '',
        channel: '',
        dateRange: null
      }
      this.getList()
    },
    async getList() {
      this.loading = true
      try {
        const res = await listPendingRealname(this.buildQuery())
        const data = (res && res.data) || {}
        this.list = data.list || []
        const cnt = data.count != null ? data.count : data.total
        this.total = Number(cnt) || 0
        const s = data.summary
        if (s) {
          this.summary = {
            pending_manual: s.pending_manual || 0,
            manual_pass: s.manual_pass || 0,
            manual_reject: s.manual_reject || 0,
            today_submit: s.today_submit || 0
          }
        }
      } finally {
        this.loading = false
      }
    },
    async openDetail(row) {
      this.detailOpen = true
      this.detail = null
      const res = await getRealnameDetail(row.user_id)
      this.detail = res && res.data
    },
    openReview(row, action) {
      this.reviewOpen = true
      this.reviewForm = {
        user_id: row.user_id,
        auth_record_id: row.id,
        action,
        comment: '',
        reject_reason: ''
      }
    },
    openReviewFromDetail(action) {
      if (!this.detail) return
      this.detailOpen = false
      this.openReview({ id: this.detail.id, user_id: this.detail.user_id }, action)
    },
    async submitReview() {
      if (this.reviewForm.action === 'reject') {
        const rr = (this.reviewForm.reject_reason || '').trim()
        if (rr.length < 4) {
          this.$message.error('驳回原因至少 4 个字')
          return
        }
      }
      this.reviewLoading = true
      try {
        await reviewRealname(this.reviewForm)
        const ok = this.reviewForm.action === 'approve'
        this.$message.success(ok ? '审核通过' : '已驳回')
        this.reviewOpen = false
        this.getList()
      } finally {
        this.reviewLoading = false
      }
    },
    async batchApprove() {
      const items = this.selectedRows.map(r => ({ user_id: r.user_id }))
      if (!items.length) return
      try {
        await this.$confirm(`确认批量通过 ${items.length} 条待审核申请？`, '提示', { type: 'warning' })
      } catch (e) {
        return
      }
      try {
        await batchReviewRealname({ items, action: 'approve', comment: '' })
        this.$message.success('批量通过成功')
        this.getList()
      } catch (e) {
        /* request 已提示 */
      }
    },
    async batchReject() {
      const items = this.selectedRows.map(r => ({ user_id: r.user_id }))
      if (!items.length) return
      try {
        const { value } = await this.$prompt('请输入驳回原因（至少 4 字，将用于所有选中项）', '批量驳回', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          inputType: 'textarea',
          inputValidator: v => {
            if (!v || String(v).trim().length < 4) return '至少 4 个字'
            return true
          }
        })
        await batchReviewRealname({ items, action: 'reject', reject_reason: String(value).trim() })
        this.$message.success('批量驳回成功')
        this.getList()
      } catch (e) {
        if (e !== 'cancel') { /* noop */ }
      }
    },
    exportCsv() {
      const headers = ['流水ID', '用户ID', '证件类型', '姓名(脱敏)', '证件后4位', '三方流水号', '渠道', '审核状态', '提交时间', '审核时间', '审核人', '驳回原因']
      const rows = this.list.map(r => [
        r.id,
        r.user_id,
        this.certTypeLabel(r.cert_type),
        r.real_name_masked,
        r.id_number_last4,
        r.third_party_flow_no || '',
        r.third_party_channel || '',
        this.authStatusText(r.auth_status),
        this.parseTime(r.created_at),
        r.reviewed_at ? this.parseTime(r.reviewed_at) : '',
        r.reviewed_by || '',
        r.auth_status === 22 ? (r.fail_reason || '') : ''
      ])
      const esc = s => `"${String(s).replace(/"/g, '""')}"`
      const lines = [headers.map(esc).join(',')]
      rows.forEach(cols => lines.push(cols.map(esc).join(',')))
      const blob = new Blob(['\ufeff' + lines.join('\n')], { type: 'text/csv;charset=utf-8;' })
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = `实名审核_${new Date().toISOString().slice(0, 10)}.csv`
      a.click()
      URL.revokeObjectURL(a.href)
      this.$message.success('已导出当前页数据（CSV）')
    }
  }
}
</script>

<style scoped>
.realname-page {
  padding-bottom: 16px;
}
.stat-row {
  margin-bottom: 16px;
}
.stat-card {
  border-radius: 8px;
  margin-bottom: 8px;
}
.stat-label {
  font-size: 13px;
  color: #909399;
}
.stat-num {
  font-size: 22px;
  font-weight: 600;
  margin-top: 4px;
}
.stat-pending .stat-num { color: #e6a23c; }
.stat-pass .stat-num { color: #67c23a; }
.stat-reject .stat-num { color: #f56c6c; }
.stat-today .stat-num { color: #409eff; }
.filter-card,
.table-card {
  border-radius: 8px;
  margin-bottom: 16px;
}
.compliance-alert {
  margin-bottom: 12px;
  border-radius: 6px;
}
.filter-form {
  margin-top: 4px;
}
.filter-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding-top: 4px;
}
.table-header {
  font-weight: 600;
  font-size: 15px;
}
.photo-block {
  margin-top: 12px;
}
.photo-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
}
.photo-wrap {
  position: relative;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  min-height: 88px;
  background: #fafafa;
  overflow: hidden;
}
.photo-wrap .wm {
  position: absolute;
  right: 8px;
  bottom: 8px;
  font-size: 11px;
  color: rgba(64, 158, 255, 0.55);
  transform: rotate(-12deg);
  pointer-events: none;
  user-select: none;
}
.photo-placeholder {
  padding: 12px;
  font-size: 12px;
  color: #606266;
  word-break: break-all;
}
</style>

<style>
.el-table .realname-row-overdue > td {
  background-color: #fff7e8 !important;
}
</style>
