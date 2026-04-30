<template>
  <BasicLayout>
    <template #wrapper>
      <el-card class="box-card">
        <div class="toolbar-row">
          <div class="toolbar-left">
            <el-button type="primary" size="small" icon="el-icon-plus" @click="openCreate">新建产品</el-button>
            <el-button size="small" icon="el-icon-refresh" @click="getList">刷新</el-button>
          </div>
        </div>

        <el-form :inline="true" size="small" class="filter-form" @submit.native.prevent="handleQuery">
          <el-form-item label="关键词">
            <el-input v-model="query.keyword" clearable placeholder="名称 / product_key" style="width:200px" @keyup.enter.native="handleQuery" />
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="query.status" clearable placeholder="全部" style="width:120px">
              <el-option label="草稿" value="draft" />
              <el-option label="已发布" value="published" />
              <el-option label="禁用" value="disabled" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" icon="el-icon-search" @click="handleQuery">搜索</el-button>
          </el-form-item>
        </el-form>

        <el-table v-loading="loading" :data="list" border stripe style="width:100%">
          <el-table-column prop="name" label="产品名称" min-width="140" show-overflow-tooltip />
          <el-table-column prop="productKey" label="产品标识" min-width="140" show-overflow-tooltip />
          <el-table-column prop="category" label="分类" width="100" show-overflow-tooltip />
          <el-table-column prop="status" label="状态" width="96">
            <template slot-scope="{ row }">
              <el-tag :type="statusTag(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="firmwareCount" label="固件数" width="80" align="center" />
          <el-table-column prop="deviceCount" label="设备数" width="80" align="center" />
          <el-table-column label="可注册" width="88" align="center">
            <template slot-scope="{ row }">
              <el-tag :type="row.registrationReady ? 'success' : 'info'" size="small">
                {{ row.registrationReady ? '是' : '否' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="updatedAt" label="更新时间" width="160" />
          <el-table-column label="操作" width="280" fixed="right">
            <template slot-scope="{ row }">
              <el-button type="text" size="small" @click="openFirmware(row)">固件</el-button>
              <el-button v-if="row.status !== 'published'" type="text" size="small" @click="doPublish(row)">发布</el-button>
              <el-button v-if="row.status !== 'disabled'" type="text" size="small" @click="doDisable(row)">禁用</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination-wrap">
          <el-pagination
            background
            layout="total, sizes, prev, pager, next"
            :current-page.sync="query.page"
            :page-sizes="[10, 20, 50]"
            :page-size.sync="query.pageSize"
            :total="total"
            @size-change="getList"
            @current-change="getList"
          />
        </div>
      </el-card>

      <el-dialog title="新建产品" :visible.sync="createOpen" width="520px" append-to-body @close="resetCreateForm">
        <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="108px">
          <el-form-item label="产品标识" prop="productKey">
            <el-input v-model="createForm.productKey" placeholder="如 audiopro002，字母开头 2～64 位" />
          </el-form-item>
          <el-form-item label="产品名称" prop="name">
            <el-input v-model="createForm.name" placeholder="展示名称" />
          </el-form-item>
          <el-form-item label="分类">
            <el-input v-model="createForm.category" />
          </el-form-item>
          <el-form-item label="设备类型">
            <el-input v-model="createForm.deviceType" placeholder="可选" />
          </el-form-item>
          <el-form-item label="说明">
            <el-input v-model="createForm.description" type="textarea" :rows="2" />
          </el-form-item>
        </el-form>
        <span slot="footer" class="dialog-footer">
          <el-button @click="createOpen = false">取消</el-button>
          <el-button type="primary" :loading="createLoading" @click="submitCreate">确定</el-button>
        </span>
      </el-dialog>

      <el-dialog :title="'固件 · ' + (fwContext.productKey || '')" :visible.sync="fwOpen" width="640px" append-to-body @open="loadFirmwareList">
        <p class="hint">上传首包固件后，若版本为「已发布」且启用，即可在设备管理中添加该产品的设备。</p>
        <el-form inline size="small" class="fw-upload-form">
          <el-form-item label="版本号">
            <el-input v-model="fwUpload.version" placeholder="如 v1.0.0" style="width:160px" />
          </el-form-item>
          <el-form-item label="文件">
            <input ref="fwFile" type="file" accept=".bin,.zip" @change="onFwFile">
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="small" :loading="fwUploading" :disabled="!fwUpload.file" @click="submitFwUpload">上传</el-button>
          </el-form-item>
        </el-form>
        <el-table v-loading="fwLoading" :data="fwList" border size="small" max-height="280">
          <el-table-column prop="version" label="版本" width="120" />
          <el-table-column prop="publish_status" label="发布" width="72" align="center">
            <template slot-scope="{ row }">{{ row.publish_status === 2 ? '已发布' : '未发布' }}</template>
          </el-table-column>
          <el-table-column prop="status" label="启用" width="72" align="center">
            <template slot-scope="{ row }">{{ row.status === 1 ? '是' : '否' }}</template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" min-width="150" />
        </el-table>
        <span slot="footer" class="dialog-footer">
          <el-button @click="fwOpen = false">关闭</el-button>
        </span>
      </el-dialog>
    </template>
  </BasicLayout>
</template>

<script>
import BasicLayout from '@/layout/BasicLayout'
import {
  listIotProducts,
  createIotProduct,
  publishIotProduct,
  disableIotProduct,
  listFirmware,
  uploadFirmware
} from '@/api/admin/platform-product'

export default {
  name: 'PlatformProduct',
  components: { BasicLayout },
  data() {
    return {
      loading: false,
      list: [],
      total: 0,
      query: {
        page: 1,
        pageSize: 20,
        keyword: '',
        status: ''
      },
      createOpen: false,
      createLoading: false,
      createForm: {
        productKey: '',
        name: '',
        category: '',
        deviceType: '',
        description: ''
      },
      createRules: {
        productKey: [{ required: true, message: '必填', trigger: 'blur' }]
      },
      fwOpen: false,
      fwContext: { id: null, productKey: '' },
      fwList: [],
      fwLoading: false,
      fwUpload: { version: '', file: null },
      fwUploading: false
    }
  },
  created() {
    this.getList()
  },
  methods: {
    statusText(s) {
      const m = { draft: '草稿', published: '已发布', disabled: '禁用' }
      return m[s] || s
    },
    statusTag(s) {
      if (s === 'published') return 'success'
      if (s === 'disabled') return 'info'
      return 'warning'
    },
    handleQuery() {
      this.query.page = 1
      this.getList()
    },
    async getList() {
      this.loading = true
      try {
        const res = await listIotProducts({
          page: this.query.page,
          pageSize: this.query.pageSize,
          keyword: (this.query.keyword || '').trim(),
          status: (this.query.status || '').trim()
        })
        const data = (res && res.data) || {}
        this.list = data.list || []
        const cnt = data.count != null ? data.count : data.total
        this.total = Number(cnt) || 0
      } catch (e) {
        void e
      } finally {
        this.loading = false
      }
    },
    openCreate() {
      this.createOpen = true
    },
    resetCreateForm() {
      this.createForm = { productKey: '', name: '', category: '', deviceType: '', description: '' }
    },
    submitCreate() {
      this.$refs.createFormRef.validate(async (valid) => {
        if (!valid) return
        this.createLoading = true
        try {
          await createIotProduct({
            productKey: (this.createForm.productKey || '').trim(),
            name: (this.createForm.name || '').trim(),
            category: (this.createForm.category || '').trim(),
            deviceType: (this.createForm.deviceType || '').trim(),
            description: (this.createForm.description || '').trim(),
            status: 'draft'
          })
          this.$message.success('创建成功')
          this.createOpen = false
          this.getList()
        } catch (e) {
          void e
        } finally {
          this.createLoading = false
        }
      })
    },
    async doPublish(row) {
      try {
        await this.$confirm('确认发布该产品？', '提示', { type: 'warning' })
        await publishIotProduct(row.id)
        this.$message.success('已发布')
        this.getList()
      } catch (e) {
        if (e !== 'cancel') void e
      }
    },
    async doDisable(row) {
      try {
        await this.$confirm('禁用后无法再上传该产品的固件，确认？', '提示', { type: 'warning' })
        await disableIotProduct(row.id)
        this.$message.success('已禁用')
        this.getList()
      } catch (e) {
        if (e !== 'cancel') void e
      }
    },
    openFirmware(row) {
      this.fwContext = { id: row.id, productKey: row.productKey }
      this.fwUpload = { version: 'v1.0.0', file: null }
      this.fwList = []
      this.fwOpen = true
    },
    async loadFirmwareList() {
      const pk = this.fwContext.productKey
      if (!pk) return
      this.fwLoading = true
      try {
        const res = await listFirmware({
          page: 1,
          page_size: 50,
          product_key: pk
        })
        const data = (res && res.data) || {}
        this.fwList = data.list || []
      } catch (e) {
        void e
      } finally {
        this.fwLoading = false
      }
    },
    onFwFile(ev) {
      const f = ev.target.files && ev.target.files[0]
      this.fwUpload.file = f || null
    },
    async submitFwUpload() {
      const pk = this.fwContext.productKey
      const ver = (this.fwUpload.version || '').trim()
      if (!pk || !ver) {
        this.$message.warning('请填写版本号')
        return
      }
      if (!this.fwUpload.file) {
        this.$message.warning('请选择 .bin 或 .zip 文件')
        return
      }
      const fd = new FormData()
      fd.append('product_key', pk)
      fd.append('version', ver)
      fd.append('file', this.fwUpload.file)
      this.fwUploading = true
      try {
        await uploadFirmware(fd)
        this.$message.success('上传成功')
        this.fwUpload.file = null
        if (this.$refs.fwFile) this.$refs.fwFile.value = ''
        await this.loadFirmwareList()
        this.getList()
      } catch (e) {
        void e
      } finally {
        this.fwUploading = false
      }
    }
  }
}
</script>

<style scoped>
.toolbar-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 12px;
}
.filter-form {
  margin-bottom: 12px;
}
.pagination-wrap {
  margin-top: 16px;
  text-align: right;
}
.hint {
  font-size: 13px;
  color: #606266;
  margin-bottom: 12px;
}
.fw-upload-form {
  margin-bottom: 12px;
}
</style>
