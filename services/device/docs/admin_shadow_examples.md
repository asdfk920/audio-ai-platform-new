# Admin 设备影子管理 - 使用示例

## 快速开始

### 1. 启动服务

```bash
cd services/device
go run device.go -f etc/device.yaml
```

服务默认运行在 `http://localhost:8000`

### 2. 测试接口

#### 查询设备影子详情

```bash
# 假设已有一个设备 AUSP2605000001
curl -X GET "http://localhost:8000/api/admin/device/shadow/detail?device_sn=AUSP2605000001"
```

**预期响应**:
```json
{
  "code": 200,
  "success": true,
  "message": "查询成功",
  "data": {
    "device_sn": "AUSP2605000001",
    "found": true,
    "reported": {
      "power": "on",
      "volume": 80
    },
    "desired": {},
    "version": 1,
    "status": "online",
    "status_text": "在线",
    "update_time": 1704067200,
    "is_online": true,
    "reported_fields_count": 2,
    "desired_fields_count": 0
  }
}
```

---

## 完整示例场景

### 场景 1: Admin 后台设备列表页面

**需求**: 展示所有设备的在线状态和基本信息

**实现步骤**:

1. **前端调用批量查询接口**:
```javascript
// 获取设备列表（假设从其他接口获取）
const deviceSNs = ['AUSP2605000001', 'AUSP2605000002', 'AUSP2605000003'];

// 批量查询影子状态
const response = await fetch('/api/admin/device/shadow/list', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    // 注意：生产环境需要添加认证头
  },
  body: JSON.stringify({ device_sns: deviceSNs })
});

const result = await response.json();

if (result.success) {
  console.log(`共 ${result.data.total} 个设备`);
  console.log(`找到 ${result.data.found_count} 个影子`);
  
  // 渲染表格
  result.data.items.forEach(item => {
    if (item.found) {
      console.log(`${item.device_sn}: ${item.status_text}`);
    } else {
      console.log(`${item.device_sn}: 影子不存在`);
    }
  });
}
```

2. **渲染表格 HTML**:
```html
<table class="device-table">
  <thead>
    <tr>
      <th>设备序列号</th>
      <th>状态</th>
      <th>版本号</th>
      <th>最后更新</th>
      <th>操作</th>
    </tr>
  </thead>
  <tbody id="device-list">
    <!-- 动态生成 -->
  </tbody>
</table>

<script>
function renderDeviceTable(items) {
  const tbody = document.getElementById('device-list');
  tbody.innerHTML = items.map(item => `
    <tr class="${item.is_online ? 'online' : 'offline'}">
      <td>${item.device_sn}</td>
      <td>
        <span class="status-badge ${item.status}">
          ${item.status_text}
        </span>
      </td>
      <td>v${item.version}</td>
      <td>${formatTime(item.update_time)}</td>
      <td>
        <button onclick="viewDetail('${item.device_sn}')">
          查看详情
        </button>
      </td>
    </tr>
  `).join('');
}

async function viewDetail(deviceSN) {
  const response = await fetch(
    `/api/admin/device/shadow/detail?device_sn=${deviceSN}`
  );
  const result = await response.json();
  
  if (result.success) {
    showDetailModal(result.data);
  } else {
    alert('查询失败: ' + result.message);
  }
}
</script>
```

---

### 场景 2: 设备详情弹窗

**需求**: 点击设备后展示完整的影子数据，包括 reported 和 desired 的所有字段

**实现代码**:

```javascript
async function showDetailModal(shadowData) {
  const modal = document.getElementById('detail-modal');
  
  modal.innerHTML = `
    <div class="modal-content">
      <h2>设备详情: ${shadowData.device_sn}</h2>
      
      <div class="status-section">
        <h3>基本信息</h3>
        <table>
          <tr><td>状态:</td><td>${shadowData.status_text}</td></tr>
          <tr><td>版本号:</td><td>${shadowData.version}</td></tr>
          <tr><td>最后更新:</td><td>${formatTime(shadowData.update_time)}</td></tr>
          <tr><td>上报字段数:</td><td>${shadowData.reported_fields_count}</td></tr>
        </table>
      </div>
      
      <div class="reported-section">
        <h3>设备上报状态 (Reported)</h3>
        <pre>${JSON.stringify(shadowData.reported, null, 2)}</pre>
      </div>
      
      <div class="desired-section">
        <h3>平台期望状态 (Desired)</h3>
        <pre>${JSON.stringify(shadowData.desired, null, 2)}</pre>
      </div>
      
      ${shadowData.metadata ? `
      <div class="metadata-section">
        <h3>元数据 (Metadata)</h3>
        <pre>${JSON.stringify(shadowData.metadata, null, 2)}</pre>
      </div>
      ` : ''}
      
      <button onclick="closeModal()">关闭</button>
    </div>
  `;
  
  modal.style.display = 'block';
}

function formatTime(timestamp) {
  if (!timestamp) return '-';
  return new Date(timestamp * 1000).toLocaleString('zh-CN');
}
```

---

### 场景 3: 监控大屏仪表盘

**需求**: 实时展示设备统计信息，包括在线率、异常数量等

**实现代码**:

```html
<div class="dashboard">
  <div class="stat-card total">
    <h3>设备总数</h3>
    <div id="total-devices" class="number">-</div>
  </div>
  
  <div class="stat-card online">
    <h3>在线设备</h3>
    <div id="online-devices" class="number">-</div>
  </div>
  
  <div class="stat-card offline">
    <h3>离线设备</h3>
    <div id="offline-devices" class="number">-</div>
  </div>
  
  <div class="stat-card abnormal">
    <h3>异常设备</h3>
    <div id="abnormal-devices" class="number">-</div>
  </div>
  
  <div class="stat-card rate">
    <h3>在线率</h3>
    <div id="online-rate" class="number">-</div>
  </div>
</div>

<style>
.dashboard {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 20px;
  margin: 20px;
}

.stat-card {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 30px;
  border-radius: 10px;
  text-align: center;
  box-shadow: 0 4px 6px rgba(0,0,0,0.1);
}

.stat-card.online { background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%); }
.stat-card.offline { background: linear-gradient(135deg, #eb3349 0%, #f45c43 100%); }
.stat-card.abnormal { background: linear-gradient(135deg, #ff512f 0%, #dd2476 100%); }

.number {
  font-size: 48px;
  font-weight: bold;
  margin-top: 10px;
}
</style>

<script>
let refreshInterval;

async function loadStats() {
  try {
    const response = await fetch('/api/admin/device/shadow/stats');
    const result = await response.json();
    
    if (result.success) {
      updateDashboard(result.data);
    } else {
      console.error('获取统计数据失败:', result.message);
    }
  } catch (error) {
    console.error('请求失败:', error);
  }
}

function updateDashboard(data) {
  animateNumber('total-devices', data.total_devices);
  animateNumber('online-devices', data.online_devices);
  animateNumber('offline-devices', data.offline_devices);
  animateNumber('abnormal-devices', data.abnormal_devices);
  document.getElementById('online-rate').textContent = 
    data.online_rate.toFixed(1) + '%';
}

function animateNumber(elementId, targetValue) {
  const element = document.getElementById(elementId);
  const startValue = parseInt(element.textContent) || 0;
  const duration = 1000; // 1秒动画
  const startTime = performance.now();
  
  function update(currentTime) {
    const elapsed = currentTime - startTime;
    const progress = Math.min(elapsed / duration, 1);
    
    const currentValue = Math.floor(
      startValue + (targetValue - startValue) * progress
    );
    
    element.textContent = currentValue.toLocaleString();
    
    if (progress < 1) {
      requestAnimationFrame(update);
    }
  }
  
  requestAnimationFrame(update);
}

// 页面加载时立即执行一次
loadStats();

// 每30秒自动刷新
refreshInterval = setInterval(loadStats, 30000);

// 页面隐藏时停止刷新，节省资源
document.addEventListener('visibilitychange', () => {
  if (document.hidden) {
    clearInterval(refreshInterval);
  } else {
    loadStats();
    refreshInterval = setInterval(loadStats, 30000);
  }
});
</script>
```

---

### 场景 4: 批量操作工具

**需求**: 运维人员需要批量检查多个设备的状态

**Python 脚本示例**:

```python
import requests
import json
from datetime import datetime

BASE_URL = "http://localhost:8000"
ADMIN_TOKEN = "your_admin_token_here"

def get_headers():
    return {
        "Authorization": f"Bearer {ADMIN_TOKEN}",
        "Content-Type": "application/json"
    }

def check_device_detail(device_sn):
    """查询单个设备详情"""
    url = f"{BASE_URL}/api/admin/device/shadow/detail"
    params = {"device_sn": device_sn}
    
    response = requests.get(url, params=params, headers=get_headers())
    result = response.json()
    
    if result["success"]:
        return result["data"]
    else:
        print(f"❌ 查询失败 [{device_sn}]: {result['message']}")
        return None

def batch_check_devices(device_sns):
    """批量查询设备状态"""
    url = f"{BASE_URL}/api/admin/device/shadow/list"
    payload = {"device_sns": device_sns}
    
    response = requests.post(url, json=payload, headers=get_headers())
    result = response.json()
    
    if not result["success"]:
        print(f"❌ 批量查询失败: {result['message']}")
        return
    
    data = result["data"]
    print(f"\n📊 统计信息:")
    print(f"   总数: {data['total']}")
    print(f"   找到: {data['found_count']}")
    print(f"   未找到: {len(data['not_found'])}")
    
    print(f"\n📋 设备状态列表:")
    print("-" * 80)
    
    for item in data["items"]:
        if item["found"]:
            status_icon = "🟢" if item["is_online"] else "🔴"
            print(f"{status_icon} {item['device_sn']:<20} | "
                  f"{item['status_text']:<6} | "
                  f"v{item['version']:<5} | "
                  f"属性数: {item['reported_fields_count']}")
        else:
            print(f"⚪  {item['device_sn']:<20} | 影子不存在")
    
    if data["not_found"]:
        print(f"\n⚠️  未找到的设备:")
        for sn in data["not_found"]:
            print(f"   - {sn}")

def get_stats():
    """获取统计数据"""
    url = f"{BASE_URL}/api/admin/device/shadow/stats"
    
    response = requests.get(url, headers=get_headers())
    result = response.json()
    
    if result["success"]:
        data = result["data"]
        print(f"\n📈 设备统计 ({datetime.now().strftime('%Y-%m-%d %H:%M:%S')})")
        print("=" * 50)
        print(f"设备总数:     {data['total_devices']:>8}")
        print(f"在线设备:     {data['online_devices']:>8} ({data['online_rate']:.1f}%)")
        print(f"离线设备:     {data['offline_devices']:>8}")
        print(f"异常设备:     {data['abnormal_devices']:>8}")
        print("=" * 50)
        
        # 简单告警逻辑
        if data["online_rate"] < 80:
            print("⚠️  警告: 在线率低于 80%!")
        if data["abnormal_devices"] > 10:
            print("⚠️  警告: 异常设备超过 10 个!")
        
        return data
    else:
        print(f"❌ 获取统计失败: {result['message']}")
        return None

# 使用示例
if __name__ == "__main__":
    import sys
    
    if len(sys.argv) < 2:
        print("用法:")
        print("  python admin_tool.py stats              # 查看统计")
        print("  python admin_tool.py check SN1 SN2 ...   # 批量检查")
        print("  python admin_tool.py detail SN           # 查看详情")
        sys.exit(1)
    
    command = sys.argv[1]
    
    if command == "stats":
        get_stats()
    
    elif command == "check":
        device_sns = sys.argv[2:]
        batch_check_devices(device_sns)
    
    elif command == "detail":
        device_sn = sys.argv[2]
        detail = check_device_detail(device_sn)
        if detail:
            print(json.dumps(detail, indent=2, ensure_ascii=False))
    
    else:
        print(f"未知命令: {command}")
```

**使用方式**:

```bash
# 查看统计
python admin_tool.py stats

# 批量检查设备
python admin_tool.py check AUSP2605000001 AUSP2605000002 AUSP2605000003

# 查看单个设备详情
python admin_tool.py detail AUSP2605000001
```

---

## 错误处理最佳实践

### 1. 前端错误处理

```javascript
async function safeAPICall(url, options = {}) {
  try {
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers
      }
    });
    
    const result = await response.json();
    
    if (!result.success) {
      throw new Error(result.message || '请求失败');
    }
    
    return result.data;
  } catch (error) {
    console.error('API 调用失败:', error);
    
    // 用户友好的错误提示
    if (error.message.includes('网络')) {
      showErrorToast('网络连接失败，请检查网络');
    } else if (error.message.includes('401')) {
      showErrorToast('登录已过期，请重新登录');
      setTimeout(() => window.location.href = '/login', 1500);
    } else {
      showErrorToast(error.message);
    }
    
    return null;
  }
}

function showErrorToast(message) {
  const toast = document.createElement('div');
  toast.className = 'error-toast';
  toast.textContent = message;
  document.body.appendChild(toast);
  
  setTimeout(() => toast.remove(), 3000);
}
```

### 2. 后端日志记录

在 [admin_shadow_handler.go](../internal/handler/admin_shadow_handler.go) 中已经包含了基本的日志，生产环境建议：

1. **添加请求ID**: 方便追踪问题
2. **记录操作人**: 安全审计需要
3. **性能监控**: 记录慢查询
4. **告警集成**: 异常情况自动通知

---

## 性能优化建议

### 1. 缓存策略

```javascript
// 简单的内存缓存实现
class APICache {
  constructor(ttl = 30000) { // 默认30秒缓存
    this.cache = new Map();
    this.ttl = ttl;
  }
  
  async get(key, fetcher) {
    const cached = this.cache.get(key);
    
    if (cached && Date.now() - cached.timestamp < this.ttl) {
      return cached.data;
    }
    
    const data = await fetcher();
    this.cache.set(key, {
      data,
      timestamp: Date.now()
    });
    
    return data;
  }
}

const apiCache = new APICache(30000);

// 使用缓存的统计查询
async function getCachedStats() {
  return apiCache.get('stats', () =>
    fetch('/api/admin/device/shadow/stats').then(r => r.json())
  );
}
```

### 2. 防抖和节流

```javascript
// 防抖：避免频繁调用
function debounce(func, wait) {
  let timeout;
  return function executedFunction(...args) {
    clearTimeout(timeout);
    timeout = setTimeout(() => func.apply(this, args), wait);
  };
}

// 节流：限制调用频率
function throttle(func, limit) {
  let inThrottle;
  return function executedFunction(...args) {
    if (!inThrottle) {
      func.apply(this, args);
      inThrottle = true;
      setTimeout(() => inThrottle = false, limit);
    }
  };
}

// 应用到搜索框
const searchInput = document.getElementById('device-search');
searchInput.addEventListener('input', debounce(async (e) => {
  const keyword = e.target.value.trim();
  if (keyword.length >= 2) {
    await searchDevices(keyword);
  }
}, 300));
```

---

## 下一步功能扩展建议

1. **分页支持**: 当设备数量很大时，需要分页查询
2. **过滤排序**: 支持按状态、时间等条件筛选和排序
3. **导出功能**: 支持导出 Excel/CSV 报表
4. **历史记录**: 查询设备状态变更历史
5. **实时推送**: 通过 WebSocket 接收设备状态变更通知
6. **批量操作**: 批量下发指令、批量重启等
7. **权限细分**: 不同管理员看到不同的数据和操作按钮

---

## 相关资源

- [API 完整文档](./admin_shadow_api.md)
- [设备影子 V2 设计文档](./device_shadow_design.md)
- [Redis Hash 数据结构](./redis_hash_structure.md)

---

**最后更新**: 2024-01-01
**维护团队**: Dev Team
**联系方式**: dev@example.com
