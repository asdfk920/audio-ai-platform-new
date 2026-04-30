<template>
  <BasicLayout>
    <template #wrapper>
      <el-card class="box-card">
        <!-- 顶部搜索 -->
        <el-form ref="queryForm" :model="queryParams" :inline="true" label-width="72px">
          <el-form-item label="关键字" prop="keyword">
            <el-input
              v-model="queryParams.keyword"
              placeholder="用户名 / 昵称 / 姓名"
              clearable
              size="small"
              style="width: 220px"
              @keyup.enter.native="handleQuery"
            />
          </el-form-item>
          <el-form-item label="角色" prop="role_id">
            <el-select v-model="queryParams.role_id" placeholder="全部" clearable size="small" style="width: 160px">
              <el-option
                v-for="r in roleOptions"
                :key="r.roleId"
                :label="r.roleName"
                :value="r.roleId"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="状态" prop="status">
            <el-select v-model="queryParams.status" placeholder="全部" clearable size="small" style="width: 120px">
              <el-option label="禁用" value="1" />
              <el-option label="正常" value="2" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" icon="el-icon-search" size="mini" @click="handleQuery">搜索</el-button>
            <el-button icon="el-icon-refresh" size="mini" @click="resetQuery">重置</el-button>
          </el-form-item>
        </el-form>

        <!-- 操作工具栏 -->
        <el-row :gutter="10" class="mb8">
          <el-col :span="1.5">
            <el-button type="primary" icon="el-icon-plus" size="mini" @click="handleAdd">新增管理员</el-button>
          </el-col>
          <el-col :span="1.5">
            <el-button
              type="success"
              icon="el-icon-open"
              size="mini"
              :disabled="!selection.length"
              @click="handleBatchStatus('2')"
            >批量启用</el-button>
          </el-col>
          <el-col :span="1.5">
            <el-button
              type="warning"
              icon="el-icon-turn-off"
              size="mini"
              :disabled="!selection.length"
              @click="handleBatchStatus('1')"
            >批量禁用</el-button>
          </el-col>
          <el-col :span="1.5">
            <el-button
              type="danger"
              icon="el-icon-delete"
              size="mini"
              :disabled="!selection.length"
              @click="handleBatchDelete"
            >批量删除</el-button>
          </el-col>
          <el-col :span="1.5">
            <el-button icon="el-icon-key" size="mini" @click="handleSelfChangePwd">修改我的密码</el-button>
          </el-col>
        </el-row>

        <!-- 列表 -->
        <el-table
          v-loading="loading"
          :data="adminList"
          border
          @selection-change="onSelectionChange"
        >
          <el-table-column type="selection" width="48" :selectable="canBatchOperate" />
          <el-table-column label="ID" prop="admin_id" align="center" width="72" />
          <el-table-column label="头像" align="center" width="68">
            <template slot-scope="scope">
              <el-avatar :size="36" :src="scope.row.avatar || defaultAvatar" />
            </template>
          </el-table-column>
          <el-table-column label="账号信息" min-width="200">
            <template slot-scope="scope">
              <div class="admin-ident">
                <div class="admin-name">
                  <span>{{ scope.row.nickname || scope.row.real_name || scope.row.username }}</span>
                  <el-tag v-if="scope.row.is_super" size="mini" type="danger" effect="dark" class="ml6">超管</el-tag>
                  <el-tag
                    v-if="scope.row.must_change_password"
                    size="mini"
                    type="warning"
                    effect="plain"
                    class="ml6"
                  >下次登录需改密</el-tag>
                </div>
                <div class="admin-sub">@{{ scope.row.username }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="真实姓名" prop="real_name" min-width="110" show-overflow-tooltip>
            <template slot-scope="scope">
              <span>{{ scope.row.real_name || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="部门" prop="dept_name" min-width="120" show-overflow-tooltip>
            <template slot-scope="scope">
              <span>{{ scope.row.dept_name || '未分配' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="手机号" prop="phone" min-width="120">
            <template slot-scope="scope">
              <span>{{ scope.row.phone || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="邮箱" prop="email" min-width="170" show-overflow-tooltip>
            <template slot-scope="scope">
              <span>{{ scope.row.email || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="角色" min-width="150">
            <template slot-scope="scope">
              <el-tag
                v-for="r in scope.row.role_list"
                :key="r.role_id"
                size="mini"
                type="info"
                effect="plain"
                class="role-tag"
              >{{ r.role_name }}</el-tag>
              <span v-if="!(scope.row.role_list || []).length">-</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" prop="status" width="90" align="center">
            <template slot-scope="scope">
              <el-switch
                :value="scope.row.status === '2'"
                :disabled="scope.row.is_super"
                active-color="#13ce66"
                inactive-color="#ff4949"
                @change="(v) => handleStatusChange(scope.row, v ? '2' : '1')"
              />
            </template>
          </el-table-column>
          <el-table-column label="最后登录" min-width="160">
            <template slot-scope="scope">
              <div class="last-login">
                <div>{{ scope.row.last_login_time || '从未登录' }}</div>
                <div v-if="scope.row.last_login_ip" class="sub">{{ scope.row.last_login_ip }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" align="center" width="280" fixed="right">
            <template slot-scope="scope">
              <el-button type="text" size="mini" @click="handleEdit(scope.row)">编辑</el-button>
              <el-button
                type="text"
                size="mini"
                :disabled="scope.row.is_super"
                @click="handleResetPwd(scope.row)"
              >重置密码</el-button>
              <el-button
                type="text"
                size="mini"
                :disabled="scope.row.is_super"
                @click="handleSecurity(scope.row)"
              >安全策略</el-button>
              <el-button
                type="text"
                size="mini"
                :disabled="scope.row.is_super"
                @click="handleToggleForce(scope.row)"
              >{{ scope.row.must_change_password ? '取消强制改密' : '强制改密' }}</el-button>
              <el-button
                type="text"
                size="mini"
                style="color:#f56c6c"
                :disabled="scope.row.is_super"
                @click="handleDelete(scope.row)"
              >删除</el-button>
            </template>
          </el-table-column>
        </el-table>

        <el-pagination
          v-show="total > 0"
          class="mt12"
          background
          :current-page.sync="queryParams.pageIndex"
          :page-size.sync="queryParams.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="getList"
        />
      </el-card>

      <!-- 新增对话框 -->
      <el-dialog :visible.sync="openAdd" title="新增管理员" width="640px" :close-on-click-modal="false">
        <el-form ref="formAdd" :model="formAdd" :rules="rulesAdd" label-width="100px">
          <el-form-item label="用户名" prop="username">
            <el-input v-model="formAdd.username" placeholder="6-20 位字母/数字/下划线" maxlength="20" />
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input
              v-model="formAdd.password"
              type="password"
              show-password
              placeholder="8-20 位，包含大小写字母和数字"
              maxlength="20"
            />
          </el-form-item>
          <el-form-item label="昵称" prop="nickname">
            <el-input v-model="formAdd.nickname" maxlength="50" />
          </el-form-item>
          <el-form-item label="真实姓名" prop="real_name">
            <el-input v-model="formAdd.real_name" maxlength="64" />
          </el-form-item>
          <el-form-item label="部门" prop="dept_id">
            <treeselect
              v-model="formAdd.dept_id"
              :options="deptOptions"
              :normalizer="deptNormalizer"
              :show-count="true"
              placeholder="请选择部门（可选）"
              :clearable="true"
            />
          </el-form-item>
          <el-form-item label="角色" prop="role_ids">
            <el-select v-model="formAdd.role_ids" multiple filterable placeholder="请选择角色" style="width: 100%">
              <el-option
                v-for="r in adminAssignableRoles"
                :key="r.roleId"
                :label="r.roleName"
                :value="r.roleId"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="手机号" prop="phone">
            <el-input v-model="formAdd.phone" maxlength="11" />
          </el-form-item>
          <el-form-item label="邮箱" prop="email">
            <el-input v-model="formAdd.email" maxlength="100" />
          </el-form-item>
          <el-form-item label="首次登录改密">
            <el-switch v-model="formAdd.must_change_password" />
            <span class="hint ml8">启用后：新管理员首次登录必须修改密码</span>
          </el-form-item>
          <el-form-item label="备注" prop="remark">
            <el-input v-model="formAdd.remark" type="textarea" :rows="2" maxlength="255" />
          </el-form-item>
        </el-form>
        <div slot="footer">
          <el-button @click="openAdd = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submitAdd">确 定</el-button>
        </div>
      </el-dialog>

      <!-- 编辑对话框 -->
      <el-dialog :visible.sync="openEdit" title="编辑管理员" width="640px" :close-on-click-modal="false">
        <el-form ref="formEdit" :model="formEdit" :rules="rulesEdit" label-width="100px">
          <el-form-item label="用户名">
            <el-input :value="formEdit.username" disabled />
          </el-form-item>
          <el-form-item label="昵称" prop="nickname">
            <el-input v-model="formEdit.nickname" maxlength="50" />
          </el-form-item>
          <el-form-item label="真实姓名" prop="real_name">
            <el-input v-model="formEdit.real_name" maxlength="64" />
          </el-form-item>
          <el-form-item label="部门" prop="dept_id">
            <treeselect
              v-model="formEdit.dept_id"
              :options="deptOptions"
              :normalizer="deptNormalizer"
              :show-count="true"
              placeholder="请选择部门（可选）"
              :clearable="true"
            />
          </el-form-item>
          <el-form-item label="角色" prop="role_ids">
            <el-select v-model="formEdit.role_ids" multiple filterable placeholder="请选择角色" style="width: 100%">
              <el-option
                v-for="r in adminAssignableRoles"
                :key="r.roleId"
                :label="r.roleName"
                :value="r.roleId"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="手机号" prop="phone">
            <el-input v-model="formEdit.phone" maxlength="11" />
          </el-form-item>
          <el-form-item label="邮箱" prop="email">
            <el-input v-model="formEdit.email" maxlength="100" />
          </el-form-item>
          <el-form-item label="头像">
            <el-upload
              class="avatar-uploader"
              :show-file-list="false"
              :http-request="uploadEditAvatar"
              :before-upload="beforeAvatarUpload"
              action="#"
            >
              <img v-if="formEdit.avatar" :src="formEdit.avatar" class="avatar">
              <i v-else class="el-icon-plus avatar-uploader-icon" />
            </el-upload>
            <div class="hint">支持 jpg/png，建议 200*200，不超过 2MB</div>
          </el-form-item>
          <el-form-item label="备注" prop="remark">
            <el-input v-model="formEdit.remark" type="textarea" :rows="2" maxlength="255" />
          </el-form-item>
        </el-form>
        <div slot="footer">
          <el-button @click="openEdit = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submitEdit">确 定</el-button>
        </div>
      </el-dialog>

      <!-- 重置密码对话框 -->
      <el-dialog :visible.sync="openReset" title="重置密码" width="420px" :close-on-click-modal="false">
        <el-form ref="formReset" :model="formReset" :rules="rulesReset" label-width="110px">
          <el-form-item label="目标管理员">
            <span>{{ currentRow && currentRow.username }}</span>
          </el-form-item>
          <el-form-item label="新密码" prop="new_password">
            <el-input
              v-model="formReset.new_password"
              type="password"
              show-password
              placeholder="8-20 位，含大小写字母和数字"
              maxlength="20"
            />
          </el-form-item>
          <el-form-item label="强制下次改密">
            <el-switch v-model="formReset.require_change_on_login" />
            <span class="hint ml8">建议开启：本次设置后对方登录时必须立即重置</span>
          </el-form-item>
        </el-form>
        <div slot="footer">
          <el-button @click="openReset = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submitReset">确 定</el-button>
        </div>
      </el-dialog>

      <!-- 安全策略对话框 -->
      <el-dialog :visible.sync="openSecurity" title="安全策略" width="520px" :close-on-click-modal="false">
        <el-form ref="formSecurity" :model="formSecurity" label-width="110px">
          <el-form-item label="目标管理员">
            <span>{{ currentRow && currentRow.username }}</span>
          </el-form-item>
          <el-form-item label="IP 白名单">
            <el-input
              v-model="formSecurity.allowed_ips"
              type="textarea"
              :rows="3"
              placeholder="留空表示不限制；多个用英文逗号分隔，支持 CIDR，例如 10.0.0.1, 192.168.1.0/24"
            />
          </el-form-item>
          <el-form-item label="允许登录时间">
            <div class="time-row">
              <el-time-select
                v-model="formSecurity.allowed_login_start"
                placeholder="起点（可选）"
                :picker-options="{ start: '00:00', step: '00:30', end: '23:30' }"
                style="width: 160px"
                clearable
              />
              <span class="time-dash">~</span>
              <el-time-select
                v-model="formSecurity.allowed_login_end"
                placeholder="终点（可选）"
                :picker-options="{ start: '00:00', step: '00:30', end: '23:30' }"
                style="width: 160px"
                clearable
              />
            </div>
            <div class="hint">留空表示不限制；仅校验 HH:MM，允许跨天（起点 &gt; 终点）。</div>
          </el-form-item>
        </el-form>
        <div slot="footer">
          <el-button @click="openSecurity = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submitSecurity">保 存</el-button>
        </div>
      </el-dialog>

      <!-- 自助改密对话框 -->
      <el-dialog :visible.sync="openSelfPwd" title="修改我的密码" width="420px" :close-on-click-modal="false">
        <el-form ref="formSelfPwd" :model="formSelfPwd" :rules="rulesSelfPwd" label-width="100px">
          <el-form-item label="旧密码" prop="old_password">
            <el-input v-model="formSelfPwd.old_password" type="password" show-password maxlength="100" />
          </el-form-item>
          <el-form-item label="新密码" prop="new_password">
            <el-input
              v-model="formSelfPwd.new_password"
              type="password"
              show-password
              placeholder="8-20 位，含大小写字母和数字"
              maxlength="20"
            />
          </el-form-item>
          <el-form-item label="确认新密码" prop="confirm_password">
            <el-input v-model="formSelfPwd.confirm_password" type="password" show-password maxlength="20" />
          </el-form-item>
        </el-form>
        <div slot="footer">
          <el-button @click="openSelfPwd = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submitSelfPwd">确 定</el-button>
        </div>
      </el-dialog>
    </template>
  </BasicLayout>
</template>

<script>
import BasicLayout from '@/layout/BasicLayout'
import Treeselect from '@riophae/vue-treeselect'
import '@riophae/vue-treeselect/dist/vue-treeselect.css'

import request from '@/utils/request'
import {
  listSysAdmin,
  getSysAdmin,
  addSysAdmin,
  updateSysAdmin,
  delSysAdmin,
  batchDelSysAdmin,
  changeSysAdminStatus,
  resetSysAdminPassword,
  setSysAdminSecurity,
  setSysAdminMustChange,
  changeSelfPassword
} from '@/api/admin/sys-admin'
import { listRole } from '@/api/admin/sys-role'
import { treeselect as deptTree } from '@/api/admin/sys-dept'

// 后端密码策略：8-20 位，同时含大小写字母 + 数字
const PASSWORD_REGEX = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)[A-Za-z\d!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]{8,20}$/

// #region agent log
function agentDebugLog(hypothesisId, message, data) {
  try {
    fetch('http://127.0.0.1:7281/ingest/098c2c1f-e0e8-4ec4-a6e0-ba69d53941cd', {
      method: 'POST',
      headers: {'Content-Type': 'application/json', 'X-Debug-Session-Id': 'a35f07'},
      body: JSON.stringify({
        sessionId: 'a35f07',
        runId: 'sys-admin-ui',
        hypothesisId,
        location: 'admin-ui/src/views/admin/sys-admin/index.vue',
        message,
        data: data || {},
        timestamp: Date.now()
      })
    }).catch(() => {
    })
  } catch (e) {
    console.log('placeholder')
  }
}
// #endregion

export default {
  name: 'PlatformAdmin',
  components: { BasicLayout, Treeselect },
  data() {
    const validatePassword = (_, value, callback) => {
      if (!value) return callback(new Error('请输入密码'))
      if (!PASSWORD_REGEX.test(value)) {
        return callback(new Error('密码 8-20 位，且同时含大小写字母和数字'))
      }
      callback()
    }
    const validateConfirm = (_, value, callback) => {
      if (!value) return callback(new Error('请再次输入新密码'))
      if (value !== this.formSelfPwd.new_password) {
        return callback(new Error('两次输入的密码不一致'))
      }
      callback()
    }
    return {
      loading: false,
      submitting: false,
      defaultAvatar: 'https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png',

      queryParams: {
        pageIndex: 1,
        pageSize: 20,
        keyword: '',
        role_id: undefined,
        status: ''
      },
      adminList: [],
      total: 0,
      selection: [],

      roleOptions: [],
      deptOptions: [],

      openAdd: false,
      openEdit: false,
      openReset: false,
      openSecurity: false,
      openSelfPwd: false,

      currentRow: null,

      formAdd: this.buildEmptyAddForm(),
      formEdit: {
        user_id: 0,
        username: '',
        nickname: '',
        real_name: '',
        dept_id: null,
        role_ids: [],
        phone: '',
        email: '',
        avatar: '',
        remark: ''
      },
      formReset: { new_password: '', require_change_on_login: true },
      formSecurity: { allowed_ips: '', allowed_login_start: '', allowed_login_end: '' },
      formSelfPwd: { old_password: '', new_password: '', confirm_password: '' },

      rulesAdd: {
        username: [
          { required: true, message: '请输入用户名', trigger: 'blur' },
          { min: 6, max: 20, message: '长度在 6 到 20 个字符', trigger: 'blur' }
        ],
        password: [{ required: true, validator: validatePassword, trigger: 'blur' }],
        nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
        role_ids: [{ required: true, type: 'array', message: '请至少选择一个角色', trigger: 'change' }],
        email: [{ type: 'email', message: '邮箱格式不正确', trigger: 'blur' }],
        phone: [{ pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确', trigger: 'blur' }]
      },
      rulesEdit: {
        nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
        role_ids: [{ required: true, type: 'array', message: '请至少选择一个角色', trigger: 'change' }],
        email: [{ type: 'email', message: '邮箱格式不正确', trigger: 'blur' }],
        phone: [{ pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确', trigger: 'blur' }]
      },
      rulesReset: {
        new_password: [{ required: true, validator: validatePassword, trigger: 'blur' }]
      },
      rulesSelfPwd: {
        old_password: [{ required: true, message: '请输入旧密码', trigger: 'blur' }],
        new_password: [{ required: true, validator: validatePassword, trigger: 'blur' }],
        confirm_password: [{ required: true, validator: validateConfirm, trigger: 'blur' }]
      }
    }
  },
  computed: {
    // 前端侧仅把可见角色开放给添加/编辑；super_admin 由后端校验拦截，这里隐藏以降低误选。
    adminAssignableRoles() {
      return (this.roleOptions || []).filter(r => {
        const k = (r.roleKey || '').toLowerCase()
        return k !== 'super_admin' && k !== 'guest'
      })
    }
  },
  created() {
    this.loadRoles()
    this.loadDepts()
    this.getList()
  },
  methods: {
    buildEmptyAddForm() {
      return {
        username: '',
        password: '',
        nickname: '',
        real_name: '',
        dept_id: null,
        role_ids: [],
        phone: '',
        email: '',
        remark: '',
        must_change_password: true
      }
    },
    deptNormalizer(node) {
      // GET /api/v1/deptTree 返回 dto.DeptLabel：id / label / children（与 sys-user 一致）
      return {
        id: node.id,
        label: node.label,
        // vue-treeselect：叶子节点应省略 children 或设为 undefined；用 0 会产生内部 null 节点，触发
        // “Cannot read properties of null (reading 'id')”
        children: node.children && node.children.length ? node.children : undefined
      }
    },
    canBatchOperate(row) {
      return !row.is_super
    },
    loadRoles() {
      listRole({ pageIndex: 1, pageSize: 500 })
        .then((res) => {
          this.roleOptions = ((res && res.data) || {}).list || []
        })
        .catch(() => {
          this.roleOptions = []
        })
    },
    loadDepts() {
      deptTree()
        .then((res) => {
          this.deptOptions = (res && res.data) || []
          // #region agent log
          const flat = (nodes, acc = []) => {
            (nodes || []).forEach((n) => {
              acc.push({
                id: n.id,
                labelLen: n.label ? String(n.label).length : 0,
                labelCp0: n.label ? String(n.label).codePointAt(0) : null,
                isQuestionMarks: n.label ? /^[?]+$/.test(String(n.label)) : null
              })
              if (n.children && n.children.length) flat(n.children, acc)
            })
            return acc
          }
          const row100 = flat(this.deptOptions).find((x) => x.id === 100)
          agentDebugLog('H1', 'dept_tree_shape', {
            rootCount: (this.deptOptions || []).length,
            id100: row100 || null
          })
          // #endregion
        })
        .catch(() => {
          this.deptOptions = []
        })
    },
    getList() {
      this.loading = true
      const params = { ...this.queryParams }
      if (!params.status) delete params.status
      if (!params.role_id) delete params.role_id
      if (!params.keyword) delete params.keyword
      listSysAdmin(params)
        .then((res) => {
          const d = (res && res.data) || {}
          this.adminList = d.list || []
          this.total = d.count || 0
        })
        .catch(() => {
          this.adminList = []
          this.total = 0
        })
        .finally(() => {
          this.loading = false
        })
    },
    handleQuery() {
      this.queryParams.pageIndex = 1
      this.getList()
    },
    resetQuery() {
      this.queryParams = { pageIndex: 1, pageSize: this.queryParams.pageSize, keyword: '', role_id: undefined, status: '' }
      this.getList()
    },
    handleSizeChange(size) {
      this.queryParams.pageSize = size
      this.queryParams.pageIndex = 1
      this.getList()
    },
    onSelectionChange(rows) {
      this.selection = rows || []
    },
    // ========= 新增 =========
    handleAdd() {
      this.formAdd = this.buildEmptyAddForm()
      this.openAdd = true
      this.$nextTick(() => this.$refs.formAdd && this.$refs.formAdd.clearValidate())
    },
    submitAdd() {
      this.$refs.formAdd.validate((valid) => {
        if (!valid) return
        const payload = { ...this.formAdd }
        if (!payload.dept_id) payload.dept_id = 0
        this.submitting = true
        addSysAdmin(payload)
          .then(() => {
            this.$message.success('新增成功')
            this.openAdd = false
            this.getList()
          })
          .catch(() => { /* 全局拦截器已弹错误 */ })
          .finally(() => { this.submitting = false })
      })
    },
    // ========= 编辑 =========
    handleEdit(row) {
      this.currentRow = row
      getSysAdmin(row.admin_id).then((res) => {
        const d = (res && res.data) || {}
        this.formEdit = {
          user_id: d.admin_id || row.admin_id,
          username: d.username || '',
          nickname: d.nickname || '',
          real_name: d.real_name || '',
          dept_id: d.dept_id || null,
          role_ids: Array.isArray(d.role_ids) ? d.role_ids.slice() : (d.role_list || []).map(r => r.role_id),
          phone: d.phone || '',
          email: d.email || '',
          avatar: d.avatar || '',
          remark: d.remark || ''
        }
        this.openEdit = true
        this.$nextTick(() => this.$refs.formEdit && this.$refs.formEdit.clearValidate())
      })
    },
    submitEdit() {
      this.$refs.formEdit.validate((valid) => {
        if (!valid) return
        const payload = {
          nickname: this.formEdit.nickname,
          real_name: this.formEdit.real_name,
          dept_id: this.formEdit.dept_id || 0,
          role_ids: this.formEdit.role_ids,
          phone: this.formEdit.phone,
          email: this.formEdit.email,
          avatar: this.formEdit.avatar,
          remark: this.formEdit.remark
        }
        // H1/H2: 更新请求未带 user_id 或 updateSysAdmin 未正确透传
        agentDebugLog('H1', 'submitEdit_payload_shape', {
          id: Number(this.formEdit.user_id),
          hasUserIdInPayload: Boolean(payload.user_id || payload.user_id === 0),
          roleIdsLen: Array.isArray(payload.role_ids) ? payload.role_ids.length : -1,
          hasAvatar: Boolean(payload.avatar),
          hasDeptId: payload.dept_id !== undefined && payload.dept_id !== null
        })
        this.submitting = true
        updateSysAdmin(this.formEdit.user_id, payload)
          .then(() => {
            this.$message.success('已保存')
            this.openEdit = false
            this.getList()
          })
          .catch((err) => {
            // 不记录 PII，仅记录后端返回结构
            agentDebugLog('H3', 'submitEdit_failed', {
              code: err && err.code,
              msg: err && err.msg ? String(err.msg).slice(0, 180) : '',
              hasData: Boolean(err && err.data)
            })
          })
          .finally(() => { this.submitting = false })
      })
    },
    beforeAvatarUpload(file) {
      const ok = ['image/jpeg', 'image/png', 'image/webp'].includes(file.type)
      if (!ok) {
        this.$message.error('仅支持 jpg/png/webp 格式')
        return false
      }
      if (file.size / 1024 / 1024 > 2) {
        this.$message.error('图片大小不能超过 2MB')
        return false
      }
      return true
    },
    uploadEditAvatar({ file, onSuccess, onError }) {
      // 和 platform-user/profile 一致：先走 /public/uploadFile 拿 URL，再回填编辑表单
      agentDebugLog('H4', 'uploadEditAvatar_start', {
        fileType: file && file.type,
        fileSizeKB: file && file.size ? Math.round(file.size / 1024) : -1
      })
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
          const url = d.full_path || d.path || ''
          if (!url) throw new Error('upload response missing avatar path')
          this.formEdit.avatar = url
          this.$message.success('头像已上传，保存后生效')
          agentDebugLog('H4', 'uploadEditAvatar_uploaded', { hasUrl: Boolean(url), urlLen: url ? String(url).length : 0 })
          onSuccess && onSuccess(res)
        })
        .catch((err) => { onError && onError(err) })
    },
    // ========= 状态切换 =========
    handleStatusChange(row, next) {
      const prev = row.status
      row.status = next // 乐观更新
      changeSysAdminStatus(row.admin_id, next)
        .then(() => {
          this.$message.success(next === '2' ? '已启用' : '已禁用')
        })
        .catch(() => {
          row.status = prev
        })
    },
    handleBatchStatus(status) {
      const ids = this.selection.map(r => r.admin_id)
      if (!ids.length) return
      const label = status === '2' ? '启用' : '禁用'
      this.$confirm(`确认${label} ${ids.length} 个管理员？`, '提示', { type: 'warning' })
        .then(() => Promise.all(ids.map(id => changeSysAdminStatus(id, status).catch(e => e))))
        .then(() => {
          this.$message.success(`批量${label}完成`)
          this.getList()
        })
        .catch(() => {})
    },
    // ========= 删除 =========
    handleDelete(row) {
      this.$confirm(`确认删除管理员「${row.username}」？（软删除）`, '提示', { type: 'warning' })
        .then(() => delSysAdmin(row.admin_id))
        .then(() => {
          this.$message.success('已删除')
          this.getList()
        })
        .catch(() => {})
    },
    handleBatchDelete() {
      const ids = this.selection.map(r => r.admin_id)
      if (!ids.length) return
      this.$confirm(`确认批量删除 ${ids.length} 个管理员？（软删除）`, '提示', { type: 'warning' })
        .then(() => batchDelSysAdmin(ids, '后台批量删除'))
        .then((res) => {
          const d = (res && res.data) || {}
          if (d.failed > 0) {
            this.$message.warning(`完成：成功 ${d.success}，失败 ${d.failed}`)
          } else {
            this.$message.success(`已删除 ${d.success || ids.length} 个`)
          }
          this.getList()
        })
        .catch(() => {})
    },
    // ========= 重置密码 =========
    handleResetPwd(row) {
      this.currentRow = row
      this.formReset = { new_password: '', require_change_on_login: true }
      this.openReset = true
      this.$nextTick(() => this.$refs.formReset && this.$refs.formReset.clearValidate())
    },
    submitReset() {
      this.$refs.formReset.validate((valid) => {
        if (!valid) return
        this.submitting = true
        resetSysAdminPassword(
          this.currentRow.admin_id,
          this.formReset.new_password,
          this.formReset.require_change_on_login
        )
          .then(() => {
            this.$message.success('密码已重置')
            this.openReset = false
            this.getList()
          })
          .catch(() => {})
          .finally(() => { this.submitting = false })
      })
    },
    // ========= 安全策略 =========
    handleSecurity(row) {
      this.currentRow = row
      // 打开前先取详情，拿到 allowed_ips 等完整字段
      getSysAdmin(row.admin_id).then((res) => {
        const d = (res && res.data) || {}
        this.formSecurity = {
          allowed_ips: d.allowed_ips || '',
          allowed_login_start: d.allowed_login_start || '',
          allowed_login_end: d.allowed_login_end || ''
        }
        this.openSecurity = true
      })
    },
    submitSecurity() {
      this.submitting = true
      setSysAdminSecurity(this.currentRow.admin_id, {
        allowedIps: this.formSecurity.allowed_ips,
        allowedLoginStart: this.formSecurity.allowed_login_start,
        allowedLoginEnd: this.formSecurity.allowed_login_end
      })
        .then(() => {
          this.$message.success('安全策略已保存')
          this.openSecurity = false
        })
        .catch(() => {})
        .finally(() => { this.submitting = false })
    },
    // ========= 强制改密标志 =========
    handleToggleForce(row) {
      const next = !row.must_change_password
      const label = next ? '开启「下次登录强制改密」' : '取消「下次登录强制改密」'
      this.$confirm(`确认${label}？`, '提示', { type: 'warning' })
        .then(() => setSysAdminMustChange(row.admin_id, next))
        .then(() => {
          row.must_change_password = next
          this.$message.success('已更新')
        })
        .catch(() => {})
    },
    // ========= 自助改密 =========
    handleSelfChangePwd() {
      this.formSelfPwd = { old_password: '', new_password: '', confirm_password: '' }
      this.openSelfPwd = true
      this.$nextTick(() => this.$refs.formSelfPwd && this.$refs.formSelfPwd.clearValidate())
    },
    submitSelfPwd() {
      this.$refs.formSelfPwd.validate((valid) => {
        if (!valid) return
        this.submitting = true
        changeSelfPassword(this.formSelfPwd.old_password, this.formSelfPwd.new_password)
          .then(() => {
            this.$message.success('密码已更新，下次登录请使用新密码')
            this.openSelfPwd = false
          })
          .catch(() => {})
          .finally(() => { this.submitting = false })
      })
    }
  }
}
</script>

<style scoped>
.mb8 { margin-bottom: 10px; }
.mt12 { margin-top: 12px; }
.ml6 { margin-left: 6px; }
.ml8 { margin-left: 8px; }

.admin-ident .admin-name {
  display: flex;
  align-items: center;
  font-weight: 500;
  color: #303133;
}
.admin-ident .admin-sub {
  margin-top: 2px;
  color: #909399;
  font-size: 12px;
}
.role-tag { margin-right: 4px; margin-bottom: 2px; }
.last-login .sub { color: #909399; font-size: 12px; }

.hint { color: #909399; font-size: 12px; }

.avatar-uploader { display: inline-block; }
.avatar-uploader .el-upload {
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
}
.avatar-uploader .el-upload:hover { border-color: #409eff; }
.avatar-uploader-icon {
  font-size: 22px;
  color: #8c939d;
  width: 80px;
  height: 80px;
  line-height: 80px;
  text-align: center;
}
.avatar { width: 80px; height: 80px; display: block; border-radius: 6px; object-fit: cover; }

.time-row { display: flex; align-items: center; }
.time-dash { margin: 0 8px; color: #909399; }
</style>
