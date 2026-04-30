-- 修复 083 在部分 Windows/psql 管道编码下误写入的 dept_name（字面量 ?）。
-- 使用 UTF-8 十六进制，避免迁移文件编码再次损坏中文。
SET search_path TO public;

UPDATE sys_dept SET dept_name = convert_from(decode('e680bbe7bb8fe79086e58a9ee585ace5aea4', 'hex'), 'UTF8') WHERE dept_id = 100;
UPDATE sys_dept SET dept_name = convert_from(decode('e68a80e69cafe983a8', 'hex'), 'UTF8') WHERE dept_id = 101;
UPDATE sys_dept SET dept_name = convert_from(decode('e5898de7abafe7bb84', 'hex'), 'UTF8') WHERE dept_id = 102;
UPDATE sys_dept SET dept_name = convert_from(decode('e5908ee7abafe7bb84', 'hex'), 'UTF8') WHERE dept_id = 103;
UPDATE sys_dept SET dept_name = convert_from(decode('e8bf90e7bbb4e7bb84', 'hex'), 'UTF8') WHERE dept_id = 104;
UPDATE sys_dept SET dept_name = convert_from(decode('e8bf90e890a5e983a8', 'hex'), 'UTF8') WHERE dept_id = 105;
UPDATE sys_dept SET dept_name = convert_from(decode('e5b882e59cbae7bb84', 'hex'), 'UTF8') WHERE dept_id = 106;
UPDATE sys_dept SET dept_name = convert_from(decode('e5aea2e69c8de7bb84', 'hex'), 'UTF8') WHERE dept_id = 107;
UPDATE sys_dept SET dept_name = convert_from(decode('e8b4a2e58aa1e983a8', 'hex'), 'UTF8') WHERE dept_id = 108;
