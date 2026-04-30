/**
 * 统一解析前端「接口根地址」，避免与业务里写死的 `/api/v1/...` 拼出重复路径导致 404。
 *
 * 错误示例（axios 会把 baseURL + url 拼在一起）：
 * - baseURL=/api + url=/api/v1/foo → /api/api/v1/foo
 * - baseURL=http://host:8000/api/v1 + url=/api/v1/foo → .../api/v1/api/v1/foo
 */
export function resolveApiBaseURL() {
  let b = (process.env.VUE_APP_BASE_API || '').trim()
  if (!b) return ''
  b = b.replace(/\/+$/, '')
  if (b === '/api' || b === '/api/v1') return ''
  if (b.endsWith('/api/v1')) {
    return b.slice(0, -7)
  }
  if (b.endsWith('/api')) {
    return b.slice(0, -4)
  }
  return b
}
