-- 删除 Apipost 里显示的测试用户（user1@example.com、测试用户1 等），保留库里真实数据
-- 会级联删除 user_auth 等关联表

-- 删除符合以下任一条件的测试用户：
-- 1. 邮箱为 example.com
-- 2. 手机号为 13800138001 / 13800138002 / 13800138003（Apipost 示例里的测试号）
-- 3. 昵称为 测试用户1 / 测试用户2 / 测试用户3 或 测试用户%
DELETE FROM users
WHERE email LIKE '%@example.com'
   OR mobile IN ('13800138001', '13800138002', '13800138003')
   OR nickname IN ('测试用户1', '测试用户2', '测试用户3')
   OR nickname LIKE '测试用户%';

-- 执行后可用下面语句确认剩余用户（真实数据）：
-- SELECT id, email, mobile, nickname, status FROM users;
