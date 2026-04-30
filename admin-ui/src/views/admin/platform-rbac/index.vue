<template>
  <BasicLayout>
    <template #wrapper>
      <el-card class="box-card rbac-card">
        <div class="toolbar">
          <div class="title">
            <i class="el-icon-lock title-icon" aria-hidden="true" />
            <span>权限管理 · 平台权限矩阵</span>
          </div>
          <div class="actions">
            <el-tooltip content="重新加载矩阵与角色配置" placement="bottom">
              <el-button size="mini" icon="el-icon-refresh" @click="loadAll">刷新</el-button>
            </el-tooltip>
            <el-tooltip content="保存所有已修改的行" placement="bottom">
              <el-button
                size="mini"
                type="primary"
                icon="el-icon-document-checked"
                :disabled="!canEdit || !hasAnyDirty"
                @click="bulkSave"
              >批量保存</el-button>
            </el-tooltip>
            <el-tooltip content="放弃所有未保存的修改（恢复为上次加载的配置）" placement="bottom">
              <el-button
                size="mini"
                icon="el-icon-refresh-left"
                :disabled="!canEdit || !hasAnyDirty"
                @click="bulkReset"
              >批量重置</el-button>
            </el-tooltip>
          </div>
        </div>

        <el-alert
          v-if="current && current.roleKey"
          class="mb12 rbac-alert-current"
          type="info"
          :closable="false"
          show-icon
        >
          <template slot="title">
            <span class="alert-line">当前登录角色：<strong class="role-bracket">【{{ currentRoleLabel }}】</strong></span>
            <span v-if="currentRoleDesc" class="alert-sub">{{ currentRoleDesc }}</span>
          </template>
        </el-alert>

        <div class="perm-legend mb12">
          <div class="perm-legend__title">
            <i class="el-icon-info perm-legend__icon" />
            权限取值说明
          </div>
          <ul class="perm-legend__list">
            <li><span class="perm-dot perm-dot--all" /><span class="perm-legend__k">全部</span>：拥有该模块所有操作权限（增删改查）</li>
            <li><span class="perm-dot perm-dot--view" /><span class="perm-legend__k">查看</span>：仅只读（仅 GET/HEAD，其余方法会被拒绝）</li>
            <li><span class="perm-dot perm-dot--self" /><span class="perm-legend__k">自己</span>：仅本人数据；只读侧同样仅 GET/HEAD，其余方法会被拒绝</li>
            <li><span class="perm-dot perm-dot--none" /><span class="perm-legend__k">无</span>：无该模块任何操作权限</li>
          </ul>
        </div>

        <el-alert
          class="mb12"
          type="warning"
          :closable="false"
          show-icon
          title="仅「超级管理员」「系统管理员」可保存或批量保存权限配置；其他角色仅可查看。"
        />

        <el-table v-loading="loading" :data="roles" border class="rbac-table" style="width: 100%">
          <el-table-column label="角色" prop="roleName" min-width="180">
            <template slot-scope="{ row }">
              <span class="role-cell">
                <span class="perm-dot" :class="roleRowDotClass(row.roleKey)" :title="row.roleKey" />
                <span>{{ row.roleName }}</span>
              </span>
            </template>
          </el-table-column>

          <el-table-column v-for="m in modules" :key="m.key" :label="m.title" min-width="168">
            <template slot-scope="{ row }">
              <div class="perm-cell">
                <span
                  class="perm-dot perm-dot--cell"
                  :class="moduleDotClass(row, m.key)"
                  :title="moduleValueLabel(row, m.key)"
                />
                <el-select
                  v-model="row.permissions.modules[m.key]"
                  size="mini"
                  class="perm-select"
                  :disabled="!canEdit"
                  @change="markDirty(row.roleKey)"
                >
                  <el-option label="全部" value="all" />
                  <el-option label="查看" value="view" />
                  <el-option label="自己" value="self" />
                  <el-option label="无" value="none" />
                </el-select>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="操作" width="168" fixed="right">
            <template slot-scope="{ row }">
              <el-tooltip content="保存本行" placement="top">
                <el-button
                  type="primary"
                  size="mini"
                  icon="el-icon-document-checked"
                  :disabled="!canEdit || !dirty[row.roleKey]"
                  @click="saveRow(row)"
                />
              </el-tooltip>
              <el-tooltip content="重置本行" placement="top">
                <el-button
                  size="mini"
                  icon="el-icon-refresh-left"
                  :disabled="!canEdit || !dirty[row.roleKey]"
                  @click="resetRow(row)"
                />
              </el-tooltip>
            </template>
          </el-table-column>
        </el-table>

        <div class="footer-hint">
          <div class="footer-hint__title">说明</div>
          <div>取值含义与上图例一致；「查看 / 自己」在只读语义下仅放行 GET/HEAD。</div>
          <div>单行「保存 / 重置」与顶部「批量保存 / 批量重置」行为一致，仅作用范围不同。</div>
        </div>
      </el-card>
    </template>
  </BasicLayout>
</template>

<script>
import { getPlatformRbacMatrix, listPlatformRbacRoles, updatePlatformRbacRole } from '@/api/admin/platform-rbac'

export default {
  name: 'PlatformRbac',
  data() {
    return {
      loading: false,
      modules: [],
      roles: [],
      current: null,
      dirty: {},
      originalByRoleKey: {}
    }
  },
  computed: {
    canEdit() {
      const rk = (this.current && this.current.roleKey) || ''
      return rk === 'super_admin' || rk === 'admin'
    },
    hasAnyDirty() {
      return Object.keys(this.dirty || {}).some((k) => this.dirty[k])
    },
    currentRoleLabel() {
      const rk = (this.current && this.current.roleKey) || ''
      const map = {
        super_admin: '超级管理员',
        admin: '系统管理员',
        operator: '运营人员',
        operations: '运营人员',
        finance: '财务管理员',
        ordinary_user: '普通用户',
        guest: '游客'
      }
      if (map[rk]) return map[rk]
      const hit = (this.roles || []).find((r) => r.roleKey === rk)
      return (hit && hit.roleName) || rk || '—'
    },
    currentRoleDesc() {
      const r = this.current && this.current.role
      const d = r && r.description
      return d ? String(d) : ''
    }
  },
  created() {
    this.loadAll()
  },
  methods: {
    markDirty(roleKey) {
      this.$set(this.dirty, roleKey, true)
    },
    roleRowDotClass(roleKey) {
      const m = {
        super_admin: 'perm-dot--all',
        admin: 'perm-dot--all',
        operator: 'perm-dot--view',
        operations: 'perm-dot--view',
        finance: 'perm-dot--view',
        ordinary_user: 'perm-dot--self',
        guest: 'perm-dot--none'
      }
      return m[roleKey] || 'perm-dot--none'
    },
    moduleDotClass(row, modKey) {
      const v = row.permissions && row.permissions.modules && row.permissions.modules[modKey]
      return this.valueToDotClass(v)
    },
    valueToDotClass(v) {
      if (v === 'all') return 'perm-dot--all'
      if (v === 'view') return 'perm-dot--view'
      if (v === 'self') return 'perm-dot--self'
      return 'perm-dot--none'
    },
    moduleValueLabel(row, modKey) {
      const v = row.permissions && row.permissions.modules && row.permissions.modules[modKey]
      const map = { all: '全部', view: '查看', self: '自己', none: '无' }
      return map[v] || v || '—'
    },
    pickDisplayedModules(modules) {
      const desired = [
        { key: 'user_mgmt', title: '用户管理' },
        { key: 'member_mgmt', title: '会员管理' },
        { key: 'device_mgmt', title: '设备管理' },
        { key: 'content_mgmt', title: '内容管理' }
      ]
      const list = Array.isArray(modules) ? modules : []
      const byKey = new Map(list.map((m) => [m && m.key, m]))
      return desired.map((d) => {
        const hit = byKey.get(d.key)
        return hit ? { ...hit, title: hit.title || d.title } : d
      })
    },
    normalizeModules(mod) {
      const v = mod || {}
      return {
        user_mgmt: v.user_mgmt || 'none',
        member_mgmt: v.member_mgmt || 'none',
        device_mgmt: v.device_mgmt || 'none',
        content_mgmt: v.content_mgmt || 'none',
        ota: v.ota || 'none',
        stats: v.stats || 'none',
        audit_log: v.audit_log || 'none',
        sys_config: v.sys_config || 'none'
      }
    },
    loadAll() {
      this.loading = true
      Promise.all([getPlatformRbacMatrix(), listPlatformRbacRoles()])
        .then(([m1, m2]) => {
          this.current = m1.data || {}
          const d2 = m2.data || {}
          this.modules = this.pickDisplayedModules(d2.modules || [])
          const list = d2.list || []
          this.roles = list.map((r) => {
            const p = r.permissions || {}
            const nm = this.normalizeModules((p && p.modules) || {})
            this.$set(this.originalByRoleKey, r.roleKey, JSON.parse(JSON.stringify(nm)))
            return {
              ...r,
              permissions: {
                ...p,
                modules: this.normalizeModules((p && p.modules) || {})
              }
            }
          })
          this.dirty = {}
        })
        .finally(() => {
          this.loading = false
        })
    },
    saveRow(row) {
      const roleKey = row.roleKey
      const modules = this.normalizeModules(row.permissions && row.permissions.modules)
      updatePlatformRbacRole(roleKey, modules)
        .then(() => {
          this.$message.success('保存成功')
          this.$set(this.dirty, roleKey, false)
          this.loadAll()
        })
        .catch(() => {})
    },
    resetRow(row) {
      const roleKey = row.roleKey
      const orig = this.originalByRoleKey && this.originalByRoleKey[roleKey]
      if (!orig) return
      row.permissions = row.permissions || {}
      row.permissions.modules = JSON.parse(JSON.stringify(orig))
      this.$set(this.dirty, roleKey, false)
    },
    bulkSave() {
      if (!this.canEdit) return
      const keys = this.roles.map((r) => r.roleKey).filter((k) => this.dirty[k])
      if (!keys.length) {
        this.$message.info('没有待保存的修改')
        return
      }
      this.$confirm(`确认将 ${keys.length} 个已修改角色的配置保存到服务器？`, '批量保存', { type: 'warning' })
        .then(() => {
          const tasks = keys.map((k) => {
            const row = this.roles.find((r) => r.roleKey === k)
            const modules = this.normalizeModules(row.permissions && row.permissions.modules)
            return updatePlatformRbacRole(k, modules)
          })
          return Promise.all(tasks)
        })
        .then(() => {
          this.$message.success('批量保存成功')
          this.loadAll()
        })
        .catch(() => {})
    },
    bulkReset() {
      if (!this.canEdit) return
      const keys = this.roles.map((r) => r.roleKey).filter((k) => this.dirty[k])
      if (!keys.length) {
        this.$message.info('没有待重置的修改')
        return
      }
      this.$confirm(`确认放弃 ${keys.length} 个角色的未保存修改？`, '批量重置', { type: 'warning' })
        .then(() => {
          keys.forEach((k) => {
            const row = this.roles.find((r) => r.roleKey === k)
            if (row) this.resetRow(row)
          })
          this.$message.success('已恢复为上次加载的配置')
        })
        .catch(() => {})
    }
  }
}
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  flex-wrap: wrap;
  gap: 8px;
}
.title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  display: flex;
  align-items: center;
  gap: 8px;
}
.title-icon {
  color: #409eff;
  font-size: 18px;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.mb12 {
  margin-bottom: 12px;
}
.rbac-alert-current .alert-line {
  display: block;
  line-height: 1.6;
}
.rbac-alert-current .alert-sub {
  display: block;
  margin-top: 4px;
  font-weight: normal;
  color: #606266;
  font-size: 13px;
}
.role-bracket {
  color: #303133;
}
.perm-legend {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 12px 14px;
  background: #fafafa;
}
.perm-legend__title {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 6px;
}
.perm-legend__icon {
  color: #909399;
}
.perm-legend__list {
  margin: 0;
  padding-left: 0;
  list-style: none;
  font-size: 13px;
  color: #606266;
  line-height: 1.85;
}
.perm-legend__list li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.perm-legend__k {
  font-weight: 600;
  color: #303133;
  margin-right: 4px;
}
.perm-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-top: 5px;
}
.perm-dot--cell {
  margin-top: 10px;
}
.perm-dot--all {
  background: #409eff;
  box-shadow: 0 0 0 1px rgba(64, 158, 255, 0.35);
}
.perm-dot--view {
  background: #e6a23c;
  box-shadow: 0 0 0 1px rgba(230, 162, 60, 0.35);
}
.perm-dot--self {
  background: #67c23a;
  box-shadow: 0 0 0 1px rgba(103, 194, 58, 0.35);
}
.perm-dot--none {
  background: #909399;
  box-shadow: 0 0 0 1px rgba(144, 147, 153, 0.35);
}
.role-cell {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.perm-cell {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.perm-select {
  width: 118px !important;
}
.rbac-table ::v-deep .el-table th {
  background: #f5f7fa;
}
.footer-hint {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid #ebeef5;
  color: #606266;
  font-size: 12px;
  line-height: 1.7;
}
.footer-hint__title {
  font-weight: 600;
  margin-bottom: 4px;
  color: #303133;
}
</style>
