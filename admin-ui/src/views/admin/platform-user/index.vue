<template>
  <BasicLayout>
    <template #wrapper>
      <el-card class="box-card">
        <el-form ref="queryForm" :model="queryParams" :inline="true" label-width="72px">
          <el-form-item label="用户名" prop="username">
            <el-input
              v-model="queryParams.username"
              placeholder="模糊搜索"
              clearable
              size="small"
              style="width: 160px"
              @keyup.enter.native="handleQuery"
            />
          </el-form-item>
          <el-form-item label="手机号" prop="mobile">
            <el-input
              v-model="queryParams.mobile"
              placeholder="模糊搜索"
              clearable
              size="small"
              style="width: 160px"
              @keyup.enter.native="handleQuery"
            />
          </el-form-item>
          <el-form-item label="邮箱" prop="email">
            <el-input
              v-model="queryParams.email"
              placeholder="模糊搜索"
              clearable
              size="small"
              style="width: 180px"
              @keyup.enter.native="handleQuery"
            />
          </el-form-item>
          <el-form-item label="昵称" prop="nickname">
            <el-input
              v-model="queryParams.nickname"
              placeholder="模糊搜索"
              clearable
              size="small"
              style="width: 140px"
              @keyup.enter.native="handleQuery"
            />
          </el-form-item>
          <el-form-item label="姓名" prop="real_name">
            <el-input
              v-model="queryParams.real_name"
              placeholder="模糊搜索"
              clearable
              size="small"
              style="width: 120px"
              @keyup.enter.native="handleQuery"
            />
          </el-form-item>
          <el-form-item label="性别" prop="gender">
            <el-select v-model="queryParams.gender" placeholder="全部" clearable size="small" style="width: 100px">
              <el-option label="未知" :value="0" />
              <el-option label="男" :value="1" />
              <el-option label="女" :value="2" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态" prop="status">
            <el-select v-model="queryParams.status" placeholder="全部" clearable size="small" style="width: 110px">
              <el-option label="禁用" :value="0" />
              <el-option label="正常" :value="1" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" icon="el-icon-search" size="mini" @click="handleQuery">搜索</el-button>
            <el-button icon="el-icon-refresh" size="mini" @click="resetQuery">重置</el-button>
          </el-form-item>
        </el-form>

        <el-row :gutter="10" class="mb8">
          <el-col :span="1.5">
            <el-button type="primary" icon="el-icon-plus" size="mini" @click="handleAdd">新增</el-button>
          </el-col>
          <el-col :span="1.5">
            <el-button type="success" icon="el-icon-download" size="mini" @click="handleExport">导出</el-button>
          </el-col>
          <el-col :span="1.5">
            <el-button type="warning" icon="el-icon-upload2" size="mini" @click="triggerImport">导入</el-button>
          </el-col>
          <input
            ref="importFile"
            type="file"
            accept=".csv,text/csv"
            style="display: none"
            @change="onImportFile"
          >
        </el-row>

        <el-table v-loading="loading" :data="userList" border>
          <el-table-column label="ID" align="center" prop="user_id" width="80" fixed />
          <el-table-column label="姓名" align="center" prop="real_name" min-width="100" show-overflow-tooltip>
            <template slot-scope="scope">
              <span>{{ scope.row.real_name || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="昵称" align="center" prop="nickname" min-width="100" show-overflow-tooltip>
            <template slot-scope="scope">
              <span>{{ scope.row.nickname || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="头像" align="center" width="72">
            <template slot-scope="scope">
              <el-image
                v-if="avatarUrl(scope.row.avatar)"
                :src="avatarUrl(scope.row.avatar)"
                style="width: 34px; height: 34px; border-radius: 50%"
                fit="cover"
                :preview-src-list="[avatarUrl(scope.row.avatar)]"
              />
              <span v-else style="color: #c0c4cc">-</span>
            </template>
          </el-table-column>
          <el-table-column label="手机号" align="center" prop="mobile" min-width="120" show-overflow-tooltip />
          <el-table-column label="性别" align="center" width="72">
            <template slot-scope="scope">
              <span>{{ genderLabel(scope.row.gender) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" align="center" width="88">
            <template slot-scope="scope">
              <el-switch
                v-model="scope.row.status"
                :active-value="1"
                :inactive-value="0"
                @change="(val) => handleStatusChange(scope.row, val)"
              />
            </template>
          </el-table-column>
          <el-table-column label="所属角色" align="center" prop="role_names" min-width="140" show-overflow-tooltip />
          <el-table-column label="创建时间" align="center" width="168">
            <template slot-scope="scope">
              <span>{{ parseTime(scope.row.created_at) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" align="center" width="220" fixed="right" class-name="small-padding fixed-width">
            <template slot-scope="scope">
              <el-button size="mini" type="text" icon="el-icon-edit" @click="handleEdit(scope.row)">编辑</el-button>
              <el-button size="mini" type="text" icon="el-icon-key" @click="handleResetPwd(scope.row)">重置密码</el-button>
              <el-button size="mini" type="text" icon="el-icon-delete" @click="handleDelete(scope.row)">删除</el-button>
              <el-button size="mini" type="primary" icon="el-icon-link" @click="openBindDeviceDialog(scope.row)">绑定设备</el-button>
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

      <!-- 新增 -->
      <el-dialog title="新增平台用户" :visible.sync="openAdd" width="560px" append-to-body :close-on-click-modal="false">
        <el-form ref="formAdd" :model="formAdd" :rules="rulesAdd" label-width="96px">
          <el-form-item label="用户名" prop="username">
            <el-input v-model="formAdd.username" placeholder="必填，最多 64 字符" maxlength="64" />
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="formAdd.password" type="password" placeholder="至少 6 位" show-password />
          </el-form-item>
          <el-form-item label="姓名" prop="real_name">
            <el-input v-model="formAdd.real_name" maxlength="64" />
          </el-form-item>
          <el-form-item label="昵称" prop="nickname">
            <el-input v-model="formAdd.nickname" maxlength="100" />
          </el-form-item>
          <el-form-item label="手机号" prop="mobile">
            <el-input v-model="formAdd.mobile" maxlength="11" placeholder="可选，11 位" />
          </el-form-item>
          <el-form-item label="邮箱" prop="email">
            <el-input v-model="formAdd.email" />
          </el-form-item>
          <el-form-item label="头像" prop="avatar">
            <el-upload
              class="avatar-uploader"
              :show-file-list="false"
              :before-upload="beforeAvatarUpload"
              :http-request="uploadAvatarForAdd"
            >
              <img v-if="avatarUrl(formAdd.avatar)" :src="avatarUrl(formAdd.avatar)" class="avatar">
              <i v-else class="el-icon-plus avatar-uploader-icon" />
            </el-upload>
            <div class="hint">支持 jpg/png/webp/gif，≤ 5MB</div>
          </el-form-item>
          <el-form-item label="性别" prop="gender">
            <el-select v-model="formAdd.gender" placeholder="可选" clearable style="width: 100%">
              <el-option label="未知" :value="0" />
              <el-option label="男" :value="1" />
              <el-option label="女" :value="2" />
            </el-select>
          </el-form-item>
          <el-form-item label="生日" prop="birthday">
            <el-date-picker
              v-model="formAdd.birthday"
              type="date"
              value-format="yyyy-MM-dd"
              placeholder="可选"
              style="width: 100%"
            />
          </el-form-item>
          <el-form-item label="角色" prop="role_ids">
            <!-- 手动新增平台用户：后端强制仅落为「普通用户」(roles.slug = 'user')，前端 UI 同步锁定 -->
            <el-select v-model="formAdd.role_ids" multiple disabled placeholder="仅允许「普通用户」" style="width: 100%">
              <el-option v-for="item in normalRoleOptions" :key="item.roleId" :label="item.roleName" :value="item.roleId" />
            </el-select>
            <div class="hint">后台新增用户仅允许「普通用户」角色，信息保存到 users 表</div>
          </el-form-item>
        </el-form>
        <div slot="footer" class="dialog-footer">
          <el-button type="primary" @click="submitAdd">确 定</el-button>
          <el-button @click="openAdd = false">取 消</el-button>
        </div>
      </el-dialog>

      <!-- 编辑 -->
      <el-dialog title="编辑平台用户" :visible.sync="openEdit" width="560px" append-to-body :close-on-click-modal="false">
        <el-form ref="formEdit" :model="formEdit" :rules="rulesEdit" label-width="96px">
          <el-form-item label="用户名">
            <el-input v-model="formEdit.username" disabled />
          </el-form-item>
          <el-form-item label="姓名" prop="real_name">
            <el-input v-model="formEdit.real_name" maxlength="64" />
          </el-form-item>
          <el-form-item label="昵称" prop="nickname">
            <el-input v-model="formEdit.nickname" maxlength="100" />
          </el-form-item>
          <el-form-item label="手机号" prop="mobile">
            <el-input v-model="formEdit.mobile" maxlength="11" />
          </el-form-item>
          <el-form-item label="邮箱" prop="email">
            <el-input v-model="formEdit.email" />
          </el-form-item>
          <el-form-item label="头像" prop="avatar">
            <el-upload
              class="avatar-uploader"
              :show-file-list="false"
              :before-upload="beforeAvatarUpload"
              :http-request="uploadAvatarForEdit"
            >
              <img v-if="avatarUrl(formEdit.avatar)" :src="avatarUrl(formEdit.avatar)" class="avatar">
              <i v-else class="el-icon-plus avatar-uploader-icon" />
            </el-upload>
            <div class="hint">上传后会自动写入该用户头像</div>
          </el-form-item>
          <el-form-item label="性别" prop="gender">
            <el-select v-model="formEdit.gender" clearable placeholder="未知" style="width: 100%">
              <el-option label="未知" :value="0" />
              <el-option label="男" :value="1" />
              <el-option label="女" :value="2" />
            </el-select>
          </el-form-item>
          <el-form-item label="生日" prop="birthday">
            <el-date-picker v-model="formEdit.birthday" type="date" value-format="yyyy-MM-dd" style="width: 100%" clearable />
          </el-form-item>
          <el-form-item label="角色" prop="role_ids">
            <el-select v-model="formEdit.role_ids" multiple disabled placeholder="仅允许「普通用户」" style="width: 100%">
              <el-option v-for="item in normalRoleOptions" :key="item.roleId" :label="item.roleName" :value="item.roleId" />
            </el-select>
            <div class="el-form-item__tip" style="color:#909399;font-size:12px;line-height:1.4;margin-top:4px;">
              按规范平台用户仅能作为「普通用户」保存；如需管理员角色请到「管理员 / 角色权限」页面调整
            </div>
          </el-form-item>
        </el-form>
        <div slot="footer" class="dialog-footer">
          <el-button type="primary" @click="submitEdit">保 存</el-button>
          <el-button @click="openEdit = false">取 消</el-button>
        </div>
      </el-dialog>

      <!-- 重置密码 -->
      <el-dialog title="重置密码" :visible.sync="openPwd" width="420px" append-to-body>
        <el-form ref="formPwd" :model="formPwd" :rules="rulesPwd" label-width="88px">
          <el-form-item label="新密码" prop="password">
            <el-input v-model="formPwd.password" type="password" show-password placeholder="至少 6 位" />
          </el-form-item>
        </el-form>
        <div slot="footer" class="dialog-footer">
          <el-button type="primary" @click="submitPwd">确 定</el-button>
          <el-button @click="openPwd = false">取 消</el-button>
        </div>
      </el-dialog>

      <!-- 绑定设备弹窗 -->
      <el-dialog
        :visible.sync="bindDeviceDialogVisible"
        title="绑定设备"
        width="500px"
        append-to-body
      >
        <el-form label-width="80px">
          <el-form-item label="用户 ID">
            <span>{{ currentUserInfo.user_id }}</span>
          </el-form-item>
          <el-form-item label="用户名">
            <span>{{ currentUserInfo.username }}</span>
          </el-form-item>
          <el-form-item label="昵称">
            <span>{{ currentUserInfo.nickname }}</span>
          </el-form-item>
          <el-form-item label="设备 SN">
            <el-input
              v-model="bindDeviceForm.deviceSn"
              placeholder="请输入设备 SN"
              clearable
            />
          </el-form-item>
        </el-form>
        <div slot="footer" class="dialog-footer">
          <el-button @click="bindDeviceDialogVisible = false">取 消</el-button>
          <el-button type="primary" @click="doBindDevice">确认绑定</el-button>
        </div>
      </el-dialog>
    </template>
  </BasicLayout>
</template>

<script>
import {
  listPlatformUser,
  getPlatformUser,
  addPlatformUser,
  exportPlatformUser,
  importPlatformUser,
  updatePlatformUser,
  updatePlatformUserStatus,
  resetPlatformUserPassword,
  setPlatformUserRoles,
  delPlatformUser
} from '@/api/admin/platform-user'
import request from '@/utils/request'
import { resolveApiBaseURL } from '@/utils/env-api'
import { listRole } from '@/api/admin/sys-role'
import { createBind } from '@/api/admin/platform-user-device-bind'
import BasicLayout from '@/layout/BasicLayout'
import Pagination from '@/components/Pagination'

export default {
  name: 'PlatformUser',
  components: {
    BasicLayout,
    Pagination
  },
  data() {
    const mobileRule = (rule, value, callback) => {
      if (!value || String(value).trim() === '') {
        callback()
        return
      }
      if (!/^1[3-9]\d{9}$/.test(String(value).trim())) {
        callback(new Error('请输入 11 位手机号'))
      } else {
        callback()
      }
    }
    return {
      loading: true,
      total: 0,
      userList: [],
      queryParams: {
        page: 1,
        pageSize: 10,
        username: undefined,
        mobile: undefined,
        email: undefined,
        nickname: undefined,
        real_name: undefined,
        gender: undefined,
        status: undefined
      },
      openAdd: false,
      formAdd: {},
      rulesAdd: {
        username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
        password: [
          { required: true, message: '请输入密码', trigger: 'blur' },
          { min: 6, message: '至少 6 位', trigger: 'blur' }
        ],
        mobile: [{ validator: mobileRule, trigger: 'blur' }],
        email: [{ type: 'email', message: '邮箱格式不正确', trigger: 'blur' }]
      },
      openEdit: false,
      formEdit: {},
      rulesEdit: {
        mobile: [{ validator: mobileRule, trigger: 'blur' }],
        email: [{ type: 'email', message: '邮箱格式不正确', trigger: 'blur' }]
      },
      currentUserId: null,
      roleOptions: [],
      openPwd: false,
      formPwd: { password: '' },
      rulesPwd: {
        password: [
          { required: true, message: '请输入新密码', trigger: 'blur' },
          { min: 6, message: '至少 6 位', trigger: 'blur' }
        ]
      },
      // 绑定设备相关
      bindDeviceDialogVisible: false,
      bindDeviceForm: {
        userId: 0,
        deviceSn: ''
      },
      currentUserInfo: {}
    }
  },
  computed: {
    // 新增弹窗专用：仅展示「普通用户」角色，保证与后端强制落为 roles.slug = 'user' 一致
    normalRoleOptions() {
      return (this.roleOptions || []).filter(r => {
        const k = (r.roleKey || '').toLowerCase()
        return k === 'user'
      })
    }
  },
  created() {
    this.getList()
  },
  methods: {
    loadRoleOptions() {
      const order = ['super_admin', 'admin', 'operator', 'user', 'guest']
      return listRole({ pageIndex: 1, pageSize: 500 }).then((res) => {
        const list = ((res && res.data) || {}).list || []
        list.sort((a, b) => {
          const ka = (a.roleKey || '').toLowerCase()
          const kb = (b.roleKey || '').toLowerCase()
          const ia = order.indexOf(ka)
          const ib = order.indexOf(kb)
          return (ia === -1 ? 999 : ia) - (ib === -1 ? 999 : ib)
        })
        this.roleOptions = list
        return list
      })
    },
    getList() {
      this.loading = true
      const q = { ...this.queryParams }
      if (q.status === '' || q.status === undefined || q.status === null) delete q.status
      if (q.gender === '' || q.gender === undefined || q.gender === null) delete q.gender
      ;['username', 'mobile', 'email', 'nickname', 'real_name', 'userId', 'memberLevel', 'realNameStatus'].forEach((k) => {
        if (q[k] === '' || q[k] === undefined || q[k] === null) delete q[k]
      })
      listPlatformUser(q)
        .then((response) => {
          const d = (response && response.data) || {}
          this.userList = d.list || []
          this.total = d.total != null ? d.total : d.count || 0
          this.loading = false
        })
        .catch(() => {
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
        pageSize: 10,
        username: undefined,
        email: undefined,
        nickname: undefined,
        real_name: undefined,
        gender: undefined,
        status: undefined
      }
      this.getList()
    },
    handleExport() {
      const q = { ...this.queryParams }
      delete q.page
      delete q.pageSize
      if (q.status === '' || q.status === undefined || q.status === null) delete q.status
      if (q.gender === '' || q.gender === undefined || q.gender === null) delete q.gender
      ;['username', 'mobile', 'email', 'nickname', 'real_name', 'userId', 'memberLevel', 'realNameStatus'].forEach((k) => {
        if (q[k] === '' || q[k] === undefined || q[k] === null) delete q[k]
      })
      exportPlatformUser(q).catch(() => {})
    },
    genderLabel(g) {
      if (g === 1) return '男'
      if (g === 2) return '女'
      return '未知'
    },
    triggerImport() {
      if (this.$refs.importFile) this.$refs.importFile.click()
    },
    onImportFile(e) {
      const input = e.target
      const f = input.files && input.files[0]
      input.value = ''
      if (!f) return
      importPlatformUser(f)
        .then((res) => {
          const d = (res && res.data) || {}
          this.$message.success(`成功 ${d.success || 0} 条，失败 ${d.failed || 0} 条`)
          if (d.errors && d.errors.length) {
            this.$alert(d.errors.slice(0, 20).join('\n'), '失败明细', { type: 'warning' })
          }
          this.getList()
        })
        .catch(() => {})
    },
    avatarUrl(p) {
      const s = (p || '').trim()
      if (!s) return ''
      if (/^https?:\/\//i.test(s)) return s
      const base = resolveApiBaseURL()
      if (s.startsWith('/')) return base + s
      return base + '/' + s
    },
    beforeAvatarUpload(file) {
      const isAllowed = ['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(file.type)
      if (!isAllowed) {
        this.$message.error('仅支持 jpg/png/webp/gif')
        return false
      }
      const isLt5M = file.size / 1024 / 1024 <= 5
      if (!isLt5M) {
        this.$message.error('头像不能超过 5MB')
        return false
      }
      return true
    },
    uploadAvatarForAdd({ file, onSuccess, onError }) {
      const fd = new FormData()
      fd.append('type', '1')
      fd.append('file', file)
      request({
        url: '/api/v1/public/uploadFile',
        method: 'post',
        data: fd
      })
        .then((res) => {
          const d = (res && res.data) || {}
          this.formAdd.avatar = d.full_path || d.path || ''
          onSuccess && onSuccess(res)
        })
        .catch((err) => {
          onError && onError(err)
        })
    },
    uploadAvatarForEdit({ file, onSuccess, onError }) {
      const uid = this.currentUserId
      if (!uid) {
        this.$message.error('请先打开用户编辑弹窗')
        onError && onError(new Error('missing user id'))
        return
      }
      // 兼容 go-admin 现有上传接口：先上传文件拿到 URL，再走更新资料写入 users.avatar
      const fd = new FormData()
      fd.append('type', '1')
      fd.append('file', file)
      request({
        url: '/api/v1/public/uploadFile',
        method: 'post',
        data: fd
      })
        .then((res) => {
          const d = (res && res.data) || {}
          const avatar = d.full_path || d.path || ''
          if (!avatar) {
            throw new Error('upload response missing avatar path')
          }
          // 与「保存资料」一致：提交完整资料，避免后端把缺失字段当空值覆盖
          const body = {
            real_name: this.formEdit.real_name,
            nickname: this.formEdit.nickname,
            mobile: this.formEdit.mobile,
            email: this.formEdit.email,
            avatar,
            gender: this.formEdit.gender,
            birthday: this.formEdit.birthday || null
          }
          return updatePlatformUser(uid, body).then(() => {
            this.formEdit.avatar = avatar
            this.$message.success('头像已更新')
            this.getList()
            onSuccess && onSuccess(res)
          })
        })
        .catch((err) => {
          onError && onError(err)
        })
    },
    handleAdd() {
      this.formAdd = {
        username: '',
        password: '',
        real_name: '',
        nickname: '',
        mobile: '',
        email: '',
        avatar: '',
        gender: undefined,
        birthday: '',
        role_ids: []
      }
      this.openAdd = true
      // 加载角色选项；加载完再把「普通用户」回填到 formAdd.role_ids，UI 显示已选中
      listRole({ pageIndex: 1, pageSize: 500 }).then((res) => {
        const list = ((res && res.data) || {}).list || []
        const order = ['super_admin', 'admin', 'operator', 'user', 'guest']
        list.sort((a, b) => {
          const ka = (a.roleKey || '').toLowerCase()
          const kb = (b.roleKey || '').toLowerCase()
          const ia = order.indexOf(ka)
          const ib = order.indexOf(kb)
          return (ia === -1 ? 999 : ia) - (ib === -1 ? 999 : ib)
        })
        this.roleOptions = list
        const normal = list.find(r => (r.roleKey || '').toLowerCase() === 'user')
        this.$set(this.formAdd, 'role_ids', normal ? [normal.roleId] : [])
      }).catch(() => {
        this.roleOptions = []
      })
      this.$nextTick(() => {
        if (this.$refs.formAdd) this.$refs.formAdd.clearValidate()
      })
    },
    submitAdd() {
      this.$refs.formAdd.validate((valid) => {
        if (!valid) return
        const payload = { ...this.formAdd }
        if (payload.gender === undefined || payload.gender === null || payload.gender === '') {
          delete payload.gender
        }
        if (!payload.birthday) delete payload.birthday
        // 强制只传普通用户角色；若未加载到就交给后端兜底
        const normal = (this.roleOptions || []).find(r => (r.roleKey || '').toLowerCase() === 'user')
        if (normal) {
          payload.role_ids = [normal.roleId]
        } else {
          delete payload.role_ids
        }
        addPlatformUser(payload)
          .then(() => {
            this.$message.success('新增成功')
            this.openAdd = false
            this.getList()
          })
          .catch((err) => {
            // request.js 已把非 200 响应弹过 Message（含后端 msg），这里不再二次弹框，避免重复
            void err
          })
      })
    },
    handleStatusChange(row, val) {
      const prev = val === 1 ? 0 : 1
      updatePlatformUserStatus(row.user_id, val)
        .then(() => {
          this.$message.success('状态已更新')
        })
        .catch((err) => {
          row.status = prev
          this.$message.error('状态更新失败')
          console.error(err)
        })
    },
    handleEdit(row) {
      this.currentUserId = row.user_id
      getPlatformUser(row.user_id).then((res) => {
        const d = (res && res.data) || {}
        this.formEdit = {
          username: d.username || '',
          real_name: d.real_name || '',
          nickname: d.nickname || '',
          mobile: d.mobile || '',
          email: d.email || '',
          avatar: d.avatar || '',
          gender: d.gender !== undefined && d.gender !== null ? d.gender : 0,
          birthday: d.birthday || '',
          // 角色固定为「普通用户」，初始占位；loadRoleOptions 完成后回填真实 id
          role_ids: []
        }
        this.openEdit = true
        this.loadRoleOptions().then(() => {
          const normal = this.normalRoleOptions[0]
          this.$set(this.formEdit, 'role_ids', normal ? [normal.roleId] : [])
        })
        this.$nextTick(() => this.$refs.formEdit && this.$refs.formEdit.clearValidate())
      })
    },
    submitEdit() {
      this.$refs.formEdit.validate((valid) => {
        if (!valid) return
        const uid = this.currentUserId
        const body = {
          real_name: this.formEdit.real_name,
          nickname: this.formEdit.nickname,
          mobile: this.formEdit.mobile,
          email: this.formEdit.email,
          avatar: this.formEdit.avatar,
          gender: this.formEdit.gender,
          birthday: this.formEdit.birthday || ''
        }
        // 角色强制锁定为「普通用户」，忽略 UI 可能残留的其他角色 id
        const normal = this.normalRoleOptions[0]
        const roleIds = normal ? [normal.roleId] : []
        updatePlatformUser(uid, body)
          .then(() => setPlatformUserRoles(uid, roleIds))
          .then(() => {
            this.$message.success('已保存')
            this.openEdit = false
            this.getList()
          })
          .catch((err) => {
            // request.js 已经把后端 msg 弹过 Message，这里不再二次弹框
            void err
          })
      })
    },
    handleResetPwd(row) {
      this.currentUserId = row.user_id
      this.formPwd = { password: '' }
      this.openPwd = true
      this.$nextTick(() => this.$refs.formPwd && this.$refs.formPwd.clearValidate())
    },
    submitPwd() {
      this.$refs.formPwd.validate((valid) => {
        if (!valid) return
        resetPlatformUserPassword(this.currentUserId, this.formPwd.password).then(() => {
          this.$message.success('密码已重置')
          this.openPwd = false
        })
      })
    },
    handleDelete(row) {
      this.$confirm('确认删除该用户？（软删除，数据保留）', '提示', {
        type: 'warning'
      })
        .then(() => delPlatformUser(row.user_id))
        .then(() => {
          this.$message.success('已删除')
          this.getList()
        })
        .catch(() => {})
    },
    /** 打开绑定设备弹窗 */
    openBindDeviceDialog(row) {
      this.currentUserInfo = {
        user_id: row.user_id,
        username: row.username || '—',
        nickname: row.nickname || '—'
      }
      this.bindDeviceForm = {
        userId: row.user_id,
        deviceSn: ''
      }
      this.bindDeviceDialogVisible = true
    },
    /** 执行绑定设备 */
    async doBindDevice() {
      if (!this.bindDeviceForm.deviceSn) {
        this.$message.error('请输入设备 SN')
        return
      }
      try {
        const res = await createBind(this.bindDeviceForm)
        if (res.code === 200) {
          this.$message.success('绑定成功')
          this.bindDeviceDialogVisible = false
          this.getList()
        } else {
          this.$message.error(res.msg || '绑定失败')
        }
      } catch (error) {
        this.$message.error('绑定失败：' + (error.message || '未知错误'))
      }
    }
  }
}
</script>

<style scoped>
.avatar-uploader {
  display: inline-block;
}
.avatar-uploader .el-upload {
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
}
.avatar-uploader .el-upload:hover {
  border-color: #409eff;
}
.avatar-uploader-icon {
  font-size: 22px;
  color: #8c939d;
  width: 80px;
  height: 80px;
  line-height: 80px;
  text-align: center;
}
.avatar {
  width: 80px;
  height: 80px;
  display: block;
  border-radius: 6px;
  object-fit: cover;
}
.hint {
  margin-top: 6px;
  font-size: 12px;
  color: #909399;
}
</style>
