<template>
  <BasicLayout>
    <template #wrapper>
      <!-- 数据看板 -->
      <el-row :gutter="12" class="stat-row">
        <el-col :xs="12" :sm="8" :md="4">
          <el-card shadow="hover" class="stat-card"><div class="stat-label">总设备</div><div class="stat-num">{{ summary.total }}</div></el-card>
        </el-col>
        <el-col :xs="12" :sm="8" :md="4">
          <el-card shadow="hover" class="stat-card stat-online"><div class="stat-label">在线</div><div class="stat-num">{{ summary.online }}</div></el-card>
        </el-col>
        <el-col :xs="12" :sm="8" :md="4">
          <el-card shadow="hover" class="stat-card stat-offline"><div class="stat-label">离线</div><div class="stat-num">{{ summary.offline }}</div></el-card>
        </el-col>
        <el-col :xs="12" :sm="8" :md="4">
          <el-card shadow="hover" class="stat-card stat-unbound"><div class="stat-label">未绑定</div><div class="stat-num">{{ summary.unbound }}</div></el-card>
        </el-col>
        <el-col :xs="12" :sm="8" :md="4">
          <el-card shadow="hover" class="stat-card"><div class="stat-label">今日新增</div><div class="stat-num">{{ summary.today_new }}</div></el-card>
        </el-col>
        <el-col :xs="12" :sm="8" :md="4">
          <el-card shadow="hover" class="stat-card"><div class="stat-label">今日活跃</div><div class="stat-num">{{ summary.today_active }}</div></el-card>
        </el-col>
      </el-row>

      <el-card class="box-card filter-card">
        <div class="toolbar-row">
          <div class="toolbar-left">
            <el-button type="primary" size="small" icon="el-icon-plus" @click="openAdd">添加设备</el-button>
            <el-button size="small" icon="el-icon-upload2" @click="importOpen = true">导入设备</el-button>
            <el-dropdown trigger="click" @command="onBatchCommand">
              <el-button size="small" :disabled="!selectedRows.length">
                批量操作<i class="el-icon-arrow-down el-icon--right" />
              </el-button>
              <el-dropdown-menu slot="dropdown">
                <el-dropdown-item command="enable">批量启用</el-dropdown-item>
                <el-dropdown-item command="disable">批量禁用</el-dropdown-item>
                <el-dropdown-item command="export" divided>导出已选</el-dropdown-item>
              </el-dropdown-menu>
            </el-dropdown>
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
              <el-form-item label="用户">
                <el-input
                  v-model="queryParams.user_query"
                  clearable
                  size="small"
                  placeholder="昵称 / 手机号 / 用户ID"
                  @keyup.enter.native="handleQuery"
                />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8">
              <el-form-item label="绑定状态">
                <el-select v-model="queryParams.bind_status" placeholder="全部" clearable size="small" style="width:100%">
                  <el-option label="全部" value="" />
                  <el-option label="已绑定" :value="1" />
                  <el-option label="未绑定" :value="0" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8">
              <el-form-item label="设备状态">
                <el-select v-model="queryParams.status" placeholder="全部" clearable size="small" style="width:100%">
                  <el-option label="全部" value="" />
                  <el-option label="正常" :value="1" />
                  <el-option label="禁用" :value="2" />
                  <el-option label="未激活" :value="3" />
                  <el-option label="报废" :value="4" />
                </el-select>
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
              <el-form-item label="产品型号">
                <el-select
                  v-model="queryParams.product_key"
                  filterable
                  clearable
                  allow-create
                  default-first-option
                  placeholder="选择 product_key"
                  size="small"
                  style="width:100%"
                >
                  <el-option v-for="k in productKeyOptions" :key="k" :label="k" :value="k" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8">
              <el-form-item label="固件版本">
                <el-input v-model="queryParams.firmware_version" clearable size="small" placeholder="模糊匹配" @keyup.enter.native="handleQuery" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :md="10">
              <el-form-item label="创建时间">
                <el-date-picker
                  v-model="queryParams.dateRange"
                  type="daterange"
                  value-format="yyyy-MM-dd"
                  range-separator="至"
                  start-placeholder="开始"
                  end-placeholder="结束"
                  size="small"
                  style="width:100%"
                  clearable
                />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="24" :md="14" class="filter-actions">
              <el-button type="primary" icon="el-icon-search" size="small" @click="handleQuery">搜索</el-button>
              <el-button icon="el-icon-refresh" size="small" @click="resetQuery">重置</el-button>
              <el-button icon="el-icon-download" size="small" @click="exportCsvFiltered">导出</el-button>
            </el-col>
          </el-row>
        </el-form>
      </el-card>

      <el-card class="box-card table-card">
        <el-table
          v-loading="loading"
          :data="list"
          border
          stripe
          @selection-change="handleSelectionChange"
          @sort-change="handleSortChange"
        >
          <template slot="empty">
            <el-empty description="">
              <div class="empty-text">暂无设备，可通过设备自动注册或手动添加录入设备</div>
              <el-button type="primary" size="small" @click="openAdd">添加设备</el-button>
              <el-button type="text" size="small" @click="openDoc">快速上手</el-button>
            </el-empty>
          </template>
          <el-table-column type="selection" width="48" align="center" :selectable="rowSelectable" />
          <el-table-column label="设备 SN" prop="sn" min-width="150" fixed="left" show-overflow-tooltip>
            <template slot-scope="scope">
              <el-tooltip :content="scope.row.sn" placement="top" :disabled="!scope.row.sn || String(scope.row.sn).length < 16">
                <span>{{ scope.row.sn }}</span>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column label="型号" prop="model" width="110" show-overflow-tooltip />
          <el-table-column label="产品 Key" prop="product_key" width="130" show-overflow-tooltip />
          <el-table-column label="固件" prop="firmware_version" width="100" />
          <el-table-column label="在线" width="88" align="center" sortable="custom" prop="display_online">
            <template slot-scope="scope">
              <el-tag :type="scope.row.display_online === 1 ? 'success' : 'info'" size="small" effect="plain">
                {{ scope.row.display_online === 1 ? '在线' : '离线' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="设备状态" width="96" align="center">
            <template slot-scope="scope">
              <el-tag :type="statusTagType(scope.row.status)" size="small" effect="plain">{{ statusText(scope.row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="绑定用户" min-width="160" show-overflow-tooltip>
            <template slot-scope="scope">
              <template v-if="scope.row.user_id">
                <span>{{ scope.row.user_nickname || '—' }}</span>
                <span v-if="scope.row.user_mobile" class="sub-muted">{{ scope.row.user_mobile }}</span>
                <span class="sub-muted">#{{ scope.row.user_id }}</span>
              </template>
              <span v-else>—</span>
            </template>
          </el-table-column>
          <el-table-column label="绑定时间" width="168" align="center" sortable="custom" prop="bind_time">
            <template slot-scope="scope">
              <span>{{ scope.row.bind_time ? parseTime(scope.row.bind_time) : '—' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="最后在线" width="168" align="center" sortable="custom" prop="last_active_at">
            <template slot-scope="scope">
              <span>{{ scope.row.last_active_at ? parseTime(scope.row.last_active_at) : '—' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" width="168" align="center" sortable="custom" prop="created_at">
            <template slot-scope="scope">
              <span>{{ parseTime(scope.row.created_at) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="IP" prop="ip" width="118" show-overflow-tooltip />
          <el-table-column label="操作" width="360" align="center" fixed="right">
            <template slot-scope="scope">
              <el-button type="primary" size="mini" plain @click="openDetail(scope.row)">详情</el-button>
              <el-button
                v-if="Number(scope.row.status) === 3"
                size="mini"
                type="success"
                plain
                @click="openActivate(scope.row)"
              >激活</el-button>
              <el-button
                v-if="Number(scope.row.status) !== 3"
                size="mini"
                :type="scope.row.status === 2 ? 'success' : 'warning'"
                plain
                @click="toggleDisable(scope.row)"
              >{{ scope.row.status === 2 ? '启用' : '禁用' }}</el-button>
              <el-button size="mini" type="danger" plain :disabled="!scope.row.user_id" @click="doUnbind(scope.row)">解绑</el-button>
              <el-button size="mini" plain @click="doReboot(scope.row)">重启</el-button>
              <el-button size="mini" plain @click="openOta(scope.row)">OTA</el-button>
            </template>
          </el-table-column>
        </el-table>
        <pagination
          :total="total"
          :page.sync="queryParams.page"
          :limit.sync="queryParams.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          @pagination="getList"
        />
      </el-card>

      <!-- 详情 -->
      <el-dialog title="设备详情" :visible.sync="detailOpen" width="900px" append-to-body>
        <template v-if="detail && detail.device">
          <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:8px">
            <span style="font-weight:600">基础信息</span>
            <el-button type="primary" size="mini" plain @click="openDeviceInfoEdit">编辑设备信息</el-button>
          </div>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="SN">{{ detail.device.sn }}</el-descriptions-item>
            <el-descriptions-item label="产品 Key">{{ detail.device.product_key }}</el-descriptions-item>
            <el-descriptions-item label="设备密钥">{{ detail.device.device_secret_masked || '—' }}</el-descriptions-item>
            <el-descriptions-item label="型号">{{ detail.device.model }}</el-descriptions-item>
            <el-descriptions-item label="固件">{{ detail.device.firmware_version }}</el-descriptions-item>
            <el-descriptions-item label="硬件">{{ detail.device.hardware_version }}</el-descriptions-item>
            <el-descriptions-item label="MAC">{{ detail.device.mac }}</el-descriptions-item>
            <el-descriptions-item label="IP">{{ detail.device.ip }}</el-descriptions-item>
            <el-descriptions-item label="在线(展示)">
              <el-tag :type="detail.device.display_online === 1 ? 'success' : 'info'" size="mini">{{ detail.device.display_online === 1 ? '在线' : '离线' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="statusTagType(detail.device.status)" size="mini">{{ statusText(detail.device.status) }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="设备名称">{{ detail.device.device_name || '—' }}</el-descriptions-item>
            <el-descriptions-item label="位置">{{ detail.device.location || '—' }}</el-descriptions-item>
            <el-descriptions-item label="分组">{{ detail.device.group_id || '—' }}</el-descriptions-item>
            <el-descriptions-item label="备注" :span="2">{{ detail.device.remark || '—' }}</el-descriptions-item>
            <el-descriptions-item label="标签" :span="2">{{ formatDeviceTags(detail.device.tags) }}</el-descriptions-item>
            <el-descriptions-item label="扩展配置" :span="2">
              <pre class="config-pre">{{ formatDeviceConfig(detail.device.config) }}</pre>
            </el-descriptions-item>
          </el-descriptions>
          <el-descriptions :column="2" border size="small" title="绑定信息" style="margin-top:12px">
            <el-descriptions-item label="用户">{{ detail.device.user_nickname || '—' }} / {{ detail.device.user_id || '—' }}</el-descriptions-item>
            <el-descriptions-item label="绑定时间">{{ detail.device.bind_time ? parseTime(detail.device.bind_time) : '—' }}</el-descriptions-item>
          </el-descriptions>

          <el-tabs v-model="detailTab" style="margin-top:12px">
            <el-tab-pane label="指令记录" name="ins">
              <el-table :data="detail.instructions || []" size="small" max-height="280">
                <el-table-column prop="id" label="ID" width="70" />
                <el-table-column prop="cmd" label="指令" width="120" />
                <el-table-column prop="status" label="状态" width="80" />
                <el-table-column prop="created_at" label="时间" width="168">
                  <template slot-scope="s">{{ parseTime(s.row.created_at) }}</template>
                </el-table-column>
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="OTA 任务" name="ota">
              <el-table :data="detail.ota_tasks || []" size="small" max-height="280">
                <el-table-column prop="id" label="ID" width="70" />
                <el-table-column prop="from_version" label="从" width="90" />
                <el-table-column prop="to_version" label="到" width="90" />
                <el-table-column prop="status" label="状态" width="80" />
                <el-table-column prop="progress" label="进度" width="70" />
                <el-table-column prop="created_at" label="时间" width="168">
                  <template slot-scope="s">{{ parseTime(s.row.created_at) }}</template>
                </el-table-column>
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="事件日志" name="ev">
              <el-table :data="detail.events || []" size="small" max-height="280">
                <el-table-column prop="event_type" label="类型" width="120" />
                <el-table-column prop="content" label="内容" min-width="160" show-overflow-tooltip />
                <el-table-column prop="operator" label="操作者" width="100" />
                <el-table-column prop="created_at" label="时间" width="168">
                  <template slot-scope="s">{{ parseTime(s.row.created_at) }}</template>
                </el-table-column>
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="状态上报" name="status">
              <div style="margin-bottom:8px;display:flex;flex-wrap:wrap;align-items:center;gap:8px">
                <el-date-picker
                  v-model="statusLogDateRange"
                  type="daterange"
                  range-separator="至"
                  start-placeholder="开始日期"
                  end-placeholder="结束日期"
                  value-format="yyyy-MM-dd"
                  size="small"
                  clearable
                  style="width:260px"
                  @change="onStatusLogDateChange"
                />
                <el-button size="small" :loading="statusLogsLoading" @click="fetchStatusLogs">查询</el-button>
                <el-button
                  type="primary"
                  plain
                  size="small"
                  :loading="reportStatusLoading"
                  :disabled="!detail || !detail.device"
                  @click="triggerReportStatus"
                >立即获取状态</el-button>
                <el-button
                  size="small"
                  :disabled="!detail || !detail.device"
                  @click="openManualStatusDialog"
                >手动填报状态</el-button>
              </div>
              <el-table v-loading="statusLogsLoading" :data="statusLogsList" size="small" max-height="320" empty-text="暂无状态上报记录">
                <el-table-column prop="batteryLevel" label="电量%" width="72" align="center" />
                <el-table-column label="上报类型" width="88" align="center">
                  <template slot-scope="s">{{ formatReportType(s.row.reportType) }}</template>
                </el-table-column>
                <el-table-column label="存储" min-width="120">
                  <template slot-scope="s">
                    {{ formatBytes(s.row.storageUsed) }} / {{ formatBytes(s.row.storageTotal) }}
                  </template>
                </el-table-column>
                <el-table-column prop="speakerCount" label="扬声器数" width="80" align="center" />
                <el-table-column label="UWB" min-width="140" show-overflow-tooltip>
                  <template slot-scope="s">{{ formatUwb(s.row) }}</template>
                </el-table-column>
                <el-table-column label="声学" width="100" align="center">
                  <template slot-scope="s">
                    {{ s.row.acousticCalibrated === 1 ? '已校准' : '未校准' }}
                    <span v-if="s.row.acousticOffset != null" class="sub-muted"> / {{ s.row.acousticOffset }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="采集时间" width="160">
                  <template slot-scope="s">{{ parseTime(s.row.reportedAt) }}</template>
                </el-table-column>
                <el-table-column label="服务端时间" width="160">
                  <template slot-scope="s">{{ parseTime(s.row.createdAt) }}</template>
                </el-table-column>
              </el-table>
              <el-pagination
                v-if="statusLogsTotal > 0"
                small
                layout="total, prev, pager, next"
                :total="statusLogsTotal"
                :page-size="statusLogsPageSize"
                :current-page="statusLogsPage"
                style="margin-top:8px;text-align:right"
                @current-change="onStatusLogsPage"
              />
            </el-tab-pane>
          </el-tabs>
        </template>
        <span slot="footer" class="dialog-footer">
          <el-button @click="detailOpen=false">关闭</el-button>
        </span>
      </el-dialog>

      <el-dialog
        title="手动填报状态"
        :visible.sync="manualStatusOpen"
        width="580px"
        append-to-body
        @close="resetManualStatusForm"
      >
        <p class="sub-muted" style="margin:0 0 12px;line-height:1.5;font-size:12px">
          将写入一条「手动」类型记录（补录/测试）。与「立即获取状态」不同：不经过设备，直接入库。
        </p>
        <el-form ref="manualStatusFormRef" label-width="112px" size="small" @submit.native.prevent>
          <el-form-item label="设备 SN">
            <el-input :value="manualStatusForm.sn" disabled />
          </el-form-item>
          <el-form-item label="电量 %" required>
            <el-input-number v-model="manualStatusForm.battery_level" :min="0" :max="100" :controls="true" style="width:100%" />
          </el-form-item>
          <el-form-item label="已用存储" required>
            <el-input-number v-model="manualStatusForm.storage_used" :min="0" :step="1024" style="width:100%" />
          </el-form-item>
          <el-form-item label="总存储" required>
            <el-input-number v-model="manualStatusForm.storage_total" :min="0" :step="1024" style="width:100%" />
          </el-form-item>
          <el-form-item label="扬声器数" required>
            <el-input-number v-model="manualStatusForm.speaker_count" :min="0" style="width:100%" />
          </el-form-item>
          <el-form-item label="UWB X/Y/Z">
            <div style="display:flex;gap:8px;flex-wrap:wrap">
              <el-input v-model="manualStatusForm.uwb_x" placeholder="X" style="width:100px" />
              <el-input v-model="manualStatusForm.uwb_y" placeholder="Y" style="width:100px" />
              <el-input v-model="manualStatusForm.uwb_z" placeholder="Z" style="width:100px" />
            </div>
          </el-form-item>
          <el-form-item label="声学校准">
            <el-select v-model="manualStatusForm.acoustic_calibrated" style="width:120px">
              <el-option :value="0" label="未校准" />
              <el-option :value="1" label="已校准" />
            </el-select>
            <span style="margin-left:12px;color:#909399">偏移</span>
            <el-input v-model="manualStatusForm.acoustic_offset" placeholder="可选" style="width:140px;margin-left:8px" />
          </el-form-item>
          <el-form-item label="采集时间">
            <el-date-picker
              v-model="manualStatusForm.reported_at"
              type="datetime"
              placeholder="留空则使用当前时间"
              value-format="yyyy-MM-dd HH:mm:ss"
              style="width:100%"
              clearable
            />
          </el-form-item>
        </el-form>
        <span slot="footer" class="dialog-footer">
          <el-button @click="manualStatusOpen = false">取消</el-button>
          <el-button type="primary" :loading="manualStatusSubmitting" @click="submitManualStatusReport">保存并上报</el-button>
        </span>
      </el-dialog>

      <el-dialog title="OTA 升级" :visible.sync="otaOpen" width="420px" append-to-body>
        <el-form label-width="88px">
          <el-form-item label="目标版本">
            <el-input v-model="otaForm.version" placeholder="须已在 ota_firmware 中配置" />
          </el-form-item>
        </el-form>
        <span slot="footer" class="dialog-footer">
          <el-button @click="otaOpen=false">取消</el-button>
          <el-button type="primary" :loading="otaLoading" @click="submitOta">创建任务</el-button>
        </span>
      </el-dialog>

      <el-dialog title="添加设备" :visible.sync="addOpen" width="480px" append-to-body @close="addForm = { sn: '', product_key: '', model: '', mac: '' }">
        <p class="hint" style="margin:0 0 12px;line-height:1.5">出厂密钥由<strong>系统自动随机生成</strong>；创建成功后会展示密钥与示例 HMAC 签名，请写入设备固件。云端以数据库中 bcrypt 存储的密钥为准。</p>
        <el-form label-width="96px">
          <el-form-item label="设备 SN" required>
            <el-input v-model="addForm.sn" placeholder="16～32 位字母数字" />
          </el-form-item>
          <el-form-item label="产品 Key" required>
            <el-select v-model="addForm.product_key" filterable allow-create default-first-option placeholder="选择或输入" style="width:100%">
              <el-option v-for="k in productKeyOptions" :key="k" :label="k" :value="k" />
            </el-select>
          </el-form-item>
          <el-form-item label="MAC">
            <el-input v-model="addForm.mac" placeholder="可选，如 AA:BB:CC:DD:EE:FF" />
          </el-form-item>
          <el-form-item label="型号">
            <el-input v-model="addForm.model" placeholder="可选" />
          </el-form-item>
        </el-form>
        <span slot="footer" class="dialog-footer">
          <el-button @click="addOpen=false">取消</el-button>
          <el-button type="primary" :loading="addLoading" @click="submitAdd">确定</el-button>
        </span>
      </el-dialog>

      <el-dialog title="请保存出厂资料（仅显示一次）" :visible.sync="createResultOpen" width="560px" append-to-body @close="createResult = null">
        <p v-if="createResult" class="hint" style="margin-bottom:12px;line-height:1.6">
          将 <strong>出厂密钥</strong> 烧录到固件；调用「云端认证激活」时需使用相同算法：参与签名的字段包含 <code>sn</code>、<code>product_key</code>、<code>nonce</code>、<code>timestamp</code>（及可选 <code>firmware_version</code>/<code>ip</code>），HMAC-SHA256 十六进制小写。下方为<strong>建档时生成的示例</strong>：<code>timestamp</code> 须在调用激活接口的 ±5 分钟内，且 <code>nonce</code> 不可重复使用。
        </p>
        <div v-if="createResult" class="create-result-block">
          <div v-for="row in createResultRows" :key="row.k" class="create-result-row">
            <div class="create-result-label">{{ row.label }}</div>
            <el-input :value="row.v" type="textarea" :autosize="{ minRows: row.multiline ? 2 : 1 }" readonly />
            <el-button size="mini" @click="copyText(row.v)">复制</el-button>
          </div>
        </div>
        <span slot="footer" class="dialog-footer">
          <el-button type="primary" @click="copyCreateResultAll">一键复制全部</el-button>
          <el-button @click="createResultOpen=false">已保存</el-button>
        </span>
      </el-dialog>

      <el-dialog title="导入设备" :visible.sync="importOpen" width="560px" append-to-body>
        <p class="hint">每行一条：<code>SN,product_key,型号</code>（型号可留空）</p>
        <el-input v-model="importText" type="textarea" :rows="10" placeholder="DEV001,pk_demo_1,Model-A" />
        <span slot="footer" class="dialog-footer">
          <el-button @click="importOpen=false">取消</el-button>
          <el-button type="primary" :loading="importLoading" @click="submitImport">导入</el-button>
        </span>
      </el-dialog>

      <el-dialog title="云端认证激活" :visible.sync="activateOpen" width="520px" append-to-body>
        <p class="hint" style="margin-bottom:12px;line-height:1.6">
          云端仅存密钥的<strong>哈希</strong>，无法在页面还原明文密钥或代为计算 HMAC。此处由<strong>已登录管理员</strong>发起可信激活：云端校验产品与「未激活」状态后，将设备置为「正常」，效果与设备自行携带密钥完成 <code>POST /activate-cloud</code> 一致。开放/脚本集成请仍使用带密钥与签名的 <code>/activate-cloud</code>。
        </p>
        <el-form label-width="108px" size="small">
          <el-form-item label="产品 Key">
            <el-input :value="activateForm.product_key" disabled />
          </el-form-item>
          <el-form-item label="设备 SN">
            <el-input :value="activateForm.sn" disabled />
          </el-form-item>
        </el-form>
        <span slot="footer" class="dialog-footer">
          <el-button @click="activateOpen=false">取消</el-button>
          <el-button type="primary" :loading="activateLoading" @click="submitActivateCloudAdmin">确认激活</el-button>
        </span>
      </el-dialog>

      <el-dialog
        title="编辑设备信息"
        :visible.sync="deviceInfoEditOpen"
        width="620px"
        append-to-body
        @close="resetDeviceInfoForm"
      >
        <el-form label-width="112px" size="small" @submit.native.prevent>
          <el-form-item label="设备名称">
            <el-input v-model="deviceInfoForm.device_name" placeholder="后台展示用名称" maxlength="128" show-word-limit />
          </el-form-item>
          <el-form-item label="位置">
            <el-input v-model="deviceInfoForm.location" placeholder="如 3楼301室" maxlength="256" />
          </el-form-item>
          <el-form-item label="分组 ID">
            <el-input v-model="deviceInfoForm.group_id" placeholder="group_xxx" maxlength="64" />
          </el-form-item>
          <el-form-item label="备注">
            <el-input v-model="deviceInfoForm.remark" type="textarea" :rows="2" placeholder="备注" />
          </el-form-item>
          <el-form-item label="标签">
            <el-input v-model="deviceInfoForm.tagsStr" placeholder="逗号分隔，如 会议室,高频" />
          </el-form-item>
          <el-form-item label="运行状态">
            <el-select v-model="deviceInfoForm.apiStatus" style="width:100%">
              <el-option :value="1" label="启用" />
              <el-option :value="0" label="禁用" />
              <el-option :value="2" label="维护（写入 maintenance_mode）" />
            </el-select>
          </el-form-item>
          <el-divider content-position="left">扩展配置</el-divider>
          <el-form-item label="上报间隔(s)">
            <el-input-number v-model="deviceInfoForm.report_interval" :min="10" :max="86400" style="width:100%" />
          </el-form-item>
          <el-form-item label="音量上限">
            <el-input-number v-model="deviceInfoForm.volume_limit" :min="0" :max="100" style="width:100%" />
          </el-form-item>
          <el-form-item label="夜间模式">
            <el-switch v-model="deviceInfoForm.night_mode" />
          </el-form-item>
          <el-form-item label="异常自重启">
            <el-switch v-model="deviceInfoForm.auto_restart" />
          </el-form-item>
          <el-form-item label="调试模式">
            <el-switch v-model="deviceInfoForm.debug_mode" />
          </el-form-item>
        </el-form>
        <span slot="footer" class="dialog-footer">
          <el-button @click="deviceInfoEditOpen = false">取消</el-button>
          <el-button type="primary" :loading="deviceInfoSubmitting" @click="submitDeviceInfoUpdate">保存</el-button>
        </span>
      </el-dialog>
    </template>
  </BasicLayout>
</template>

<script>
import {
  listPlatformDevices,
  getPlatformDeviceSummary,
  getPlatformDeviceEnum,
  getPlatformDeviceDetail,
  createPlatformDevice,
  activatePlatformDeviceCloudAdmin,
  importPlatformDevices,
  batchPlatformDeviceStatus,
  setPlatformDeviceStatus,
  unbindPlatformDevice,
  sendPlatformDeviceCommand,
  pushPlatformDeviceOTA,
  listPlatformDeviceStatusLogs,
  triggerPlatformDeviceReportStatus,
  manualPlatformDeviceStatusReport,
  updatePlatformDeviceInfo
} from '@/api/admin/platform-device'
import BasicLayout from '@/layout/BasicLayout'
import Pagination from '@/components/Pagination'

const FILTER_STORAGE_KEY = 'platform_device_filters_v1'

export default {
  name: 'PlatformDevice',
  components: { BasicLayout, Pagination },
  data() {
    return {
      loading: false,
      list: [],
      total: 0,
      summary: {
        total: 0,
        online: 0,
        offline: 0,
        unbound: 0,
        today_new: 0,
        today_active: 0
      },
      productKeyOptions: [],
      selectedRows: [],
      filterExpanded: true,
      windowWidth: typeof window !== 'undefined' ? window.innerWidth : 1200,
      queryParams: {
        page: 1,
        pageSize: 20,
        sn: '',
        sn_mode: 'exact',
        user_query: '',
        bind_status: '',
        status: '',
        online_status: '',
        product_key: '',
        firmware_version: '',
        dateRange: null,
        sort_by: '',
        sort_order: ''
      },
      detailOpen: false,
      detail: null,
      detailTab: 'ins',
      otaOpen: false,
      otaLoading: false,
      otaForm: { sn: '', version: '' },
      addOpen: false,
      addLoading: false,
      addForm: { sn: '', product_key: '', model: '', mac: '' },
      createResultOpen: false,
      createResult: null,
      importOpen: false,
      importLoading: false,
      importText: '',
      activateOpen: false,
      activateLoading: false,
      activateForm: {
        product_key: '',
        sn: ''
      },
      statusLogsList: [],
      statusLogsTotal: 0,
      statusLogsPage: 1,
      statusLogsPageSize: 10,
      statusLogsLoading: false,
      statusLogDateRange: null,
      reportStatusLoading: false,
      manualStatusOpen: false,
      manualStatusSubmitting: false,
      manualStatusForm: {
        device_id: null,
        sn: '',
        battery_level: 50,
        storage_used: 0,
        storage_total: 0,
        speaker_count: 0,
        uwb_x: '',
        uwb_y: '',
        uwb_z: '',
        acoustic_calibrated: 0,
        acoustic_offset: '',
        reported_at: ''
      },
      deviceInfoEditOpen: false,
      deviceInfoSubmitting: false,
      deviceInfoForm: {
        device_name: '',
        location: '',
        group_id: '',
        remark: '',
        tagsStr: '',
        apiStatus: 1,
        report_interval: 60,
        volume_limit: 80,
        night_mode: false,
        auto_restart: false,
        debug_mode: false
      }
    }
  },
  computed: {
    isNarrow() {
      return this.windowWidth < 992
    },
    createResultRows() {
      const d = this.createResult
      if (!d) return []
      return [
        { k: 'pk', label: '产品 Key', v: d.product_key || '' },
        { k: 'sn', label: '设备 SN', v: d.sn || '' },
        { k: 'secret', label: '出厂密钥（明文）', v: d.device_secret || '', multiline: true },
        { k: 'ts', label: '示例 timestamp（秒）', v: d.bootstrap_timestamp != null ? String(d.bootstrap_timestamp) : '' },
        { k: 'nonce', label: '示例 nonce', v: d.bootstrap_nonce || '' },
        { k: 'sig', label: '示例签名（HMAC-SHA256 小写 hex）', v: d.bootstrap_signature || '', multiline: true }
      ]
    }
  },
  watch: {
    detailTab(val) {
      if (val === 'status' && this.detailOpen && this.detail && this.detail.device && (this.detail.device.id || this.detail.device.sn)) {
        this.fetchStatusLogs()
      }
    },
    detail(val) {
      if (!this.detailOpen || this.detailTab !== 'status') return
      if (!val || !val.device || (!val.device.id && !val.device.sn)) return
      this.fetchStatusLogs()
    }
  },
  created() {
    this.restoreFilters()
    this.loadProductKeys()
    this.loadSummary()
    this.getList()
    window.addEventListener('resize', this.onResize)
  },
  beforeDestroy() {
    window.removeEventListener('resize', this.onResize)
  },
  methods: {
    onResize() {
      this.windowWidth = window.innerWidth
    },
    restoreFilters() {
      try {
        const raw = localStorage.getItem(FILTER_STORAGE_KEY)
        if (!raw) return
        const o = JSON.parse(raw)
        if (o && typeof o === 'object') {
          this.queryParams = { ...this.queryParams, ...o, page: 1 }
        }
      } catch (e) {
        // ignore
        void e
      }
    },
    persistFilters() {
      try {
        const rest = { ...this.queryParams }
        delete rest.page
        localStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify(rest))
      } catch (e) {
        // ignore
        void e
      }
    },
    async loadSummary() {
      try {
        const res = await getPlatformDeviceSummary()
        const d = (res && res.data) || {}
        this.summary = {
          total: d.total || 0,
          online: d.online || 0,
          offline: d.offline || 0,
          unbound: d.unbound || 0,
          today_new: d.today_new || 0,
          today_active: d.today_active || 0
        }
      } catch (e) {
        // ignore
        void e
      }
    },
    async loadProductKeys() {
      try {
        const res = await getPlatformDeviceEnum()
        const d = (res && res.data) || {}
        const arr = d.product_keys
        this.productKeyOptions = Array.isArray(arr) ? arr : []
      } catch (e) {
        this.productKeyOptions = []
      }
    },
    refreshAll() {
      this.loadSummary()
      this.loadProductKeys()
      this.getList()
    },
    statusText(s) {
      const m = { 1: '正常', 2: '禁用', 3: '未激活', 4: '报废' }
      return m[s] || s
    },
    statusTagType(s) {
      if (s === 1) return 'success'
      if (s === 2) return 'danger'
      if (s === 4) return 'info'
      return 'warning'
    },
    rowSelectable() {
      return true
    },
    handleSelectionChange(rows) {
      this.selectedRows = rows || []
    },
    handleSortChange({ prop, order }) {
      const map = {
        created_at: 'created_at',
        last_active_at: 'last_active_at',
        bind_time: 'bind_time',
        display_online: 'last_active_at'
      }
      if (!order) {
        this.queryParams.sort_by = ''
        this.queryParams.sort_order = ''
      } else {
        this.queryParams.sort_by = map[prop] || 'id'
        this.queryParams.sort_order = order === 'ascending' ? 'asc' : 'desc'
      }
      this.queryParams.page = 1
      this.getList()
    },
    buildQuery() {
      const q = {
        page: this.queryParams.page,
        pageSize: this.queryParams.pageSize
      }
      const sn = (this.queryParams.sn || '').trim()
      if (sn) {
        q.sn = sn
        q.sn_mode = this.queryParams.sn_mode || 'exact'
      }
      const uq = (this.queryParams.user_query || '').trim()
      if (uq) q.user_query = uq
      if (this.queryParams.bind_status !== '' && this.queryParams.bind_status != null) {
        q.bind_status = this.queryParams.bind_status
      }
      if (this.queryParams.status !== '' && this.queryParams.status != null) q.status = this.queryParams.status
      if (this.queryParams.online_status !== '' && this.queryParams.online_status != null) {
        q.online_status = this.queryParams.online_status
      }
      const pk = (this.queryParams.product_key || '').trim()
      if (pk) q.product_key = pk
      const fw = (this.queryParams.firmware_version || '').trim()
      if (fw) q.firmware_version = fw
      if (this.queryParams.dateRange && this.queryParams.dateRange.length === 2) {
        q.created_from = this.queryParams.dateRange[0]
        q.created_to = this.queryParams.dateRange[1]
      }
      if (this.queryParams.sort_by) {
        q.sort_by = this.queryParams.sort_by
        q.sort_order = this.queryParams.sort_order || 'desc'
      }
      return q
    },
    handleQuery() {
      this.queryParams.page = 1
      this.persistFilters()
      this.getList()
    },
    resetQuery() {
      this.queryParams = {
        page: 1,
        pageSize: 20,
        sn: '',
        sn_mode: 'exact',
        user_query: '',
        bind_status: '',
        status: '',
        online_status: '',
        product_key: '',
        firmware_version: '',
        dateRange: null,
        sort_by: '',
        sort_order: ''
      }
      localStorage.removeItem(FILTER_STORAGE_KEY)
      this.getList()
    },
    async getList() {
      this.loading = true
      try {
        const res = await listPlatformDevices(this.buildQuery())
        const data = (res && res.data) || {}
        this.list = data.list || []
        const cnt = data.count != null ? data.count : data.total
        this.total = Number(cnt) || 0
      } finally {
        this.loading = false
      }
    },
    async openDetail(row) {
      this.detailOpen = true
      this.detail = null
      this.detailTab = 'ins'
      this.statusLogsList = []
      this.statusLogsTotal = 0
      this.statusLogsPage = 1
      this.statusLogDateRange = null
      try {
        const rowId = row && (row.id != null ? row.id : row.device_id != null ? row.device_id : row.deviceId)
        const rowSnRaw =
          row &&
          (row.sn != null && String(row.sn) !== ''
            ? row.sn
            : row.device_sn != null && String(row.device_sn) !== ''
              ? row.device_sn
              : row.deviceSn)
        const rowSn = rowSnRaw != null && String(rowSnRaw) !== '' ? String(rowSnRaw).trim() : ''
        const res = await getPlatformDeviceDetail(rowSn, rowId)
        const payload = res && res.data
        if (!payload || !payload.device) {
          this.$message.error('设备详情数据异常，请稍后重试或联系管理员')
          return
        }
        this.detail = payload
      } catch (e) {
        const msg =
          (e && e.msg) ||
          (e && e.response && e.response.data && e.response.data.msg) ||
          (e && e.message) ||
          '加载设备详情失败'
        this.$message.error(msg)
      }
    },
    onStatusLogDateChange() {
      this.statusLogsPage = 1
      if (this.detailTab === 'status') {
        this.fetchStatusLogs()
      }
    },
    onStatusLogsPage(p) {
      this.statusLogsPage = p
      this.fetchStatusLogs()
    },
    async fetchStatusLogs() {
      if (!this.detail || !this.detail.device) return
      const dev = this.detail.device
      const devIdRaw = dev.id != null && dev.id !== '' ? dev.id : dev.device_id
      const devSn =
        (dev.sn != null && String(dev.sn).trim() !== '' ? String(dev.sn).trim() : '') ||
        (dev.device_sn != null && String(dev.device_sn).trim() !== '' ? String(dev.device_sn).trim() : '')
      if (
        (devIdRaw == null || devIdRaw === '' || !Number.isFinite(Number(devIdRaw))) &&
        !devSn
      ) {
        return
      }
      this.statusLogsLoading = true
      try {
        const q = { page: this.statusLogsPage, pageSize: this.statusLogsPageSize }
        if (devIdRaw != null && devIdRaw !== '' && Number.isFinite(Number(devIdRaw))) {
          q.device_id = Number(devIdRaw)
        }
        if (this.statusLogDateRange && this.statusLogDateRange.length === 2) {
          q.from = this.statusLogDateRange[0]
          q.to = this.statusLogDateRange[1]
        }
        const res = await listPlatformDeviceStatusLogs(devSn, q)
        const data = (res && res.data) || {}
        this.statusLogsList = data.list || []
        const cnt = data.count != null ? data.count : data.total
        this.statusLogsTotal = Number(cnt) || 0
      } catch (e) {
        this.statusLogsList = []
        this.statusLogsTotal = 0
        const msg =
          (e && e.msg) ||
          (e && e.response && e.response.data && e.response.data.msg) ||
          (e && e.message) ||
          '加载状态上报失败'
        this.$message.error(msg)
      } finally {
        this.statusLogsLoading = false
      }
    },
    async triggerReportStatus() {
      if (!this.detail || !this.detail.device) return
      const dev = this.detail.device
      this.reportStatusLoading = true
      try {
        const rid =
          dev.id != null && dev.id !== '' && Number.isFinite(Number(dev.id)) ? Number(dev.id) : undefined
        await triggerPlatformDeviceReportStatus({
          device_id: rid,
          sn: rid ? undefined : dev.sn || undefined
        })
        this.$message.success('已下发立即上报指令，设备在线时将上报；可稍后点击查询刷新')
        setTimeout(() => this.fetchStatusLogs(), 1500)
      } catch (e) {
        const msg =
          (e && e.msg) ||
          (e && e.response && e.response.data && e.response.data.msg) ||
          (e && e.message) ||
          '下发失败'
        this.$message.error(msg)
      } finally {
        this.reportStatusLoading = false
      }
    },
    openManualStatusDialog() {
      if (!this.detail || !this.detail.device) return
      const d = this.detail.device
      const id =
        d.id != null && d.id !== '' && Number.isFinite(Number(d.id)) ? Number(d.id) : null
      this.manualStatusForm = {
        device_id: id,
        sn: d.sn || '',
        battery_level: 50,
        storage_used: 0,
        storage_total: 0,
        speaker_count: 0,
        uwb_x: '',
        uwb_y: '',
        uwb_z: '',
        acoustic_calibrated: 0,
        acoustic_offset: '',
        reported_at: ''
      }
      this.manualStatusOpen = true
    },
    resetManualStatusForm() {
      this.manualStatusForm = {
        device_id: null,
        sn: '',
        battery_level: 50,
        storage_used: 0,
        storage_total: 0,
        speaker_count: 0,
        uwb_x: '',
        uwb_y: '',
        uwb_z: '',
        acoustic_calibrated: 0,
        acoustic_offset: '',
        reported_at: ''
      }
    },
    optFloatStr(s) {
      if (s == null || String(s).trim() === '') return undefined
      const n = Number(String(s).trim())
      return Number.isFinite(n) ? n : undefined
    },
    async submitManualStatusReport() {
      const f = this.manualStatusForm
      if (!f.device_id && !f.sn) {
        this.$message.warning('缺少设备信息')
        return
      }
      this.manualStatusSubmitting = true
      try {
        const fid =
          f.device_id != null && f.device_id !== '' && Number.isFinite(Number(f.device_id))
            ? Number(f.device_id)
            : undefined
        const payload = {
          device_id: fid,
          sn: fid ? undefined : f.sn,
          battery_level: f.battery_level,
          storage_used: f.storage_used,
          storage_total: f.storage_total,
          speaker_count: f.speaker_count,
          acoustic_calibrated: f.acoustic_calibrated
        }
        const ox = this.optFloatStr(f.uwb_x)
        const oy = this.optFloatStr(f.uwb_y)
        const oz = this.optFloatStr(f.uwb_z)
        if (ox !== undefined) payload.uwb_x = ox
        if (oy !== undefined) payload.uwb_y = oy
        if (oz !== undefined) payload.uwb_z = oz
        const ao = this.optFloatStr(f.acoustic_offset)
        if (ao !== undefined) payload.acoustic_offset = ao
        if (f.reported_at) payload.reported_at = f.reported_at
        await manualPlatformDeviceStatusReport(payload)
        this.$message.success('已保存')
        this.manualStatusOpen = false
        this.fetchStatusLogs()
      } catch (e) {
        const msg =
          (e && e.msg) ||
          (e && e.response && e.response.data && e.response.data.msg) ||
          (e && e.message) ||
          '保存失败'
        this.$message.error(msg)
      } finally {
        this.manualStatusSubmitting = false
      }
    },
    formatBytes(n) {
      if (n == null || n === '') return '—'
      const v = Number(n)
      if (!Number.isFinite(v) || v < 0) return '—'
      const u = ['B', 'KB', 'MB', 'GB', 'TB']
      let i = 0
      let x = v
      while (x >= 1024 && i < u.length - 1) {
        x /= 1024
        i++
      }
      return `${x.toFixed(i > 0 ? 1 : 0)} ${u[i]}`
    },
    formatReportType(v) {
      const m = { auto: '定时', manual: '手动', sync: '补发' }
      const k = v != null ? String(v).toLowerCase() : ''
      return m[k] || (v ? String(v) : '—')
    },
    formatDeviceTags(tags) {
      if (tags == null) return '—'
      if (Array.isArray(tags)) return tags.length ? tags.join(', ') : '—'
      return String(tags) || '—'
    },
    formatDeviceConfig(cfg) {
      if (cfg == null || cfg === '') return '—'
      try {
        return typeof cfg === 'object' ? JSON.stringify(cfg, null, 2) : String(cfg)
      } catch (e) {
        return '—'
      }
    },
    deviceStatusToApiStatus(dbStatus, maintenanceMode) {
      if (maintenanceMode) return 2
      if (dbStatus === 2) return 0
      return 1
    },
    openDeviceInfoEdit() {
      if (!this.detail || !this.detail.device) return
      const d = this.detail.device
      const cfg = d.config && typeof d.config === 'object' ? d.config : {}
      const tags = d.tags
      let tagsStr = ''
      if (Array.isArray(tags)) tagsStr = tags.join(', ')
      else if (typeof tags === 'string') tagsStr = tags
      const maint = !!cfg.maintenance_mode
      this.deviceInfoForm = {
        device_name: d.device_name || '',
        location: d.location || '',
        group_id: d.group_id || '',
        remark: d.remark || '',
        tagsStr,
        apiStatus: this.deviceStatusToApiStatus(d.status, maint),
        report_interval: cfg.report_interval != null ? Number(cfg.report_interval) : 60,
        volume_limit: cfg.volume_limit != null ? Number(cfg.volume_limit) : 80,
        night_mode: !!cfg.night_mode,
        auto_restart: !!cfg.auto_restart,
        debug_mode: !!cfg.debug_mode
      }
      this.deviceInfoEditOpen = true
    },
    resetDeviceInfoForm() {
      this.deviceInfoForm = {
        device_name: '',
        location: '',
        group_id: '',
        remark: '',
        tagsStr: '',
        apiStatus: 1,
        report_interval: 60,
        volume_limit: 80,
        night_mode: false,
        auto_restart: false,
        debug_mode: false
      }
    },
    async reloadDetail() {
      if (!this.detail || !this.detail.device) return
      const row = this.detail.device
      const rowId = row.id != null ? row.id : row.device_id != null ? row.device_id : row.deviceId
      const rowSnRaw =
        row.sn != null && String(row.sn) !== ''
          ? row.sn
          : row.device_sn != null && String(row.device_sn) !== ''
            ? row.device_sn
            : row.deviceSn
      const rowSn = rowSnRaw != null && String(rowSnRaw) !== '' ? String(rowSnRaw).trim() : ''
      if (
        (rowId == null || rowId === '' || !Number.isFinite(Number(rowId))) &&
        !rowSn
      ) {
        return
      }
      try {
        const res = await getPlatformDeviceDetail(rowSn, rowId)
        const payload = res && res.data
        if (payload && payload.device) {
          this.detail = payload
        }
      } catch (e) {
        const msg =
          (e && e.msg) ||
          (e && e.response && e.response.data && e.response.data.msg) ||
          (e && e.message) ||
          '刷新详情失败'
        this.$message.error(msg)
      }
    },
    async submitDeviceInfoUpdate() {
      if (!this.detail || !this.detail.device) return
      const dev = this.detail.device
      const id =
        dev.id != null && dev.id !== '' && Number.isFinite(Number(dev.id)) ? Number(dev.id) : null
      if (!id) {
        this.$message.warning('无法识别设备 ID')
        return
      }
      this.deviceInfoSubmitting = true
      try {
        const f = this.deviceInfoForm
        const tags = f.tagsStr
          ? f.tagsStr
              .split(/[,，]/)
              .map((s) => s.trim())
              .filter(Boolean)
          : []
        await updatePlatformDeviceInfo({
          device_id: id,
          updates: {
            device_name: f.device_name,
            location: f.location,
            group_id: f.group_id,
            remark: f.remark,
            tags,
            status: f.apiStatus,
            config: {
              report_interval: f.report_interval,
              volume_limit: f.volume_limit,
              night_mode: f.night_mode,
              auto_restart: f.auto_restart,
              debug_mode: f.debug_mode
            }
          }
        })
        this.$message.success('保存成功')
        this.deviceInfoEditOpen = false
        await this.reloadDetail()
      } catch (e) {
        const msg =
          (e && e.msg) ||
          (e && e.response && e.response.data && e.response.data.msg) ||
          (e && e.message) ||
          '保存失败'
        this.$message.error(msg)
      } finally {
        this.deviceInfoSubmitting = false
      }
    },
    formatUwb(row) {
      const x = row.uwbX
      const y = row.uwbY
      const z = row.uwbZ
      if (x == null && y == null && z == null) return '—'
      const f = (v) => (v == null || !Number.isFinite(Number(v)) ? '—' : Number(v).toFixed(2))
      return `${f(x)}, ${f(y)}, ${f(z)}`
    },
    openAdd() {
      this.addForm = { sn: '', product_key: '', model: '', mac: '' }
      this.addOpen = true
    },
    copyText(text) {
      const s = text != null ? String(text) : ''
      if (!s) {
        this.$message.warning('无内容')
        return
      }
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(s).then(() => this.$message.success('已复制')).catch(() => this.fallbackCopy(s))
      } else {
        this.fallbackCopy(s)
      }
    },
    fallbackCopy(s) {
      const ta = document.createElement('textarea')
      ta.value = s
      ta.style.position = 'fixed'
      ta.style.left = '-9999px'
      document.body.appendChild(ta)
      ta.select()
      try {
        document.execCommand('copy')
        this.$message.success('已复制')
      } catch (e) {
        this.$message.error('复制失败，请手动复制')
      }
      document.body.removeChild(ta)
    },
    copyCreateResultAll() {
      const lines = this.createResultRows.map((r) => `${r.label}: ${r.v}`)
      this.copyText(lines.join('\n'))
    },
    async submitAdd() {
      const sn = (this.addForm.sn || '').trim()
      const pk = (this.addForm.product_key || '').trim()
      if (!sn || !pk) {
        this.$message.warning('请填写 SN 与产品 Key')
        return
      }
      this.addLoading = true
      try {
        const body = {
          sn,
          product_key: pk,
          model: (this.addForm.model || '').trim()
        }
        const mac = (this.addForm.mac || '').trim()
        if (mac) body.mac = mac
        const res = await createPlatformDevice(body)
        const d = res && (res.data !== undefined ? res.data : res)
        this.createResult = d
        this.createResultOpen = true
        this.addOpen = false
        this.$message.success('设备已创建，请保存下方出厂密钥与示例参数')
        this.loadSummary()
        this.loadProductKeys()
        this.getList()
      } catch (e) {
        // request 拦截器已弹出后端 msg
        void e
      } finally {
        this.addLoading = false
      }
    },
    async submitImport() {
      const lines = (this.importText || '').split(/\r?\n/).map((l) => l.trim()).filter(Boolean)
      const items = []
      for (const line of lines) {
        const p = line.split(',').map((x) => x.trim())
        if (p.length < 2) continue
        items.push({ sn: p[0], product_key: p[1], model: p[2] || '' })
      }
      if (!items.length) {
        this.$message.warning('无有效行')
        return
      }
      this.importLoading = true
      try {
        const res = await importPlatformDevices({ items })
        const d = (res && res.data) || {}
        const ok = d.success || 0
        const errs = d.errors || []
        this.$message.success(`成功 ${ok} 条` + (errs.length ? `，失败 ${errs.length} 条` : ''))
        if (errs.length) {
          console.warn(errs)
        }
        this.importOpen = false
        this.importText = ''
        this.loadSummary()
        this.loadProductKeys()
        this.getList()
      } finally {
        this.importLoading = false
      }
    },
    async onBatchCommand(cmd) {
      const sns = this.selectedRows.map((r) => r.sn).filter(Boolean)
      if (!sns.length) return
      if (cmd === 'export') {
        this.exportRows(this.selectedRows)
        return
      }
      const status = cmd === 'enable' ? 1 : 2
      const act = cmd === 'enable' ? '启用' : '禁用'
      try {
        await this.$confirm(`确认批量${act} ${sns.length} 台设备？`, '提示', { type: 'warning' })
      } catch (e) {
        return
      }
      try {
        await batchPlatformDeviceStatus({ sns, status })
        this.$message.success('操作成功')
        this.loadSummary()
        this.getList()
      } catch (e) { /* request 已提示 */ }
    },
    openActivate(row) {
      this.activateForm = {
        product_key: row.product_key || '',
        sn: row.sn || ''
      }
      this.activateOpen = true
    },
    async submitActivateCloudAdmin() {
      const f = this.activateForm
      const sn = String(f.sn || '').trim()
      const pk = String(f.product_key || '').trim()
      if (!sn || !pk) {
        this.$message.warning('缺少 SN 或产品 Key')
        return
      }
      this.activateLoading = true
      try {
        await activatePlatformDeviceCloudAdmin({ sn, product_key: pk })
        this.$message.success('设备已激活为「正常」')
        this.activateOpen = false
        this.loadSummary()
        this.getList()
      } catch (e) {
        /* request 已提示 */
      } finally {
        this.activateLoading = false
      }
    },
    async toggleDisable(row) {
      const next = row.status === 2 ? 1 : 2
      const act = next === 2 ? '禁用' : '启用'
      try {
        await this.$confirm(`确认${act}设备 ${row.sn}？`, '提示', { type: 'warning' })
      } catch (e) {
        return
      }
      try {
        await setPlatformDeviceStatus({ sn: row.sn, status: next })
        this.$message.success('操作成功')
        this.loadSummary()
        this.getList()
      } catch (e) { /* */ }
    },
    async doUnbind(row) {
      try {
        await this.$confirm(`确认解绑设备 ${row.sn}？将解除与用户的绑定关系。`, '强制解绑', { type: 'warning' })
      } catch (e) {
        return
      }
      try {
        await unbindPlatformDevice({ sn: row.sn })
        this.$message.success('已解绑')
        this.loadSummary()
        this.getList()
      } catch (e) { /* */ }
    },
    async doReboot(row) {
      try {
        await this.$confirm(`向设备 ${row.sn} 下发远程重启指令？`, '重启设备', { type: 'warning' })
      } catch (e) {
        return
      }
      try {
        await sendPlatformDeviceCommand({ sn: row.sn, command: 'reboot', params: {}})
        this.$message.success('重启指令已入队')
      } catch (e) { /* */ }
    },
    openOta(row) {
      this.otaForm = { sn: row.sn, version: '' }
      this.otaOpen = true
    },
    async submitOta() {
      const v = (this.otaForm.version || '').trim()
      if (!v) {
        this.$message.warning('请填写版本号')
        return
      }
      this.otaLoading = true
      try {
        await pushPlatformDeviceOTA({ sn: this.otaForm.sn, version: v })
        this.$message.success('OTA 任务已创建')
        this.otaOpen = false
        this.getList()
      } finally {
        this.otaLoading = false
      }
    },
    exportRows(rows) {
      const headers = ['sn', 'model', 'product_key', 'firmware_version', 'online', 'status', 'user_id', 'nickname', 'mobile', 'bind_time', 'last_active', 'ip']
      const lines = [headers.join(',')]
      rows.forEach((r) => {
        const cols = [
          r.sn,
          r.model,
          r.product_key,
          r.firmware_version,
          r.display_online === 1 ? 'online' : 'offline',
          this.statusText(r.status),
          r.user_id,
          (r.user_nickname || '').replace(/,/g, ' '),
          (r.user_mobile || '').replace(/,/g, ' '),
          r.bind_time ? this.parseTime(r.bind_time) : '',
          r.last_active_at ? this.parseTime(r.last_active_at) : '',
          r.ip
        ]
        lines.push(cols.map((x) => `"${String(x).replace(/"/g, '""')}"`).join(','))
      })
      this.downloadCsv(lines, `设备列表_${this.formatNowFile()}.csv`)
      this.$message.success('已导出')
    },
    exportCsvFiltered() {
      this.exportRows(this.list)
    },
    downloadCsv(lines, name) {
      const blob = new Blob(['\ufeff' + lines.join('\n')], { type: 'text/csv;charset=utf-8;' })
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = name
      a.click()
      URL.revokeObjectURL(a.href)
    },
    formatNowFile() {
      const d = new Date()
      const p = (n) => String(n).padStart(2, '0')
      return `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}_${p(d.getHours())}${p(d.getMinutes())}${p(d.getSeconds())}`
    },
    openDoc() {
      window.open('https://github.com/go-admin-team/go-admin', '_blank')
    }
  }
}
</script>

<style scoped>
.stat-row {
  margin-bottom: 12px;
}
.stat-card {
  margin-bottom: 8px;
  text-align: center;
}
.stat-label {
  font-size: 12px;
  color: #909399;
}
.stat-num {
  font-size: 20px;
  font-weight: 600;
  margin-top: 4px;
}
.stat-online .stat-num { color: #67c23a; }
.stat-offline .stat-num { color: #909399; }
.stat-unbound .stat-num { color: #e6a23c; }
.filter-card {
  margin-bottom: 16px;
}
.toolbar-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}
.toolbar-left > .el-button,
.toolbar-left > .el-dropdown {
  margin-right: 8px;
  margin-bottom: 4px;
}
.filter-form {
  margin-bottom: 0;
}
.sn-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
.sn-mode {
  width: 88px;
  flex-shrink: 0;
}
.filter-actions {
  display: flex;
  align-items: flex-end;
  flex-wrap: wrap;
  gap: 8px;
}
@media (max-width: 991px) {
  .filter-collapsed .el-col:nth-child(n+4) {
    display: none;
  }
}
.sub-muted {
  margin-left: 6px;
  color: #909399;
  font-size: 12px;
}
.empty-text {
  margin-bottom: 12px;
  color: #606266;
}
.table-card {
  margin-bottom: 16px;
}
.hint {
  font-size: 13px;
  color: #606266;
  margin-bottom: 8px;
}
.hint code {
  background: #f4f4f5;
  padding: 2px 6px;
  border-radius: 4px;
}
.create-result-block {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.create-result-row {
  display: grid;
  grid-template-columns: 120px 1fr auto;
  gap: 8px;
  align-items: start;
}
.create-result-label {
  font-size: 13px;
  color: #606266;
  padding-top: 6px;
}
@media (max-width: 600px) {
  .create-result-row {
    grid-template-columns: 1fr;
  }
  .create-result-label {
    padding-top: 0;
  }
}
.config-pre {
  font-size: 12px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 140px;
  overflow: auto;
  background: #f5f7fa;
  padding: 8px;
  border-radius: 4px;
}
</style>
