-- 为设备模块相关表/字段补齐注释（使用 U& Unicode 转义，避免注释乱码）
SET search_path TO public;
SET client_encoding TO 'UTF8';

-- device
COMMENT ON TABLE device IS U&'\8bbe\5907\4e3b\8868\ff1a\5b58\50a8\8bbe\5907 SN/\4ea7\54c1Key/\5bc6\94a5/\578b\53f7/\7248\672c\7b49\552f\4e00\4fe1\606f';
COMMENT ON COLUMN device.sn IS U&'\8bbe\5907\5e8f\5217\53f7(SN)\ff0c\552f\4e00';
COMMENT ON COLUMN device.product_key IS U&'\4ea7\54c1Key\ff08\7528\4e8e\533a\5206\4ea7\54c1\578b\53f7/\4ea7\54c1\7ebf\ff09';
COMMENT ON COLUMN device.device_secret IS U&'\8bbe\5907\5bc6\94a5\ff08\9a8c\8bc1/\9274\6743\ff09';
COMMENT ON COLUMN device.firmware_version IS U&'\5f53\524d\56fa\4ef6\7248\672c';
COMMENT ON COLUMN device.hardware_version IS U&'\786c\4ef6\7248\672c';
COMMENT ON COLUMN device.model IS U&'\8bbe\5907\578b\53f7/\673a\578b';
COMMENT ON COLUMN device.mac IS U&'MAC\5730\5740';
COMMENT ON COLUMN device.ip IS U&'\6700\540e\4e00\6b21\4e0a\62a5 IP';
COMMENT ON COLUMN device.online_status IS U&'\5728\7ebf\72b6\6001\ff1a0\79bb\7ebf 1\5728\7ebf';
COMMENT ON COLUMN device.status IS U&'\8bbe\5907\72b6\6001\ff1a1\6b63\5e38 2\7981\7528 3\672a\6fc0\6d3b';
COMMENT ON COLUMN device.created_at IS U&'\521b\5efa\65f6\95f4';
COMMENT ON COLUMN device.updated_at IS U&'\66f4\65b0\65f6\95f4';

-- user_device_bind
COMMENT ON TABLE user_device_bind IS U&'\7528\6237-\8bbe\5907\7ed1\5b9a\8868\ff1aApp\626b\7801/\914d\7f51\7ed1\5b9a\8bbe\5907';
COMMENT ON COLUMN user_device_bind.user_id IS U&'\7528\6237ID';
COMMENT ON COLUMN user_device_bind.device_id IS U&'\8bbe\5907ID';
COMMENT ON COLUMN user_device_bind.sn IS U&'\8bbe\5907SN\ff08\5197\4f59\ff0c\4fbf\4e8e\68c0\7d22\ff09';
COMMENT ON COLUMN user_device_bind.alias IS U&'\8bbe\5907\522b\540d\ff08\5982\5ba2\5385\97f3\7bb1\ff09';
COMMENT ON COLUMN user_device_bind.is_default IS U&'\662f\5426\9ed8\8ba4\8bbe\5907\ff1a1\662f 0\5426';
COMMENT ON COLUMN user_device_bind.bind_type IS U&'\7ed1\5b9a\65b9\5f0f\ff1a1\84dd\7259\7ed1\5b9a 2=AP\914d\7f51';
COMMENT ON COLUMN user_device_bind.status IS U&'\72b6\6001\ff1a1\6b63\5e38 2\5df2\89e3\7ed1';
COMMENT ON COLUMN user_device_bind.bound_at IS U&'\7ed1\5b9a\65f6\95f4';
COMMENT ON COLUMN user_device_bind.unbound_at IS U&'\89e3\7ed1\65f6\95f4';
COMMENT ON COLUMN user_device_bind.created_at IS U&'\521b\5efa\65f6\95f4';
COMMENT ON COLUMN user_device_bind.updated_at IS U&'\66f4\65b0\65f6\95f4';

-- device_shadow
COMMENT ON TABLE device_shadow IS U&'\8bbe\5907\5f71\5b50\8868\ff1a\6301\4e45\5316\8bbe\5907\4e0a\62a5\72b6\6001(reported)\4e0e\671f\671b\72b6\6001(desired)';
COMMENT ON COLUMN device_shadow.device_id IS U&'\8bbe\5907ID';
COMMENT ON COLUMN device_shadow.sn IS U&'\8bbe\5907SN';
COMMENT ON COLUMN device_shadow.reported IS U&'\8bbe\5907\4e0a\62a5\72b6\6001(JSON)';
COMMENT ON COLUMN device_shadow.desired IS U&'\671f\671b\72b6\6001/\6307\4ee4(JSON)';
COMMENT ON COLUMN device_shadow.last_report_time IS U&'\6700\540e\4e00\6b21\4e0a\62a5\65f6\95f4';
COMMENT ON COLUMN device_shadow.created_at IS U&'\521b\5efa\65f6\95f4';
COMMENT ON COLUMN device_shadow.updated_at IS U&'\66f4\65b0\65f6\95f4';

-- device_state_log
COMMENT ON TABLE device_state_log IS U&'\8bbe\5907\72b6\6001\4e0a\62a5\65e5\5fd7\ff1a\5728\7ebf/\7535\91cf/\97f3\91cf/\4fe1\53f7\7b49\5386\53f2\8bb0\5f55';
COMMENT ON COLUMN device_state_log.device_id IS U&'\8bbe\5907ID';
COMMENT ON COLUMN device_state_log.sn IS U&'\8bbe\5907SN';
COMMENT ON COLUMN device_state_log.battery IS U&'\7535\91cf(0-100\7b49\ff0c\7ea6\5b9a\7531\8bbe\5907\534f\8bae)';
COMMENT ON COLUMN device_state_log.volume IS U&'\97f3\91cf';
COMMENT ON COLUMN device_state_log.online_status IS U&'\5728\7ebf\72b6\6001\ff1a0\79bb\7ebf 1\5728\7ebf';
COMMENT ON COLUMN device_state_log.network IS U&'\7f51\7edc\7c7b\578b\ff1awifi/4g \7b49';
COMMENT ON COLUMN device_state_log.rssi IS U&'\4fe1\53f7\5f3a\5ea6';
COMMENT ON COLUMN device_state_log.ip IS U&'\8bbe\5907\4e0a\62a5 IP';
COMMENT ON COLUMN device_state_log.created_at IS U&'\4e0a\62a5\65f6\95f4';

-- device_instruction
COMMENT ON TABLE device_instruction IS U&'\8bbe\5907\6307\4ee4\8868\ff1aApp\4e0b\53d1\6307\4ee4\2192\670d\52a1\7aef\2192\8bbe\5907\6267\884c\53ca\56de\6267\7ed3\679c';
COMMENT ON COLUMN device_instruction.device_id IS U&'\8bbe\5907ID';
COMMENT ON COLUMN device_instruction.sn IS U&'\8bbe\5907SN';
COMMENT ON COLUMN device_instruction.user_id IS U&'\4e0b\53d1\7528\6237ID\ff080\8868\793a\7cfb\7edf\ff09';
COMMENT ON COLUMN device_instruction.cmd IS U&'\6307\4ee4\540d\ff08\5982 restart/set_volume \7b49\ff09';
COMMENT ON COLUMN device_instruction.params IS U&'\6307\4ee4\53c2\6570(JSON)';
COMMENT ON COLUMN device_instruction.status IS U&'\72b6\6001\ff1a1\5f85\4e0b\53d1 2\5df2\53d1\9001 3\5df2\6267\884c 4\5931\8d25';
COMMENT ON COLUMN device_instruction.result IS U&'\8bbe\5907\56de\6267\7ed3\679c(JSON)';
COMMENT ON COLUMN device_instruction.created_at IS U&'\521b\5efa\65f6\95f4';
COMMENT ON COLUMN device_instruction.updated_at IS U&'\66f4\65b0\65f6\95f4';

-- ota_firmware
COMMENT ON TABLE ota_firmware IS U&'OTA\56fa\4ef6\8868\ff1a\6309 product_key + version \7ba1\7406\56fa\4ef6\53d1\5e03';
COMMENT ON COLUMN ota_firmware.product_key IS U&'\4ea7\54c1Key';
COMMENT ON COLUMN ota_firmware.version IS U&'\7248\672c\53f7';
COMMENT ON COLUMN ota_firmware.file_url IS U&'\56fa\4ef6\4e0b\8f7d\5730\5740';
COMMENT ON COLUMN ota_firmware.file_size IS U&'\6587\4ef6\5927\5c0f(\5b57\8282)';
COMMENT ON COLUMN ota_firmware.md5 IS U&'MD5\6821\9a8c';
COMMENT ON COLUMN ota_firmware.upgrade_type IS U&'\5347\7ea7\7c7b\578b\ff1a1\53ef\9009 2\5f3a\5236';
COMMENT ON COLUMN ota_firmware.publish_status IS U&'\53d1\5e03\72b6\6001\ff1a1\672a\53d1\5e03 2\5df2\53d1\5e03';
COMMENT ON COLUMN ota_firmware.release_note IS U&'\53d1\5e03\8bf4\660e';
COMMENT ON COLUMN ota_firmware.created_at IS U&'\521b\5efa\65f6\95f4';

-- ota_upgrade_task
COMMENT ON TABLE ota_upgrade_task IS U&'OTA\5347\7ea7\4efb\52a1\8868\ff1a\8bbe\5907\6309\4efb\52a1\6267\884c\5347\7ea7\8fdb\5ea6/\7ed3\679c';
COMMENT ON COLUMN ota_upgrade_task.device_id IS U&'\8bbe\5907ID';
COMMENT ON COLUMN ota_upgrade_task.sn IS U&'\8bbe\5907SN';
COMMENT ON COLUMN ota_upgrade_task.firmware_id IS U&'\5bf9\5e94 OTA\56fa\4ef6ID';
COMMENT ON COLUMN ota_upgrade_task.from_version IS U&'\5347\7ea7\524d\7248\672c';
COMMENT ON COLUMN ota_upgrade_task.to_version IS U&'\5347\7ea7\540e\7248\672c';
COMMENT ON COLUMN ota_upgrade_task.status IS U&'\72b6\6001\ff1a1\7b49\5f85 2\5347\7ea7\4e2d 3\6210\529f 4\5931\8d25';
COMMENT ON COLUMN ota_upgrade_task.progress IS U&'\8fdb\5ea6(0-100)';
COMMENT ON COLUMN ota_upgrade_task.error_msg IS U&'\9519\8bef\4fe1\606f';
COMMENT ON COLUMN ota_upgrade_task.created_at IS U&'\521b\5efa\65f6\95f4';
COMMENT ON COLUMN ota_upgrade_task.updated_at IS U&'\66f4\65b0\65f6\95f4';

-- device_certificate
COMMENT ON TABLE device_certificate IS U&'\8bbe\5907\8bc1\4e66\8868\ff08\53cc\5411TLS\ff09';
COMMENT ON COLUMN device_certificate.device_id IS U&'\8bbe\5907ID';
COMMENT ON COLUMN device_certificate.sn IS U&'\8bbe\5907SN';
COMMENT ON COLUMN device_certificate.cert IS U&'\8bbe\5907\8bc1\4e66\5185\5bb9';
COMMENT ON COLUMN device_certificate.private_key IS U&'\79c1\94a5\5185\5bb9';
COMMENT ON COLUMN device_certificate.not_before IS U&'\751f\6548\65f6\95f4';
COMMENT ON COLUMN device_certificate.not_after IS U&'\5931\6548\65f6\95f4';
COMMENT ON COLUMN device_certificate.status IS U&'\72b6\6001\ff1a1\6b63\5e38 2\8fc7\671f 3\540a\9500';
COMMENT ON COLUMN device_certificate.created_at IS U&'\521b\5efa\65f6\95f4';

-- device_event_log
COMMENT ON TABLE device_event_log IS U&'\8bbe\5907\4e8b\4ef6\65e5\5fd7\ff08\5ba1\8ba1/\6392\67e5\ff09';
COMMENT ON COLUMN device_event_log.device_id IS U&'\8bbe\5907ID';
COMMENT ON COLUMN device_event_log.sn IS U&'\8bbe\5907SN';
COMMENT ON COLUMN device_event_log.event_type IS U&'\4e8b\4ef6\7c7b\578b\ff08online/offline/upgrade/bind \7b49\ff09';
COMMENT ON COLUMN device_event_log.content IS U&'\4e8b\4ef6\63cf\8ff0';
COMMENT ON COLUMN device_event_log.extra IS U&'\6269\5c55\5b57\6bb5(JSON)';
COMMENT ON COLUMN device_event_log.created_at IS U&'\53d1\751f\65f6\95f4';

-- device_group
COMMENT ON TABLE device_group IS U&'\8bbe\5907\5206\7ec4\ff08\540e\7eed\6269\5c55\ff09';
COMMENT ON COLUMN device_group.user_id IS U&'\7528\6237ID';
COMMENT ON COLUMN device_group.name IS U&'\5206\7ec4\540d\79f0';
COMMENT ON COLUMN device_group.remark IS U&'\5907\6ce8';
COMMENT ON COLUMN device_group.created_at IS U&'\521b\5efa\65f6\95f4';

