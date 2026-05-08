# SQL 查询修复说明

## 🐛 问题描述

**错误信息**: `pq: 无法确定参数 $2 的数据类型 (42P18)`

**原因**: 在动态构建 SQL 查询时，参数位置计算错误，导致 PostgreSQL 无法正确推断参数类型。

## 🔧 修复方案

### 修复前（错误代码）

```go
// 动态构建 WHERE 子句
whereClause := "WHERE user_id = $1"
args := []interface{}{userID}
argCount := 1

if req.Status != "" {
    argCount++
    whereClause += fmt.Sprintf(" AND status = $%d", argCount)
    args = append(args, req.Status)
}

// 查询总数
countQuery := fmt.Sprintf(`
    SELECT COUNT(*) FROM audio_separation_tasks
    %s
`, whereClause)

// 查询列表 - 问题在这里！
argCount++
limitOffsetClause := fmt.Sprintf("LIMIT $%d OFFSET $%d", argCount+1, argCount+2)
query := fmt.Sprintf(`
    SELECT ... FROM audio_separation_tasks
    %s
    ORDER BY created_at DESC
    %s
`, whereClause, limitOffsetClause)

args = append(args, req.PageSize, offset)
```

**问题**: 
- 当 `req.Status` 为空时，`argCount` 为 1，但 LIMIT 使用了 `$2` 和 `$3`
- 当 `req.Status` 不为空时，`argCount` 为 2，但 LIMIT 使用了 `$3` 和 `$4`
- 参数位置混乱，PostgreSQL 无法推断类型

### 修复后（正确代码）

```go
// 查询总数 - 分开处理
var countQuery string
var countArgs []interface{}

if req.Status != "" {
    countQuery = `SELECT COUNT(*) FROM audio_separation_tasks WHERE user_id = $1 AND status = $2`
    countArgs = []interface{}{userID, req.Status}
} else {
    countQuery = `SELECT COUNT(*) FROM audio_separation_tasks WHERE user_id = $1`
    countArgs = []interface{}{userID}
}

// 查询列表 - 分开处理
var query string
var args []interface{}

if req.Status != "" {
    query = `
        SELECT ... FROM audio_separation_tasks
        WHERE user_id = $1 AND status = $2
        ORDER BY created_at DESC
        LIMIT $3 OFFSET $4
    `
    args = []interface{}{userID, req.Status, req.PageSize, offset}
} else {
    query = `
        SELECT ... FROM audio_separation_tasks
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `
    args = []interface{}{userID, req.PageSize, offset}
}
```

**优点**:
- ✅ 参数位置明确，不会混淆
- ✅ 类型明确，PostgreSQL 可以正确推断
- ✅ 代码清晰，易于维护

## 📊 修复后的 SQL 语句

### 无状态筛选

```sql
-- 查询总数
SELECT COUNT(*) FROM audio_separation_tasks WHERE user_id = $1

-- 查询列表
SELECT 
    task_id, content_id, audio_url, status, progress, message, result_url,
    created_at, updated_at
FROM audio_separation_tasks
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
```

**参数**:
- `$1`: userID (BIGINT)
- `$2`: req.PageSize (INTEGER)
- `$3`: offset (INTEGER)

### 有状态筛选

```sql
-- 查询总数
SELECT COUNT(*) FROM audio_separation_tasks WHERE user_id = $1 AND status = $2

-- 查询列表
SELECT 
    task_id, content_id, audio_url, status, progress, message, result_url,
    created_at, updated_at
FROM audio_separation_tasks
WHERE user_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4
```

**参数**:
- `$1`: userID (BIGINT)
- `$2`: req.Status (VARCHAR)
- `$3`: req.PageSize (INTEGER)
- `$4`: offset (INTEGER)

## ✅ 测试验证

### 测试 1: 无筛选条件

**请求**:
```bash
GET http://localhost:8004/api/v1/inference/tasks
Authorization: Bearer {token}
```

**预期 SQL**:
```sql
SELECT COUNT(*) FROM audio_separation_tasks WHERE user_id = 123
SELECT ... WHERE user_id = 123 ORDER BY created_at DESC LIMIT 20 OFFSET 0
```

**预期结果**: 返回用户 123 的所有 3 个任务

### 测试 2: 状态筛选

**请求**:
```bash
GET http://localhost:8004/api/v1/inference/tasks?status=completed
Authorization: Bearer {token}
```

**预期 SQL**:
```sql
SELECT COUNT(*) FROM audio_separation_tasks WHERE user_id = 123 AND status = 'completed'
SELECT ... WHERE user_id = 123 AND status = 'completed' ORDER BY created_at DESC LIMIT 20 OFFSET 0
```

**预期结果**: 返回用户 123 的已完成任务

## 🔍 调试技巧

### 1. 打印 SQL 和参数

```go
log.Printf("Query: %s", query)
log.Printf("Args: %+v", args)
```

### 2. 在数据库中手动执行

```sql
-- 复制打印的 SQL，替换参数
SELECT 
    task_id, content_id, audio_url, status, progress, message, result_url,
    created_at, updated_at
FROM audio_separation_tasks
WHERE user_id = 123
ORDER BY created_at DESC
LIMIT 20 OFFSET 0;
```

### 3. 检查参数类型

```go
log.Printf("userID type: %T", userID)           // int64
log.Printf("PageSize type: %T", req.PageSize)   // int
log.Printf("offset type: %T", offset)           // int
log.Printf("Status type: %T", req.Status)       // string
```

## 📝 最佳实践

### 1. 避免动态构建 SQL

❌ **不推荐**:
```go
whereClause := "WHERE user_id = $1"
if status != "" {
    whereClause += fmt.Sprintf(" AND status = $%d", argCount)
}
```

✅ **推荐**:
```go
if status != "" {
    query = `SELECT * FROM table WHERE user_id = $1 AND status = $2`
    args = []interface{}{userID, status}
} else {
    query = `SELECT * FROM table WHERE user_id = $1`
    args = []interface{}{userID}
}
```

### 2. 使用类型明确的参数

❌ **不推荐**:
```go
args := []interface{}{userID}  // userID 是 int64
args = append(args, pageSize)   // pageSize 是 int
```

✅ **推荐**:
```go
args := []interface{}{
    int64(userID),      // 明确类型
    int32(pageSize),    // 明确类型
}
```

### 3. 使用 SQL Builder

对于复杂的查询，考虑使用 SQL Builder 库：

```go
import "github.com/Masterminds/squirrel"

query, args, err := squirrel.
    Select("task_id", "content_id", "audio_url", "status", "progress", "message", "result_url", "created_at", "updated_at").
    From("audio_separation_tasks").
    Where(squirrel.Eq{"user_id": userID}).
    OrderBy("created_at DESC").
    Limit(uint64(pageSize)).
    Offset(uint64(offset)).
    ToSql()
```

## 🎯 总结

**问题根源**: 动态构建 SQL 时参数位置计算错误

**解决方案**: 分开处理不同情况，使用明确的 SQL 语句和参数

**修复文件**: `internal/logic/task_query_logic.go`

**测试方法**: 在 Apifox 中发送 GET 请求查询任务列表

现在查询应该可以正常工作了！🎉
