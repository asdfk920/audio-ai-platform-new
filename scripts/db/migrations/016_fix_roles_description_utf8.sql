-- 修复 roles.description 中文乱码（Windows/psql client_encoding 不一致导致）
-- 使用 Unicode 转义字符串，避免客户端编码差异。
SET search_path TO public;
SET client_encoding TO 'UTF8';

UPDATE roles SET
  description = U&'\8d85\7ea7\7ba1\7406\5458\ff1a\5168\6a21\5757\ff1b\4ec5\7cfb\7edf\5185\7f6e\552f\4e00\8d26\53f7\ff0c\4e0d\5bf9\5916\5206\914d',
  updated_at = CURRENT_TIMESTAMP
WHERE name = 'super_admin' OR slug = 'super_admin';

UPDATE roles SET
  description = U&'\7cfb\7edf\7ba1\7406\5458\ff1a\7528\6237\002f\8bbe\5907\002f\5185\5bb9\002fOTA\002f\7edf\8ba1\002f\65e5\5fd7\5168\6743\9650\ff1b\4e0d\542b\7cfb\7edf\914d\7f6e',
  updated_at = CURRENT_TIMESTAMP
WHERE name = 'admin' OR slug = 'admin';

UPDATE roles SET
  description = U&'\8fd0\8425\4eba\5458\ff1a\5185\5bb9\5168\6743\9650\4e0e\6570\636e\7edf\8ba1\ff1bOTA\002f\65e5\5fd7\4ec5\67e5\770b\ff1b\4e0d\6d89\53ca\7528\6237\5b89\5168\4e0e\7cfb\7edf\914d\7f6e',
  updated_at = CURRENT_TIMESTAMP
WHERE name = 'operator' OR slug = 'operator';

UPDATE roles SET
  description = U&'\666e\901a\7528\6237\ff1a\4ec5\672c\4eba\8d26\53f7\3001\8bbe\5907\4e0e\5185\5bb9\ff1b\65e0\540e\53f0\7ba1\7406\6743\9650',
  updated_at = CURRENT_TIMESTAMP
WHERE name = 'user' OR slug = 'user';

UPDATE roles SET
  description = U&'\6e38\5ba2\ff1a\65e0\540e\53f0\80fd\529b\ff1b\4ec5\516c\5f00\7aef\6d4f\89c8\ff08\4e1a\52a1\4fa7\81ea\884c\9650\5236\ff09',
  updated_at = CURRENT_TIMESTAMP
WHERE name = 'guest' OR slug = 'guest';

