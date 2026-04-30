-- 回滚：仅删除本迁移插入的 sys_config 行（保留表结构以免破坏已有环境）
SET search_path TO public;
DELETE FROM sys_config WHERE id IN (1, 2, 3, 4, 5);
