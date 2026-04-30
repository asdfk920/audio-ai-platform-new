<template>
  <BasicLayout>
    <template #wrapper>
      <el-card class="box-card filter-card">
        <div class="toolbar-row">
          <div class="toolbar-left">
            <el-button type="primary" size="small" icon="el-icon-plus" @click="openAdd">新增绑定</el-button>
            <el-button size="small" icon="el-icon-delete" :disabled="!selectedRows.length" @click="batchUnbind">批量解绑</el-button>
          </div>
          <div class="toolbar-right">
            <el-button size="small" icon="el-icon-refresh" @click="refreshAll">刷新</el-button>
            <el-button
              v-if="isNarrow"
              type="text"
              size="small"
              @click="filterExpanded = !filterExpanded"
            >{{ filterExpanded ? '收起筛选' : '展开筛选' }}</el-button>
          </div>
        </div>

        <el-form
          :model="queryParams"
          label-width="92px"
          class="filter-form"
          @submit.native.prevent="handleQuery"
        >
          <el-row :gutter="12" :class="{ 'filter-collapsed': isNarrow && !filterExpanded }">
            <el-col :xs="24" :sm="12" :md="8">
              <el-form-item label="用户 ID">
                <el-input
                  v-model="queryParams.userId"
                  clearable
                  size="small"
                  placeholder="请输入用户 ID"
                  @keyup.enter.native="handleQuery"
                />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8">
              <el-form-item label="设备 SN">
                <el-input
                  v-model="queryParams.deviceSn"
                  clearable
                  size="small"
                  placeholder="请输入设备 SN"
                  @keyup.enter.native="handleQuery"
                />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8">
              <el-form-item label="绑定状态">
                <el-select v-model="queryParams.status" placeholder="全部" clearable size="small" style="width:100%">
                  <el-option label="全部" value="" />
                  <el-option label="绑定中" :value="1" />
                  <el-option label="已解绑" :value="2" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row>
            <el-col :span="24" style="text-align: right;">
              <el-button type="primary" size="small" icon="el-icon-search" @click="handleQuery">搜索</el-button>
              <el-button size="small" icon="el-icon-refresh" @click="resetQuery">重置</el-button>
            </el-col>
          </el-row>
        </el-form>

        <el-table
          v-loading="loading"
          :data="list"
          border
          class="table-list"
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="50" align="center" />
          <el-table-column label="ID" prop="id" width="70" sortable />
          <el-table-column label="用户 ID" prop="userId" width="90" />
          <el-table-column label="用户名" prop="username" min-width="120" />
          <el-table-column label="昵称" prop="nickname" min-width="100" />
          <el-table-column label="设备 SN" prop="deviceSn" min-width="140" />
          <el-table-column label="设备型号" prop="deviceModel" min-width="100" />
          <el-table-column label="绑定时间" prop="bindTime" width="160">
            <template slot-scope="scope">
              {{ scope.row.bindTime ? formatDateTime(scope.row.bindTime) : '-' }}
            </template>
          </el-table-column>
          <el-table-column label="状态" prop="bindStatus" width="90">
            <template slot-scope="scope">
              <el-tag v-if="scope.row.bindStatus === 1" type="success">绑定中</el-tag>
              <el-tag v-else type="info">已解绑</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="180" fixed="right">
            <template slot-scope="scope">
              <el-button
                v-if="scope.row.bindStatus === 1"
                type="text"
                size="small"
                icon="el-icon-delete"
                @click="handleUnbind(scope.row)"
              >解绑</el-button>
              <el-button
                type="text"
                size="small"
                icon="el-icon-view"
                @click="handleDetail(scope.row)"
              >详情</el-button>
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

      <!-- 新增绑定弹窗 -->
      <el-dialog :title="dialog.title" :visible.sync="dialogOpen" width="600px" @close="dialogClose">
        <el-form ref="form" :model="form" :rules="rules" label-width="100px">
          <el-form-item label="用户 ID" prop="userId">
            <el-input v-model="form.userId" placeholder="请输入用户 ID" disabled />
          </el-form-item>
          <el-form-item label="选择设备" prop="deviceSn">
            <el-select v-model="form.deviceSn" placeholder="请选择设备" filterable size="small" style="width:100%">
              <el-option
                v-for="item in availableDevices"
                :key="item.id"
                :label="item.sn + ' (' + item.model + ')'"
                :value="item.sn"
              >
                <span style="float: left">{{ item.sn }}</span>
                <span style="float: right; color: #8492a6; font-size: 12px">{{ item.model }}</span>
              </el-option>
            </el-select>
          </el-form-item>
        </el-form>
        <div slot="footer" class="dialog-footer">
          <el-button size="small" @click="dialogOpen = false">取 消</el-button>
          <el-button type="primary" size="small" :loading="submitLoading" @click="submitForm">确 定</el-button>
        </div>
      </el-dialog>

      <!-- 详情弹窗 -->
      <el-dialog title="绑定详情" :visible.sync="detailOpen" width="600px">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="绑定 ID">{{ detailData.id }}</el-descriptions-item>
          <el-descriptions-item label="用户 ID">{{ detailData.userId }}</el-descriptions-item>
          <el-descriptions-item label="用户名">{{ detailData.username }}</el-descriptions-item>
          <el-descriptions-item label="昵称">{{ detailData.nickname }}</el-descriptions-item>
          <el-descriptions-item label="设备 SN">{{ detailData.deviceSn }}</el-descriptions-item>
          <el-descriptions-item label="设备型号">{{ detailData.deviceModel }}</el-descriptions-item>
          <el-descriptions-item label="绑定时间">{{ detailData.bindTime ? formatDateTime(detailData.bindTime) : '-' }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag v-if="detailData.bindStatus === 1" type="success">绑定中</el-tag>
            <el-tag v-else type="info">已解绑</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="设备状态">
            <el-tag v-if="detailData.deviceStatus === 1" type="success">正常</el-tag>
            <el-tag v-else-if="detailData.deviceStatus === 2" type="danger">禁用</el-tag>
            <el-tag v-else type="warning">未激活</el-tag>
          </el-descriptions-item>
        </el-descriptions>
        <div slot="footer" class="dialog-footer">
          <el-button size="small" @click="detailOpen = false">关 闭</el-button>
        </div>
      </el-dialog>
    </template>
  </BasicLayout>
</template>

<script>
import { getBindList, createBind, unbind, batchUnbind, getAvailableDevices } from '@/api/admin/platform-user-device-bind'

export default {
  name: 'PlatformUserDeviceBind',
  data() {
    return {
      loading: true,
      submitLoading: false,
      list: [],
      selectedRows: [],
      total: 0,
      queryParams: {
        page: 1,
        pageSize: 20,
        userId: '',
        deviceSn: '',
        status: null
      },
      dialogOpen: false,
      dialog: {
        title: ''
      },
      form: {
        userId: 0,
        deviceSn: ''
      },
      rules: {
        userId: [{ required: true, message: '用户 ID 不能为空', trigger: 'blur' }],
        deviceSn: [{ required: true, message: '请选择设备', trigger: 'change' }]
      },
      availableDevices: [],
      detailOpen: false,
      detailData: {},
      filterExpanded: false,
      isNarrow: false
    }
  },
  created() {
    this.getList()
    this.checkWidth()
    window.addEventListener('resize', this.checkWidth)
  },
  beforeDestroy() {
    window.removeEventListener('resize', this.checkWidth)
  },
  methods: {
    formatDateTime(v) {
      if (!v) return '—'
      const s = String(v)
      return s.length > 19 ? s.slice(0, 19).replace('T', ' ') : s
    },
    checkWidth() {
      this.isNarrow = document.body.getBoundingClientRect().width < 768
    },
    getList() {
      this.loading = true
      getBindList(this.queryParams).then(res => {
        this.list = res.rows || res.data || []
        this.total = res.total || 0
        this.loading = false
      }).catch(() => {
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
        userId: '',
        deviceSn: '',
        status: null
      }
      this.getList()
    },
    refreshAll() {
      this.getList()
      this.$message.success('刷新成功')
    },
    handleSelectionChange(selection) {
      this.selectedRows = selection
    },
    openAdd(row) {
      this.form = {
        userId: 0,
        deviceSn: ''
      }
      this.availableDevices = []

      // 如果是从某行打开，则填充该用户 ID
      if (row && row.userId) {
        this.form.userId = row.userId
      } else {
        // 否则弹窗让用户输入
        this.$prompt('请输入用户 ID', '新增绑定', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          inputPattern: /^\d+$/,
          inputErrorMessage: '请输入有效的数字',
          beforeClose: async(action, instance, done) => {
            if (action === 'confirm') {
              const userId = parseInt(instance.inputValue)
              if (!userId) {
                this.$message.error('用户 ID 不能为空')
                return
              }
              this.form.userId = userId
              // 加载可绑定设备
              await this.loadAvailableDevices(userId)
              done()
            } else {
              done()
            }
          }
        }).then(() => {
          this.dialog.title = '新增绑定'
          this.dialogOpen = true
        }).catch(() => {})
        return
      }

      // 加载可绑定设备
      this.loadAvailableDevices(this.form.userId).then(() => {
        this.dialog.title = '新增绑定'
        this.dialogOpen = true
      })
    },
    loadAvailableDevices(userId) {
      return getAvailableDevices(userId).then(res => {
        this.availableDevices = res.data || []
        if (this.availableDevices.length === 0) {
          this.$message.warning('该用户没有可绑定的设备')
        }
      }).catch(() => {
        this.$message.error('加载设备列表失败')
      })
    },
    submitForm() {
      this.$refs.form.validate(valid => {
        if (!valid) return
        this.submitLoading = true
        createBind(this.form).then(res => {
          this.$message.success('绑定成功')
          this.submitLoading = false
          this.dialogOpen = false
          this.getList()
        }).catch(() => {
          this.submitLoading = false
        })
      })
    },
    dialogClose() {
      this.$refs.form && this.$refs.form.resetFields()
      this.form = { userId: 0, deviceSn: '' }
      this.availableDevices = []
    },
    handleUnbind(row) {
      this.$confirm('确认解除该设备与用户的绑定？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        unbind(row.id).then(res => {
          this.$message.success('解绑成功')
          this.getList()
        }).catch(() => {})
      }).catch(() => {})
    },
    batchUnbind() {
      if (this.selectedRows.length === 0) {
        this.$message.warning('请选择要解绑的记录')
        return
      }
      this.$confirm(`确认批量解绑选中的 ${this.selectedRows.length} 条记录？`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        const ids = this.selectedRows.map(item => item.id)
        batchUnbind(ids).then(res => {
          this.$message.success('批量解绑成功')
          this.getList()
        }).catch(() => {})
      }).catch(() => {})
    },
    handleDetail(row) {
      this.detailData = row
      this.detailOpen = true
    }
  }
}
</script>

<style scoped>
.stat-row {
  margin-bottom: 16px;
}
.stat-card {
  text-align: center;
}
.stat-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}
.stat-num {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
}
.stat-online .stat-num {
  color: #67C23A;
}
.stat-offline .stat-num {
  color: #909399;
}
.stat-unbound .stat-num {
  color: #E6A23C;
}
.filter-card {
  margin-bottom: 16px;
}
.toolbar-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.toolbar-left, .toolbar-right {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.filter-form {
  margin-bottom: 16px;
}
.filter-collapsed :deep(.el-form-item) {
  margin-bottom: 12px;
}
.table-list {
  margin-top: 16px;
}
</style>
