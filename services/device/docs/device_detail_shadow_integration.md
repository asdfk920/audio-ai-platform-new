# 设备详情页 - 设备影子集成指南

## 📋 概述

本文档介绍如何在设备详情页中集成**设备影子查看功能**，让用户可以实时查看设备的完整状态信息。

## 🎯 适用场景

从截图可以看到，这是一个 **Admin 后台设备详情弹窗**，包含：
- ✅ 基础信息（SN、密钥、固件、MAC、IP、在线状态等）
- ✅ 绑定信息
- 📋 标签页：指令记录 | OTA 任务 | 事件日志 | **状态上报** ← 在这里展示影子

## 🔌 接口信息

### 接口地址

```
GET /api/device/shadow/detail?device_sn={device_sn}
```

### 认证方式

- 需要 JWT Token（用户登录后的 token）
- 请求头：`Authorization: Bearer {token}`

### 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| device_sn | string | 是 | 设备序列号 |

### 响应数据结构

```json
{
  "code": 200,
  "success": true,
  "message": "查询成功",
  "data": {
    "device_sn": "AUSP2605000001",
    "found": true,

    // ========== 基本信息 ==========
    "basic_info": {
      "device_sn": "AUSP2605000001",
      "status": "online",
      "status_text": "在线",
      "is_online": true,
      "firmware_version": "v2.1.0",
      "ip_address": "192.168.1.100",
      "battery_level": 85,
      "run_state": "运行中"
    },

    // ========== 上报属性列表（格式化） ==========
    "reported_attrs": [
      {
        "key": "power",
        "label": "电源状态",
        "value": "on",
        "value_text": "on",
        "type": "string",
        "category": "system",
        "changed": false
      },
      {
        "key": "volume",
        "label": "音量",
        "value": 80,
        "value_text": "80",
        "type": "number",
        "category": "media",
        "changed": true  // ⚠️ 与期望值不同
      },
      {
        "key": "play_state",
        "label": "播放状态",
        "value": "playing",
        "value_text": "playing",
        "type": "string",
        "category": "media",
        "changed": false
      }
    ],

    // ========== 期望属性列表 ==========
    "desired_attrs": [
      {
        "key": "volume",
        "label": "音量",
        "value": 90,
        "value_text": "90",
        "type": "number",
        "category": "media"
      }
    ],

    // ========== 原始数据（用于高级功能） ==========
    "raw_reported": {
      "power": "on",
      "volume": 80,
      "play_state": "playing",
      "current_song": "song_001"
    },
    "raw_desired": {
      "volume": 90
    },

    // ========== 版本信息 ==========
    "version_info": {
      "current_version": 42,
      "update_count": 41,
      "last_updated": 1704067200
    },

    // ========== 时间线信息 ==========
    "timeline_info": {
      "last_updated_at": 1704067200,
      "duration_seconds": 3600,
      "active_ratio": 0.95
    },

    // ========== 状态历史 ==========
    "status_history": [
      {
        "from_status": "",
        "to_status": "online",
        "change_time": 1704067200,
        "reason": ""
      }
    ],

    // ========== 元数据 ==========
    "metadata": {
      "firmware": "v2.1.0",
      "ip": "192.168.1.100"
    }
  }
}
```

---

## 💻 前端实现方案

### 方案 1: 在"状态上报"标签页中展示（推荐）

#### HTML 结构

```html
<!-- 设备详情弹窗 -->
<div class="device-detail-modal">
  <div class="modal-header">
    <h2>设备详情</h2>
    <button class="close-btn" onclick="closeModal()">×</button>
  </div>

  <!-- 基础信息区域 -->
  <div class="basic-info-section">
    <h3>基础信息</h3>
    <table class="info-table">
      <tr><td>SN</td><td id="device-sn">AUSP260</td></tr>
      <tr><td>在线(展示)</td><td id="online-status">🟢 在线</td></tr>
      <tr><td>状态</td><td id="device-status">正常</td></tr>
      <!-- 其他基础字段... -->
    </table>
  </div>

  <!-- 绑定信息区域 -->
  <div class="bind-info-section">
    <h3>绑定信息</h3>
    <table class="info-table">
      <tr><td>用户</td><td id="bind-user">— / —</td></tr>
      <tr><td>绑定时间</td><td id="bind-time">—</td></tr>
    </table>
  </div>

  <!-- 标签页区域 -->
  <div class="tabs-section">
    <div class="tab-headers">
      <button class="tab-btn active" onclick="switchTab('commands')">指令记录</button>
      <button class="tab-btn" onclick="switchTab('ota')">OTA 任务</button>
      <button class="tab-btn" onclick="switchTab('events')">事件日志</button>
      <button class="tab-btn" onclick="switchTab('shadow')">状态上报</button>  <!-- 新增 -->
    </div>

    <!-- 状态上报标签内容 -->
    <div id="tab-shadow" class="tab-content" style="display:none;">
      <div class="shadow-container">
        <!-- 影子概览卡片 -->
        <div class="shadow-overview">
          <div class="overview-card online">
            <div class="card-label">设备状态</div>
            <div class="card-value" id="shadow-status">—</div>
          </div>
          <div class="overview-card">
            <div class="card-label">版本号</div>
            <div class="card-value" id="shadow-version">—</div>
          </div>
          <div class="overview-card">
            <div class="card-label">更新次数</div>
            <div class="card-value" id="shadow-updates">—</div>
          </div>
          <div class="overview-card">
            <div class="card-label">最后更新</div>
            <div class="card-value" id="shadow-last-update">—</div>
          </div>
        </div>

        <!-- 属性表格 -->
        <div class="attributes-section">
          <h4>设备属性 (Reported)</h4>
          <table class="attributes-table" id="reported-attrs-table">
            <thead>
              <tr>
                <th>属性名称</th>
                <th>当前值</th>
                <th>期望值</th>
                <th>类型</th>
                <th>分类</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              <!-- 动态生成 -->
            </tbody>
          </table>
        </div>

        <!-- 原始 JSON 数据（可折叠） -->
        <details class="raw-data-section">
          <summary>查看原始数据 (JSON)</summary>
          <pre id="raw-json-data"></pre>
        </details>
      </div>
    </div>

    <!-- 其他标签内容... -->
  </div>
</div>
```

#### CSS 样式

```css
/* 影子容器样式 */
.shadow-container {
  padding: 20px;
}

/* 概览卡片 */
.shadow-overview {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.overview-card {
  background: #f5f7fa;
  border-radius: 8px;
  padding: 16px;
  text-align: center;
  transition: all 0.3s ease;
}

.overview-card.online {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.overview-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
}

.card-label {
  font-size: 12px;
  color: #999;
  margin-bottom: 8px;
  text-transform: uppercase;
}

.overview-card.online .card-label {
  color: rgba(255,255,255,0.8);
}

.card-value {
  font-size: 24px;
  font-weight: bold;
  color: #333;
}

.overview-card.online .card-value {
  color: white;
}

/* 属性表格 */
.attributes-section {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
}

.attributes-section h4 {
  margin: 0;
  padding: 16px 20px;
  background: #fafafa;
  border-bottom: 1px solid #eee;
  font-size: 14px;
  color: #333;
}

.attributes-table {
  width: 100%;
  border-collapse: collapse;
}

.attributes-table th,
.attributes-table td {
  padding: 12px 16px;
  text-align: left;
  border-bottom: 1px solid #f0f0f0;
  font-size: 13px;
}

.attributes-table th {
  background: #fafafa;
  font-weight: 600;
  color: #666;
}

.attributes-table tr:hover {
  background: #f9f9f9;
}

/* 属性值样式 */
.attr-value {
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 600;
  color: #333;
}

.attr-value.string { color: #e91e63; }
.attr-value.number { color: #2196f3; }
.attr-value.boolean { color: #4caf50; }

/* 变更标记 */
.status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 600;
}

.status-badge.synced {
  background: #e8f5e9;
  color: #2e7d32;
}

.status-badge.changed {
  background: #fff3e0;
  color: #ef6c00;
}

/* 分类标签 */
.category-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  background: #f0f0f0;
  color: #666;
}

.category-tag.system { background: #e3f2fd; color: #1565c0; }
.category-tag.media { background: #fce4ec; color: #c2185b; }
.category-tag.custom { background: #f3e5f5; color: #7b1fa2; }

/* 原始数据区域 */
.raw-data-section {
  margin-top: 20px;
  background: #fafafa;
  border-radius: 8px;
  padding: 16px;
}

.raw-data-section summary {
  cursor: pointer;
  font-weight: 600;
  color: #666;
  outline: none;
}

.raw-data-section pre {
  margin-top: 12px;
  padding: 16px;
  background: #263238;
  color: #aed581;
  border-radius: 4px;
  overflow-x: auto;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  line-height: 1.5;
}

/* 加载动画 */
.loading-spinner {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #f3f3f3;
  border-top: 3px solid #3498db;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #999;
}

.empty-state-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.empty-state-text {
  font-size: 14px;
  line-height: 1.5;
}
```

#### JavaScript 实现

```javascript
// 全局变量存储当前设备SN
let currentDeviceSN = null;

// 打开设备详情弹窗
function openDeviceDetail(deviceSN) {
  currentDeviceSN = deviceSN;

  // 显示弹窗
  document.querySelector('.device-detail-modal').style.display = 'block';

  // 默认显示第一个标签
  switchTab('commands');

  // 自动加载影子数据（如果用户切换到该标签）
  loadShadowData(deviceSN);
}

// 切换标签页
function switchTab(tabName) {
  // 隐藏所有标签内容
  document.querySelectorAll('.tab-content').forEach(tab => {
    tab.style.display = 'none';
  });

  // 移除所有按钮的 active 状态
  document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.classList.remove('active');
  });

  // 显示选中的标签
  const selectedTab = document.getElementById(`tab-${tabName}`);
  if (selectedTab) {
    selectedTab.style.display = 'block';
  }

  // 高亮选中按钮
  event.target.classList.add('active');

  // 如果切换到"状态上报"标签，加载数据
  if (tabName === 'shadow' && currentDeviceSN) {
    loadShadowData(currentDeviceSN);
  }
}

// 加载设备影子数据
async function loadShadowData(deviceSN) {
  const container = document.getElementById('tab-shadow');
  if (!container) return;

  // 显示加载状态
  container.innerHTML = `
    <div class="loading-spinner">
      <div class="spinner"></div>
      <p style="margin-left: 12px;">正在加载设备影子...</p>
    </div>
  `;

  try {
    const response = await fetch(
      `/api/device/shadow/detail?device_sn=${encodeURIComponent(deviceSN)}`,
      {
        headers: {
          'Authorization': `Bearer ${getToken()}`
        }
      }
    );

    const result = await response.json();

    if (!result.success) {
      throw new Error(result.message || '查询失败');
    }

    renderShadowData(result.data);

  } catch (error) {
    console.error('加载影子失败:', error);
    container.innerHTML = `
      <div class="empty-state">
        <div class="empty-state-icon">⚠️</div>
        <div class="empty-state-text">
          加载失败<br>
          <small>${error.message}</small>
        </div>
      </div>
    `;
  }
}

// 渲染影子数据
function renderShadowData(shadowData) {
  if (!shadowData.found) {
    renderEmptyShadow();
    return;
  }

  // 1. 渲染概览卡片
  renderOverviewCards(shadowData);

  // 2. 渲染属性表格
  renderAttributesTable(shadowData);

  // 3. 渲染原始 JSON 数据
  renderRawJSON(shadowData);
}

// 渲染概览卡片
function renderOverviewCards(data) {
  const statusEl = document.getElementById('shadow-status');
  const versionEl = document.getElementById('shadow-version');
  const updatesEl = document.getElementById('shadow-updates');
  const lastUpdateEl = document.getElementById('shadow-last-update');

  // 设备状态
  statusEl.textContent = data.basic_info?.status_text || '—';

  // 版本号
  versionEl.textContent = `v${data.version_info?.current_version || '—'}`;

  // 更新次数
  updatesEl.textContent = data.version_info?.update_count || '—';

  // 最后更新时间
  lastUpdateEl.textContent = formatTime(data.version_info?.last_updated);
}

// 渲染属性表格
function renderAttributesTable(data) {
  const tbody = document.querySelector('#reported-attrs-table tbody');
  tbody.innerHTML = '';

  const reportedAttrs = data.reported_attrs || [];
  const desiredMap = {};

  // 将期望属性转为 Map，方便查找
  (data.desired_attrs || []).forEach(attr => {
    desiredMap[attr.key] = attr.value;
  });

  reportedAttrs.forEach(attr => {
    const desiredValue = desiredMap[attr.key];
    const isChanged = attr.changed;

    const tr = document.createElement('tr');

    tr.innerHTML = `
      <td>
        <strong>${attr.label}</strong>
        <br>
        <small style="color:#999">${attr.key}</small>
      </td>
      <td>
        <span class="attr-value ${attr.type}">${attr.value_text}</span>
      </td>
      <td>
        ${desiredValue !== undefined
          ? `<span class="attr-value ${getAttributeType(desiredValue)}">${formatAttributeValue(desiredValue)}</span>`
          : '<span style="color:#ccc">—</span>'
        }
      </td>
      <td>${attr.type}</td>
      <td>
        <span class="category-tag ${attr.category}">${getCategoryLabel(attr.category)}</span>
      </td>
      <td>
        ${isChanged
          ? '<span class="status-badge changed">⚠️ 待同步</span>'
          : '<span class="status-badge synced">✓ 已同步</span>'
        }
      </td>
    `;

    tbody.appendChild(tr);
  });

  // 如果没有属性
  if (reportedAttrs.length === 0) {
    tbody.innerHTML = `
      <tr>
        <td colspan="6" style="text-align:center; color:#999; padding:40px;">
          暂无上报数据
        </td>
      </tr>
    `;
  }
}

// 渲染原始 JSON
function renderRawJSON(data) {
  const rawJsonEl = document.getElementById('raw-json-data');
  rawJsonEl.textContent = JSON.stringify({
    basic_info: data.basic_info,
    reported: data.raw_reported,
    desired: data.raw_desired,
    version_info: data.version_info,
    metadata: data.metadata
  }, null, 2);
}

// 渲染空状态
function renderEmptyShadow() {
  const container = document.getElementById('tab-shadow');
  container.innerHTML = `
    <div class="empty-state">
      <div class="empty-state-icon">📭</div>
      <div class="empty-state-text">
        该设备暂无影子数据<br>
        <small>设备可能尚未上线或未初始化影子</small>
      </div>
    </div>
  `;
}

// 辅助函数：格式化时间
function formatTime(timestamp) {
  if (!timestamp) return '—';
  return new Date(timestamp * 1000).toLocaleString('zh-CN');
}

// 辅助函数：获取值的类型
function getAttributeType(value) {
  if (typeof value === 'string') return 'string';
  if (typeof value === 'number') return 'number';
  if (typeof value === 'boolean') return 'boolean';
  return 'unknown';
}

// 辅助函数：格式化值显示
function formatAttributeValue(value) {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

// 辅助函数：获取分类标签
function getCategoryLabel(category) {
  const labels = {
    system: '系统',
    media: '媒体',
    custom: '自定义'
  };
  return labels[category] || category;
}

// 获取 Token（根据你的认证方式调整）
function getToken() {
  return localStorage.getItem('token') || '';
}

// 关闭弹窗
function closeModal() {
  document.querySelector('.device-detail-modal').style.display = 'none';
  currentDeviceSN = null;
}
```

---

### 方案 2: 使用 Vue.js 组件（如果项目使用 Vue）

```vue
<template>
  <div class="device-shadow-panel">
    <!-- 概览卡片 -->
    <div class="shadow-overview">
      <div class="overview-card" :class="{ online: shadowData.isOnline }">
        <div class="card-label">设备状态</div>
        <div class="card-value">{{ shadowData.statusText || '—' }}</div>
      </div>
      <div class="overview-card">
        <div class="card-label">版本号</div>
        <div class="card-value">v{{ shadowData.versionInfo?.currentVersion || '—' }}</div>
      </div>
      <div class="overview-card">
        <div class="card-label">更新次数</div>
        <div class="card-value">{{ shadowData.versionInfo?.updateCount || '—' }}</div>
      </div>
      <div class="overview-card">
        <div class="card-label">最后更新</div>
        <div class="card-value">{{ formatTime(shadowData.versionInfo?.lastUpdated) }}</div>
      </div>
    </div>

    <!-- 属性表格 -->
    <el-table :data="shadowData.reportedAttrs" v-loading="loading" stripe>
      <el-table-column prop="label" label="属性名称" min-width="120">
        <template #default="{ row }">
          <div>{{ row.label }}</div>
          <div style="color:#999;font-size:11px">{{ row.key }}</div>
        </template>
      </el-table-column>

      <el-table-column label="当前值" min-width="100">
        <template #default="{ row }">
          <code :class="`attr-value ${row.type}`">{{ row.valueText }}</code>
        </template>
      </el-table-column>

      <el-table-column label="期望值" min-width="100">
        <template #default="{ row }">
          <code v-if="getDesiredValue(row.key)" class="attr-value">
            {{ getDesiredValue(row.key) }}
          </code>
          <span v-else style="color:#ccc">—</span>
        </template>
      </el-table-column>

      <el-table-column prop="type" label="类型" width="80" />

      <el-table-column label="分类" width="80">
        <template #default="{ row }">
          <el-tag size="small" :type="getCategoryType(row.category)">
            {{ getCategoryLabel(row.category) }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag v-if="row.changed" type="warning" size="small">
            待同步
          </el-tag>
          <el-tag v-else type="success" size="small">
            已同步
          </el-tag>
        </template>
      </el-table-column>
    </el-table>

    <!-- 原始 JSON -->
    <el-collapse>
      <el-collapse-item title="查看原始数据">
        <pre>{{ formattedJSON }}</pre>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

const props = defineProps({
  deviceSn: String
})

const loading = ref(false)
const shadowData = ref({})

const desiredMap = computed(() => {
  const map = {}
  ;(shadowData.value.desiredAttrs || []).forEach(attr => {
    map[attr.key] = attr.value
  })
  return map
})

const formattedJSON = computed(() => {
  return JSON.stringify({
    basicInfo: shadowData.value.basicInfo,
    reported: shadowData.value.rawReported,
    desired: shadowData.value.rawDesired,
    versionInfo: shadowData.value.versionInfo
  }, null, 2)
})

const loadShadowData = async () => {
  loading.value = true
  try {
    const res = await fetch(`/api/device/shadow/detail?device_sn=${props.deviceSn}`, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    const result = await res.json()
    if (result.success) {
      shadowData.value = result.data
    }
  } catch (err) {
    console.error(err)
  } finally {
    loading.value = false
  }
}

const getDesiredValue = (key) => {
  return desiredMap.value[key]
}

const getCategoryType = (category) => {
  const types = { system: '', media: 'danger', custom: 'info' }
  return types[category] || ''
}

const getCategoryLabel = (category) => {
  const labels = { system: '系统', media: '媒体', custom: '自定义' }
  return labels[category] || category
}

const formatTime = (timestamp) => {
  if (!timestamp) return '—'
  return new Date(timestamp * 1000).toLocaleString('zh-CN')
}

onMounted(() => {
  loadShadowData()
})
</script>

<style scoped>
.device-shadow-panel {
  padding: 20px;
}

.shadow-overview {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.overview-card {
  background: #f5f7fa;
  border-radius: 8px;
  padding: 20px;
  text-align: center;
}

.overview-card.online {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.card-label {
  font-size: 12px;
  opacity: 0.8;
  margin-bottom: 8px;
}

.card-value {
  font-size: 24px;
  font-weight: bold;
}

.attr-value {
  font-family: monospace;
  font-weight: 600;
}

.attr-value.string { color: #e91e63; }
.attr-value.number { color: #2196f3; }
.attr-value.boolean { color: #4caf50; }

pre {
  background: #263238;
  color: #aed581;
  padding: 16px;
  border-radius: 4px;
  overflow-x: auto;
  font-size: 12px;
}
</style>
```

---

## 🔄 实时刷新策略

### 方案 1: 定时轮询（简单）

```javascript
let refreshInterval = null;

function startAutoRefresh() {
  // 每10秒自动刷新一次
  refreshInterval = setInterval(() => {
    if (currentDeviceSN && isShadowTabActive()) {
      loadShadowData(currentDeviceSN);
    }
  }, 10000);
}

function stopAutoRefresh() {
  if (refreshInterval) {
    clearInterval(refreshInterval);
    refreshInterval = null;
  }
}

function isShadowTabActive() {
  const tabContent = document.getElementById('tab-shadow');
  return tabContent && tabContent.style.display !== 'none';
}

// 页面可见性变化时控制刷新
document.addEventListener('visibilitychange', () => {
  if (document.hidden) {
    stopAutoRefresh();
  } else if (isShadowTabActive()) {
    startAutoRefresh();
  }
});
```

### 方案 2: WebSocket 推送（高级）

如果你已经实现了 WebSocket 服务，可以使用推送机制：

```javascript
let ws = null;

function connectWebSocket(deviceSN) {
  const wsUrl = `ws://your-domain/ws/app`;

  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    console.log('WebSocket 已连接');

    // 订阅设备状态变更
    ws.send(JSON.stringify({
      cmd: 'subscribe',
      device_sn: deviceSN,
      token: getToken()
    }));
  };

  ws.onmessage = (event) => {
    const message = JSON.parse(event.data);

    if (message.cmd === 'device_status_change') {
      // 收到状态变更通知，立即刷新
      console.log('收到设备状态变更:', message.data);
      loadShadowData(deviceSN);
    }
  };

  ws.onerror = (error) => {
    console.error('WebSocket 错误:', error);
  };

  ws.onclose = () => {
    console.log('WebSocket 已关闭');
    // 自动重连
    setTimeout(() => connectWebSocket(deviceSN), 3000);
  };
}
```

---

## 📊 数据可视化建议

### 1. 电量进度条

```javascript
function renderBatteryLevel(level) {
  let color = '#4caf50'; // 绿色
  if (level <= 20) color = '#f44336'; // 红色
  else if (level <= 50) color = '#ff9800'; // 橙色

  return `
    <div class="battery-indicator">
      <div class="battery-bar" style="width:${level}%;background:${color}">
      </div>
      <span class="battery-text">${level}%</span>
    </div>
  `;
}
```

### 2. 运行状态图标

```javascript
function renderRunState(state) {
  const icons = {
    '运行中': '▶️',
    '待机': '⏸️',
    '暂停': '⏹️',
    '离线': '⚫'
  };

  return `${icons[state] || '❓'} ${state}`;
}
```

### 3. WiFi 信号强度

```javascript
function renderWiFiRSSI(rssi) {
  if (rssi >= -50) return '📶 极强';
  if (rssi >= -60) return '📶 强';
  if (rssi >= -70) return '📶 中等';
  if (rssi >= -80) return '📶 弱';
  return '📶 极弱';
}
```

---

## ⚠️ 注意事项

### 1. 性能优化

- **防抖处理**: 用户快速切换标签时，取消之前的请求
- **缓存策略**: 相同设备的数据缓存 5 秒
- **懒加载**: 只有用户点击"状态上报"标签时才加载数据

```javascript
let currentRequest = null;

async function loadShadowDataWithDebounce(deviceSN) {
  // 取消之前的请求
  if (currentRequest) {
    currentRequest.abort();
  }

  // 创建新的请求
  currentRequest = new AbortController();

  try {
    const response = await fetch(
      `/api/device/shadow/detail?device_sn=${deviceSN}`,
      { signal: currentRequest.signal }
    );
    // 处理响应...
  } catch (error) {
    if (error.name !== 'AbortError') {
      console.error(error);
    }
  }
}
```

### 2. 错误处理

- **网络错误**: 显示重试按钮
- **权限不足**: 提示用户重新登录
- **设备不存在**: 显示友好的空状态
- **数据格式异常**: 显示原始 JSON 供调试

### 3. 安全考虑

- **敏感数据脱敏**: 如果有 IP、MAC 等隐私信息，考虑部分隐藏
  ```javascript
  function maskIP(ip) {
    if (!ip) return '—';
    const parts = ip.split('.');
    return parts.length === 4
      ? `${parts[0]}.${parts[1]}.*.*`
      : ip;
  }
  ```

- **操作审计**: 记录谁在什么时间查看了哪个设备的影子

---

## 🧪 测试用例

### 测试场景 1: 正常情况

```javascript
async function testNormalCase() {
  const response = await fetch('/api/device/shadow/detail?device_sn=AUSP2605000001');
  const data = await response.json();

  console.assert(data.success === true, '应该成功');
  console.assert(data.data.found === true, '应该找到影子');
  console.assert(data.data.reported_attrs.length > 0, '应该有上报属性');
  console.log('✅ 测试通过');
}
```

### 测试场景 2: 设备不存在

```javascript
async function testNotFoundCase() {
  const response = await fetch('/api/device/shadow/detail?device_sn=NONEXISTENT');
  const data = await response.json();

  console.assert(data.success === true, '应该成功');
  console.assert(data.data.found === false, '应该返回 found=false');
  console.log('✅ 测试通过');
}
```

### 测试场景 3: 参数缺失

```javascript
async function testMissingParam() {
  const response = await fetch('/api/device/shadow/detail');
  const data = await response.json();

  console.assert(data.success === false, '应该失败');
  console.assert(data.code === 400, '应该返回400');
  console.log('✅ 测试通过');
}
```

---

## 📝 集成检查清单

在完成集成后，请逐项确认：

- [ ] 接口调用正常，返回正确的数据结构
- [ ] 基础信息正确显示（状态、版本号等）
- [ ] 属性表格完整展示所有字段
- [ ] 中文标签正确翻译
- [ ] 类型标识颜色区分明显
- [ ] 分类标签显示准确
- [ ] "待同步"标记正确高亮不同属性
- [ ] 原始 JSON 可折叠查看
- [ ] 加载状态显示友好
- [ ] 错误状态提示清晰
- [ ] 空状态文案合理
- [ ] 定时刷新功能正常
- [ ] 切换标签时不会重复请求
- [ ] 页面关闭时清理定时器
- [ ] WebSocket 连接/断开正常（如适用）
- [ ] 性能测试通过（大量属性时不卡顿）
- [ ] 兼容主流浏览器（Chrome/Firefox/Safari）

---

## 🆘 常见问题

### Q1: 为什么有些属性显示"—"？

**A**: 可能原因：
1. 设备还未上报该属性
2. 属性值为空字符串或 null
3. 属性被过滤了（如密码等敏感信息）

### Q2: "待同步"是什么意思？

**A**: 表示设备的**当前值**与**平台期望值**不一致。例如：
- 当前音量：80
- 期望音量：90
- 这意味着平台希望设备将音量调到 90，但设备还是 80

### Q3: 如何手动触发设备同步？

**A**: 可以在下发期望值后，等待设备主动拉取并执行。或者通过指令下发的方式强制同步。

### Q4: 数据多久更新一次？

**A**: 取决于：
1. 设备的上报频率（通常 30秒~5分钟）
2. 平台下发的频率
3. 是否开启了实时推送（WebSocket）

### Q5: 可以导出影子数据吗？

**A**: 可以！点击"查看原始数据"，复制 JSON 即可。后续可以添加"导出 Excel/CSV"按钮。

---

## 📞 技术支持

如有问题，请联系：
- **开发团队**: Dev Team
- **文档最后更新**: 2024-01-01
- **相关文档**:
  - [API 文档](./admin_shadow_api.md)
  - [Admin 后台示例](./admin_shadow_examples.md)

---

**祝集成顺利！🎉**
