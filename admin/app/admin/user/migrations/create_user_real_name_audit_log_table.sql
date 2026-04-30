-- 实名认证审核日志表
CREATE TABLE IF NOT EXISTS user_real_name_audit_log (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL COMMENT '用户 ID',
    auth_id BIGINT NOT NULL COMMENT '实名认证记录 ID',
    operator_id BIGINT NOT NULL COMMENT '操作员 ID',
    action VARCHAR(20) NOT NULL COMMENT '操作类型：audit',
    old_status SMALLINT NOT NULL COMMENT '审核前状态',
    new_status SMALLINT NOT NULL COMMENT '审核后状态',
    remark TEXT COMMENT '审核备注',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_audit_log_user_id ON user_real_name_audit_log(user_id);
CREATE INDEX idx_audit_log_auth_id ON user_real_name_audit_log(auth_id);
CREATE INDEX idx_audit_log_operator_id ON user_real_name_audit_log(operator_id);
CREATE INDEX idx_audit_log_created_at ON user_real_name_audit_log(created_at);

-- 注释
COMMENT ON TABLE user_real_name_audit_log IS '实名认证审核日志表';
COMMENT ON COLUMN user_real_name_audit_log.user_id IS '用户 ID';
COMMENT ON COLUMN user_real_name_audit_log.auth_id IS '实名认证记录 ID';
COMMENT ON COLUMN user_real_name_audit_log.operator_id IS '操作员 ID';
COMMENT ON COLUMN user_real_name_audit_log.action IS '操作类型';
COMMENT ON COLUMN user_real_name_audit_log.old_status IS '审核前状态';
COMMENT ON COLUMN user_real_name_audit_log.new_status IS '审核后状态';
COMMENT ON COLUMN user_real_name_audit_log.remark IS '审核备注';
