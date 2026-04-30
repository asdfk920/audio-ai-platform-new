<template>
  <BasicLayout>
    <template #wrapper>
      <el-card class="box-card">
        <!-- 搜索表单 -->
        <el-form ref="queryForm" :model="queryParams" :inline="true" label-width="80px">
          <el-form-item label="标题" prop="title">
            <el-input
              v-model="queryParams.title"
              placeholder="模糊搜索"
              clearable
              size="small"
              style="width: 180px"
              @keyup.enter.native="handleQuery"
            />
          </el-form-item>
          <el-form-item label="艺术家" prop="artist">
            <el-input
              v-model="queryParams.artist"
              placeholder="模糊搜索"
              clearable
              size="small"
              style="width: 160px"
              @keyup.enter.native="handleQuery"
            />
          </el-form-item>
          <el-form-item label="会员等级" prop="vip_level">
            <el-select v-model="queryParams.vip_level" placeholder="全部" clearable size="small" style="width: 120px">
              <el-option label="全部" :value="0" />
              <el-option label="标准" :value="1" />
              <el-option label="专业" :value="2" />
              <el-option label="金卡" :value="3" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态" prop="status">
            <el-select v-model="queryParams.status" placeholder="全部" clearable size="small" style="width: 120px">
              <el-option label="草稿" :value="0" />
              <el-option label="已上架" :value="1" />
              <el-option label="已下架" :value="2" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" icon="el-icon-search" size="mini" @click="handleQuery">搜索</el-button>
            <el-button icon="el-icon-refresh" size="mini" @click="resetQuery">重置</el-button>
          </el-form-item>
        </el-form>

        <!-- 操作按钮 -->
        <el-row :gutter="10" class="mb8">
          <el-col :span="1.5">
            <el-button type="primary" icon="el-icon-plus" size="mini" @click="handleAdd">新增</el-button>
          </el-col>
          <el-col :span="1.5">
            <el-button type="success" icon="el-icon-download" size="mini" @click="handleExport">导出</el-button>
          </el-col>
        </el-row>

        <!-- 表格 -->
        <el-table v-loading="loading" :data="contentList" border>
          <el-table-column label="ID" align="center" prop="id" width="80" fixed />
          <el-table-column label="封面" align="center" width="100">
            <template slot-scope="scope">
              <el-image
                v-if="scope.row.cover_url"
                :src="scope.row.cover_url"
                :preview-src-list="[scope.row.cover_url]"
                style="width: 60px; height: 60px"
                fit="cover"
              >
                <div slot="error" class="image-slot-muted">—</div>
              </el-image>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column label="标题" align="center" prop="title" min-width="200" show-overflow-tooltip />
          <el-table-column label="艺术家" align="center" prop="artist" min-width="120" show-overflow-tooltip />
          <el-table-column label="时长" align="center" prop="duration_sec" width="80">
            <template slot-scope="scope">
              {{ formatDuration(scope.row.duration_sec) }}
            </template>
          </el-table-column>
          <el-table-column label="会员等级" align="center" prop="vip_level" width="100">
            <template slot-scope="scope">
              <el-tag :type="getVipLevelType(scope.row.vip_level)">
                {{ getVipLevelName(scope.row.vip_level) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="音频有效期" align="center" min-width="200" show-overflow-tooltip>
            <template slot-scope="scope">
              {{ formatAudioValidity(scope.row) }}
            </template>
          </el-table-column>
          <el-table-column label="状态" align="center" prop="status" width="100">
            <template slot-scope="scope">
              <el-tag :type="getStatusType(scope.row.status)">
                {{ getStatusName(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" align="center" prop="created_at" width="160">
            <template slot-scope="scope">
              {{ parseTime(scope.row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" align="center" width="340" fixed="right">
            <template slot-scope="scope">
              <el-button
                type="text"
                size="small"
                @click="handleDetail(scope.row)"
              >
                详情
              </el-button>
              <el-button
                type="text"
                size="small"
                @click="handleUpdate(scope.row)"
              >
                编辑
              </el-button>
              <el-button
                v-if="scope.row.status !== 1"
                type="text"
                size="small"
                @click="handleOnline(scope.row)"
              >
                上架
              </el-button>
              <el-button
                v-if="scope.row.status === 1"
                type="text"
                size="small"
                @click="handleOffline(scope.row)"
              >
                下架
              </el-button>
              <el-button
                type="text"
                size="small"
                @click="handleDelete(scope.row)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 分页 -->
        <el-pagination
          :current-page="queryParams.page"
          :page-sizes="[10, 20, 50, 100]"
          :page-size="queryParams.page_size"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </el-card>

      <!-- 须在 #wrapper 内：BasicLayout 仅渲染具名插槽 wrapper，外部的 el-dialog 会被丢弃 -->
      <el-dialog :title="dialogTitle" :visible.sync="dialogVisible" width="720px" append-to-body :close-on-click-modal="false" custom-class="content-dialog">
        <el-form ref="form" :model="form" :rules="rules" label-width="100px">
          <el-row :gutter="12">
            <el-col :span="24">
              <el-form-item label="标题" prop="title">
                <el-input v-model="form.title" placeholder="必填" clearable maxlength="500" show-word-limit />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="艺术家" prop="artist">
                <el-input v-model="form.artist" placeholder="必填" clearable maxlength="255" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="时长(秒)" prop="duration_sec">
                <el-input-number v-model="form.duration_sec" :min="1" :max="86400" :step="1" controls-position="right" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="会员等级" prop="vip_level">
                <el-select v-model="form.vip_level" placeholder="必填" style="width: 100%">
                  <el-option label="全部" :value="0" />
                  <el-option label="标准" :value="1" />
                  <el-option label="专业" :value="2" />
                  <el-option label="金卡" :value="3" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col v-if="form.id" :span="12">
              <el-form-item label="状态" prop="status">
                <el-select v-model="form.status" placeholder="请选择" style="width: 100%">
                  <el-option label="草稿" :value="0" />
                  <el-option label="已上架" :value="1" />
                  <el-option label="已下架" :value="2" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="音频有效期">
                <el-date-picker
                  v-model="form.audio_valid_range"
                  type="datetimerange"
                  range-separator="至"
                  start-placeholder="开始时间"
                  end-placeholder="结束时间"
                  value-format="yyyy-MM-dd HH:mm:ss"
                  format="yyyy-MM-dd HH:mm:ss"
                  style="width: 100%"
                  clearable
                />
                <div class="muted mt8">留空表示不限制；需同时选择起止时间</div>
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="封面">
                <div class="upload-row">
                  <el-upload
                    action="#"
                    :auto-upload="false"
                    :show-file-list="false"
                    accept=".jpg,.jpeg,.png,.webp"
                    :on-change="onCoverChange"
                  >
                    <el-button size="small" type="default">选择文件</el-button>
                  </el-upload>
                  <span class="muted ml8">或填写 URL</span>
                </div>
                <el-input v-model="form.cover_url" placeholder="封面 URL（与上传二选一）" clearable class="mt8" />
                <el-image
                  v-if="coverPreviewSrc"
                  :src="coverPreviewSrc"
                  :preview-src-list="[coverPreviewSrc]"
                  class="mt10 cover-preview"
                  fit="cover"
                />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="音频">
                <div class="upload-row">
                  <el-upload
                    action="#"
                    :auto-upload="false"
                    :show-file-list="false"
                    accept=".mp3,.wav,.flac,.aac,.ogg,.m4a"
                    :on-change="onAudioChange"
                    :disabled="audioUploading"
                  >
                    <el-button size="small" type="default" :loading="audioUploading">{{ audioUploading ? '加密上传中...' : '选择文件' }}</el-button>
                  </el-upload>
                  <span v-if="audioPickName" class="ml8 file-name">{{ audioPickName }}</span>
                  <el-tag v-if="audioUploadStatus" :type="audioUploadStatusType" size="mini" class="ml8">{{ audioUploadStatus }}</el-tag>
                </div>
                <div v-if="audioKey" class="mt8 audio-key-box">
                  <span class="muted">audio_key：</span>
                  <el-input v-model="audioKey" readonly size="mini" style="width: 240px">
                    <el-button slot="append" icon="el-icon-document-copy" @click="copyAudioKey">复制</el-button>
                  </el-input>
                  <span class="muted ml8">（请妥善保管，仅创建时可见）</span>
                </div>
                <el-input v-model="form.audio_url" placeholder="音频 URL（与上传二选一）" clearable class="mt8" />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-divider content-position="left">空间坐标</el-divider>
            </el-col>
            <el-col :span="8">
              <el-form-item label="X" prop="pos_x">
                <el-input-number v-model="form.pos_x" :step="0.1" :precision="2" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="Y" prop="pos_y">
                <el-input-number v-model="form.pos_y" :step="0.1" :precision="2" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="Z" prop="pos_z">
                <el-input-number v-model="form.pos_z" :step="0.1" :precision="2" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-divider content-position="left">角度</el-divider>
            </el-col>
            <el-col :span="8">
              <el-form-item label="Yaw" prop="yaw">
                <el-input-number v-model="form.yaw" :step="1" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="Pitch" prop="pitch">
                <el-input-number v-model="form.pitch" :step="1" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="Roll" prop="roll">
                <el-input-number v-model="form.roll" :step="1" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-divider content-position="left">渲染参数</el-divider>
            </el-col>
            <el-col :span="8">
              <el-form-item label="距离" prop="render_distance">
                <el-input-number v-model="form.render_distance" :min="0" :step="1" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="增益" prop="render_gain">
                <el-input-number v-model="form.render_gain" :min="0" :step="0.1" :precision="2" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="滤波" prop="render_filter">
                <el-input v-model="form.render_filter" placeholder="如 lowpass" maxlength="32" clearable />
              </el-form-item>
            </el-col>
            <el-col v-if="form.id" :span="24">
              <el-form-item label="歌词" prop="lyrics">
                <el-input v-model="form.lyrics" type="textarea" :rows="4" placeholder="歌词（可选）" clearable />
              </el-form-item>
            </el-col>
            <el-col v-if="form.id" :span="24">
              <el-form-item label="描述" prop="description">
                <el-input v-model="form.description" type="textarea" :rows="3" placeholder="描述（可选）" clearable />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
        <div slot="footer" class="dialog-footer">
          <el-button @click="cancel">取 消</el-button>
          <el-button type="primary" @click="submitForm">{{ form.id ? '确 定' : '确认创建' }}</el-button>
        </div>
      </el-dialog>

      <el-dialog title="内容详情" :visible.sync="detailVisible" width="600px" append-to-body>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="ID">{{ detailData.id }}</el-descriptions-item>
          <el-descriptions-item label="标题">{{ detailData.title }}</el-descriptions-item>
          <el-descriptions-item label="艺术家">{{ detailData.artist }}</el-descriptions-item>
          <el-descriptions-item label="时长">{{ formatDuration(detailData.duration_sec) }}</el-descriptions-item>
          <el-descriptions-item label="会员等级">
            <el-tag :type="getVipLevelType(detailData.vip_level)">
              {{ getVipLevelName(detailData.vip_level) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusType(detailData.status)">
              {{ getStatusName(detailData.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="音频 URL">
            <el-input v-model="detailData.audio_url" readonly size="mini">
              <el-button slot="append" icon="el-icon-document-copy" @click="copyText(detailData.audio_url)">复制</el-button>
            </el-input>
          </el-descriptions-item>
          <el-descriptions-item label="音频 Key">
            <el-input v-model="detailData.audio_key" readonly size="mini" placeholder="仅创建时可见">
              <el-button slot="append" icon="el-icon-document-copy" @click="copyText(detailData.audio_key)">复制</el-button>
            </el-input>
          </el-descriptions-item>
          <el-descriptions-item label="封面 URL">
            <el-input v-model="detailData.cover_url" readonly size="mini">
              <el-button slot="append" icon="el-icon-document-copy" @click="copyText(detailData.cover_url)">复制</el-button>
            </el-input>
          </el-descriptions-item>
          <el-descriptions-item label="私有格式下载">
            <div class="download-links">
              <el-button type="primary" size="small" icon="el-icon-download" @click="downloadPackage" :disabled="!detailData.audio_id && !detailData.cover_id">打包下载（音频+封面）</el-button>
              <div class="muted mt8">下载 .aasp 私有格式文件，需设备端密钥解密</div>
            </div>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ parseTime(detailData.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ parseTime(detailData.updated_at) }}</el-descriptions-item>
        </el-descriptions>
        <div slot="footer" class="dialog-footer">
          <el-button @click="detailVisible = false">关 闭</el-button>
        </div>
      </el-dialog>
    </template>
  </BasicLayout>
</template>

<script>
import {
  listContent,
  getContentDetail,
  addContent,
  updateContent,
  onlineContent,
  offlineContent,
  deleteContent,
  uploadFile
} from '@/api/admin/platform-content'

export default {
  name: 'PlatformContent',
  data() {
    return {
      loading: true,
      total: 0,
      contentList: [],
      dialogTitle: '',
      dialogVisible: false,
      detailVisible: false,
      detailData: {},
      queryParams: {
        page: 1,
        page_size: 10,
        title: undefined,
        artist: undefined,
        vip_level: undefined,
        status: undefined
      },
      form: {},
      coverPreviewUrl: '',
      audioPickName: '',
      audioUploading: false,
      audioUploadStatus: '',
      audioUploadStatusType: '',
      audioKey: '',
      audioFileId: 0,
      coverFileId: 0,
      rules: {
        title: [{ required: true, message: '标题不能为空', trigger: 'blur' }],
        artist: [{ required: true, message: '艺术家不能为空', trigger: 'blur' }],
        duration_sec: [{ required: true, message: '时长不能为空', trigger: 'blur' }],
        vip_level: [{ required: true, message: '会员等级不能为空', trigger: 'change' }]
      }
    }
  },
  computed: {
    coverPreviewSrc() {
      if (this.coverPreviewUrl) return this.coverPreviewUrl
      const u = this.form && this.form.cover_url
      return u && String(u).trim() ? u : ''
    }
  },
  created() {
    this.getList()
  },
  methods: {
    getList() {
      this.loading = true
      listContent(this.queryParams).then(response => {
        const d = (response && response.data) || {}
        this.contentList = d.list || []
        this.total = d.total != null ? d.total : 0
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
      this.$refs.queryForm.resetFields()
      this.handleQuery()
    },
    handleExport() {
      this.$message.info('导出功能开发中')
    },
    handleSizeChange(val) {
      this.queryParams.page_size = val
      this.getList()
    },
    handlePageChange(val) {
      this.queryParams.page = val
      this.getList()
    },
    handleAdd() {
      fetch('http://127.0.0.1:7774/ingest/97f27dd3-163b-4d88-a4e7-e2b55236be17', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Debug-Session-Id': '167bb8' },
        body: JSON.stringify({
          sessionId: '167bb8',
          hypothesisId: 'H5',
          location: 'platform-content/index.vue:handleAdd',
          message: 'dialog_open',
          data: {},
          timestamp: Date.now()
        })
      }).catch(() => {})
      this.reset()
      this.dialogTitle = '添加音频内容'
      this.dialogVisible = true
    },
    handleUpdate(row) {
      this.reset()
      const contentId = row.id
      getContentDetail(contentId).then(response => {
        const d = (response && response.data) || {}
        this.form = {
          id: d.id,
          title: d.title || '',
          artist: d.artist || '',
          duration_sec: d.duration_sec != null ? d.duration_sec : d.duration,
          vip_level: d.vip_level != null ? d.vip_level : 0,
          status: d.status != null ? d.status : 0,
          cover_url: d.cover_url || '',
          audio_url: d.audio_url || '',
          lyrics: d.lyrics || '',
          description: d.description || '',
          cover_file: undefined,
          audio_file: undefined
        }
        if (this.coverPreviewUrl) {
          URL.revokeObjectURL(this.coverPreviewUrl)
          this.coverPreviewUrl = ''
        }
        this.audioPickName = ''
        this.applySpatialFromDetail(d.spatial_params)
        if (d.audio_valid_from && d.audio_valid_until) {
          this.$set(this.form, 'audio_valid_range', [d.audio_valid_from, d.audio_valid_until])
        } else {
          this.$set(this.form, 'audio_valid_range', null)
        }
        this.dialogTitle = '编辑内容'
        this.dialogVisible = true
      })
    },
    handleDetail(row) {
      const contentId = row.id
      getContentDetail(contentId).then(response => {
        this.detailData = (response && response.data) || {}
        this.loadDownloadLinks(contentId)
        this.detailVisible = true
      })
    },
    loadDownloadLinks(contentId) {
      this.$set(this.detailData, 'package_download_url', `/api/admin/file/package/${contentId}`)
      this.$set(this.detailData, 'package_file_name', `content_${contentId}_package.aasp`)
    },
    getPackageDownloadUrl(contentId) {
      const request = (url, options) => fetch(url, options)
      return request({
        url: `/api/admin/file/package/${contentId}`,
        method: 'get'
      })
    },
    downloadPackage() {
      const contentId = this.detailData.id
      if (!contentId) {
        this.$message.warning('暂无内容ID')
        return
      }
      const url = `/api/admin/file/package/${contentId}`
      const fileName = this.detailData.package_file_name || `content_${contentId}_package.aasp`
      const a = document.createElement('a')
      a.href = url
      a.download = fileName
      a.click()
    },
    copyText(text) {
      if (!text) {
        this.$message.warning('暂无内容可复制')
        return
      }
      const input = document.createElement('input')
      input.value = text
      document.body.appendChild(input)
      input.select()
      document.execCommand('copy')
      document.body.removeChild(input)
      this.$message.success('已复制到剪贴板')
    },
    handleOnline(row) {
      const contentId = row.id
      this.$confirm('确认要上架该内容吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        return onlineContent(contentId)
      }).then(() => {
        this.$message.success('上架成功')
        this.getList()
      }).catch(() => {
        this.$message.info('已取消')
      })
    },
    handleOffline(row) {
      const contentId = row.id
      this.$confirm('确认要下架该内容吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        return offlineContent(contentId)
      }).then(() => {
        this.$message.success('下架成功')
        this.getList()
      }).catch(() => {
        this.$message.info('已取消')
      })
    },
    handleDelete(row) {
      const contentId = row.id
      this.$confirm('确认要删除该内容吗？', '警告', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        return deleteContent(contentId)
      }).then(() => {
        this.$message.success('删除成功')
        this.getList()
      }).catch(() => {
        this.$message.info('已取消')
      })
    },
    submitForm() {
      this.$refs.form.validate(valid => {
        fetch('http://127.0.0.1:7774/ingest/97f27dd3-163b-4d88-a4e7-e2b55236be17', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Debug-Session-Id': '167bb8' },
          body: JSON.stringify({
            sessionId: '167bb8',
            hypothesisId: 'H5',
            location: 'platform-content/index.vue:submitForm',
            message: 'validate_result',
            data: { valid: !!valid, hasId: !!this.form.id },
            timestamp: Date.now()
          })
        }).catch(() => {})
        if (valid) {
          const contentId = this.form.id
          if (!contentId) {
            const okCover = this.form.cover_file instanceof File || (this.form.cover_url && String(this.form.cover_url).trim())
            const okAudio = this.form.audio_file instanceof File || (this.form.audio_url && String(this.form.audio_url).trim())
            if (!okCover || !okAudio) {
              this.$message.error('请上传封面与音频，或填写对应 URL')
              return
            }
          }
          const payload = { ...this.form }
          if (this.audioKey) {
            payload.audio_key = this.audioKey
          }
          if (this.audioFileId) {
            payload.audio_id = this.audioFileId
          }
          if (this.coverFileId) {
            payload.cover_id = this.coverFileId
          }
          if (payload.audio_valid_range && payload.audio_valid_range.length === 2) {
            payload.audio_validity_mode = 'range'
            payload.audio_valid_from = payload.audio_valid_range[0]
            payload.audio_valid_until = payload.audio_valid_range[1]
          } else {
            payload.audio_validity_mode = 'none'
            payload.audio_valid_from = ''
            payload.audio_valid_until = ''
          }
          delete payload.audio_valid_range
          if (contentId) {
            updateContent(contentId, payload).then(() => {
              this.$message.success('修改成功')
              this.dialogVisible = false
              this.getList()
            }).catch(() => {
              // 失败原因由 @/utils/request 拦截器统一弹出（含后端 msg）
            })
          } else {
            addContent(payload).then(() => {
              this.$message.success('新增成功')
              this.dialogVisible = false
              this.getList()
            }).catch(() => {
              // 失败原因由 @/utils/request 拦截器统一弹出（含后端 msg）；勿再写死「新增失败」以免盖住具体错误
            })
          }
        }
      })
    },
    cancel() {
      this.dialogVisible = false
      this.reset()
    },
    reset() {
      if (this.coverPreviewUrl) {
        URL.revokeObjectURL(this.coverPreviewUrl)
      }
      this.coverPreviewUrl = ''
      this.audioPickName = ''
      this.audioUploading = false
      this.audioUploadStatus = ''
      this.audioUploadStatusType = ''
      this.audioKey = ''
      this.audioFileId = 0
      this.coverFileId = 0
      this.form = {
        id: undefined,
        title: '',
        artist: '',
        duration_sec: 180,
        vip_level: 0,
        status: 0,
        cover_url: '',
        audio_url: '',
        lyrics: '',
        description: '',
        cover_file: undefined,
        audio_file: undefined,
        pos_x: 1.5,
        pos_y: 0,
        pos_z: 2.0,
        yaw: 90,
        pitch: 0,
        roll: 0,
        render_distance: 10,
        render_gain: 1.0,
        render_filter: 'lowpass',
        audio_valid_range: null
      }
      if (this.$refs.form) {
        this.$refs.form.clearValidate()
      }
    },
    onCoverChange(file) {
      const raw = file && file.raw
      if (!raw) return
      this.form.cover_file = raw
      if (this.coverPreviewUrl) URL.revokeObjectURL(this.coverPreviewUrl)
      this.coverPreviewUrl = URL.createObjectURL(raw)
      this.form.cover_url = ''

      uploadFile(raw).then(res => {
        const d = (res && res.data) || {}
        this.form.cover_url = d.file_url || ''
        this.coverFileId = d.file_id || 0
      }).catch(err => {
        this.$message.error('封面文件加密上传失败: ' + (err.msg || err.message || '未知错误'))
      })
    },
    onAudioChange(file) {
      const raw = file && file.raw
      if (!raw) return
      this.audioUploading = true
      this.audioUploadStatus = '加密上传中...'
      this.audioUploadStatusType = 'warning'
      this.audioKey = ''

      uploadFile(raw).then(res => {
        const d = (res && res.data) || {}
        this.form.audio_url = d.file_url || ''
        this.audioKey = d.audio_key || ''
        this.audioFileId = d.file_id || 0
        this.audioUploadStatus = '已加密'
        this.audioUploadStatusType = 'success'
        this.audioPickName = raw.name + ' (已加密)'
        this.audioUploading = false
      }).catch(err => {
        this.audioUploadStatus = '上传失败'
        this.audioUploadStatusType = 'danger'
        this.audioUploading = false
        this.$message.error('文件加密上传失败: ' + (err.msg || err.message || '未知错误'))
      })
    },
    applySpatialFromDetail(sp) {
      let obj = sp
      if (typeof sp === 'string' && sp) {
        try {
          obj = JSON.parse(sp)
        } catch (e) {
          obj = null
        }
      }
      if (!obj || typeof obj !== 'object') return
      const p = obj.position || {}
      const o = obj.orientation || {}
      const r = obj.render || {}
      if (p.x != null) this.$set(this.form, 'pos_x', Number(p.x))
      if (p.y != null) this.$set(this.form, 'pos_y', Number(p.y))
      if (p.z != null) this.$set(this.form, 'pos_z', Number(p.z))
      if (o.yaw != null) this.$set(this.form, 'yaw', Number(o.yaw))
      if (o.pitch != null) this.$set(this.form, 'pitch', Number(o.pitch))
      if (o.roll != null) this.$set(this.form, 'roll', Number(o.roll))
      if (r.distance != null) this.$set(this.form, 'render_distance', Number(r.distance))
      if (r.gain != null) this.$set(this.form, 'render_gain', Number(r.gain))
      if (r.filter != null) this.$set(this.form, 'render_filter', String(r.filter))
    },
    formatAudioValidity(row) {
      const a = row && row.audio_valid_from
      const b = row && row.audio_valid_until
      if (!a && !b) return '不限'
      return `${a || '-'} 至 ${b || '-'}`
    },
    formatDuration(seconds) {
      if (!seconds) return '-'
      const m = Math.floor(seconds / 60)
      const s = seconds % 60
      return `${m}:${s.toString().padStart(2, '0')}`
    },
    getVipLevelName(level) {
      const map = {
        0: '免费',
        1: '普通',
        2: '银卡',
        3: '金卡'
      }
      return map[level] || '-'
    },
    getVipLevelType(level) {
      const map = {
        0: 'success',
        1: '',
        2: 'warning',
        3: 'danger'
      }
      return map[level] || ''
    },
    getStatusName(status) {
      const map = {
        0: '草稿',
        1: '已上架',
        2: '已下架'
      }
      return map[status] || '-'
    },
    getStatusType(status) {
      const map = {
        0: 'info',
        1: 'success',
        2: 'warning'
      }
      return map[status] || ''
    },
    copyAudioKey() {
      if (!this.audioKey) return
      const input = document.createElement('textarea')
      input.value = this.audioKey
      document.body.appendChild(input)
      input.select()
      document.execCommand('copy')
      document.body.removeChild(input)
      this.$message.success('audio_key 已复制到剪贴板')
    }
  }
}
</script>

<style scoped>
.upload-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}
.muted {
  color: #909399;
  font-size: 12px;
}
.file-name {
  font-size: 13px;
  color: #606266;
}
.cover-preview {
  width: 160px;
  height: 160px;
  border-radius: 4px;
  border: 1px solid #ebeef5;
}
.mt8 {
  margin-top: 8px;
}
.mt10 {
  margin-top: 10px;
}
.ml8 {
  margin-left: 8px;
}
.box-card {
  margin-bottom: 20px;
}

.mb8 {
  margin-bottom: 8px;
}

.el-pagination {
  margin-top: 20px;
  text-align: right;
}

.image-slot-muted {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  font-size: 12px;
  color: #c0c4cc;
  background: #f5f7fa;
}

.audio-key-box {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}
</style>

<style>
.content-dialog {
  z-index: 3000 !important;
}

.content-dialog .el-dialog__wrapper {
  z-index: 3000 !important;
}
</style>
