SET search_path TO public;

DELETE FROM sys_dept WHERE dept_id IN (100, 101, 102, 103, 104, 105, 106, 107, 108);

SELECT setval(
  pg_get_serial_sequence('sys_dept', 'dept_id'),
  GREATEST((SELECT COALESCE(MAX(dept_id), 1) FROM sys_dept), 1)
);
