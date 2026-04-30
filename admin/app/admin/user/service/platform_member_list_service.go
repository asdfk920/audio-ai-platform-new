package service

import (
	"errors"
	"fmt"
	"time"

	log "github.com/go-admin-team/go-admin-core/logger"
	"github.com/go-admin-team/go-admin-core/sdk/service"
	"gorm.io/gorm"

	"go-admin/app/admin/user/service/dto"
	"go-admin/common/actions"
	cDto "go-admin/common/dto"
)

type PlatformMemberListService struct {
	service.Service
}

// GetPage 获取会员列表
func (s *PlatformMemberListService) GetPage(req *dto.PlatformMemberListReq, p *actions.DataPermission, list *[]dto.PlatformMemberListItem, count *int64) error {
	var err error

	// 多表关联查询
	err = s.Orm.Debug().
		Table("users").
		Select("users.user_id, users.mobile, users.nickname, users.avatar, users.member_level, users.status, users.created_at, users.updated_at, um.expire_at, um.status as member_status, um.created_at as start_time").
		Joins("LEFT JOIN user_member um ON users.user_id = um.user_id").
		Scopes(
			cDto.MakeCondition(req.GetNeedSearch()),
			cDto.Paginate(req.GetPageSize(), req.GetPageIndex()),
		).
		Find(list).Limit(-1).Offset(-1).
		Count(count).Error

	if err != nil {
		log.Errorf("db error: %s", err)
		return err
	}

	// 补充额外信息：手机号脱敏、会员等级名称、剩余天数、绑定设备数等
	for i := range *list {
		item := &(*list)[i]

		// 手机号脱敏
		item.Mobile = s.maskMobile(item.Mobile)

		// 会员等级名称
		item.MemberLevelName = s.getMemberLevelName(item.MemberLevel)

		// 转换时间戳（从 start_time 字段获取）
		// CreateTime 已经在 SQL 查询中映射为 start_time

		// 计算剩余天数
		if item.ExpireTime > 0 {
			now := time.Now().Unix()
			item.RemainingDays = (item.ExpireTime - now) / 86400
			if item.RemainingDays < 0 {
				item.RemainingDays = 0
			}
		}

		// 查询绑定设备数量
		deviceCount, err := s.getBindDeviceCount(item.UserId)
		if err != nil {
			log.Errorf("查询用户 %d 绑定设备数失败：%v", item.UserId, err)
		}
		item.BindDeviceCount = deviceCount

		// 会员状态转换（如果数据库状态与过期时间不一致，以过期时间为准）
		if item.ExpireTime > 0 && item.ExpireTime < time.Now().Unix() {
			item.MemberStatus = 1 // 过期
		}
	}

	return nil
}

// GetDetail 获取会员详情（全量信息）
func (s *PlatformMemberListService) GetDetail(userId int64) (*dto.PlatformMemberDetailResp, error) {
	var member dto.PlatformMemberDetailResp

	// 1. 查询用户基础信息和会员信息
	err := s.Orm.Table("users").
		Select("users.user_id, users.mobile, users.nickname, users.avatar, users.member_level, users.status as user_status").
		Joins("LEFT JOIN user_member um ON users.user_id = um.user_id").
		Select("users.*, um.expire_at, um.status as member_status, um.created_at as start_time, um.updated_at as last_renewal_time, um.auto_renew").
		Where("users.user_id = ?", userId).
		First(&member).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, fmt.Errorf("查询会员详情失败：%w", err)
	}

	// 2. 手机号脱敏
	member.Mobile = s.maskMobile(member.Mobile)

	// 3. 会员等级名称
	member.MemberLevelName = s.getMemberLevelName(member.MemberLevel)

	// 4. 计算剩余天数和即将到期状态
	if member.ExpireTime > 0 {
		now := time.Now().Unix()
		member.RemainingDays = (member.ExpireTime - now) / 86400
		if member.RemainingDays < 0 {
			member.RemainingDays = 0
		}
		// 判断是否即将到期（7 天内）
		member.IsExpiringSoon = member.RemainingDays <= 7 && member.RemainingDays >= 0
	}

	// 5. 判断会员状态
	if member.ExpireTime > 0 && member.ExpireTime < time.Now().Unix() {
		member.MemberStatus = 1 // 过期
	}
	if member.MemberStatus == 3 {
		// 未开通会员
		member.MemberLevel = 0
		member.MemberLevelName = "普通会员"
	}

	// 6. 查询绑定设备数量
	deviceCount, err := s.getBindDeviceCount(userId)
	if err != nil {
		log.Errorf("查询用户 %d 绑定设备数失败：%v", userId, err)
	}
	member.BindDeviceCount = deviceCount
	member.CurrentBindCount = deviceCount

	// 7. 查询会员等级配置
	levelConfig := s.getMemberLevelConfig(member.MemberLevel)
	member.LevelConfig = levelConfig
	if levelConfig != nil {
		member.DeviceBindLimit = levelConfig.DeviceBindLimit
	}

	// 8. 查询会员权益使用情况
	s.fillMemberBenefits(&member, userId)

	// 9. 查询订单记录
	member.OrderRecords = s.getOrderRecords(userId)

	// 10. 查询管理员操作记录
	member.OperateRecords = s.getOperateRecords(userId)

	// 11. 查询续费记录
	member.RenewalRecords = s.getRenewalRecords(userId)

	return &member, nil
}

// Update 更新会员信息（完整业务逻辑）
func (s *PlatformMemberListService) Update(req *dto.PlatformMemberUpdateReq) (*dto.PlatformMemberUpdateResp, error) {
	// 1. 参数校验
	if req.Remark == "" {
		return nil, errors.New("操作备注不能为空")
	}

	// 校验到期时间合法性
	now := time.Now()
	if req.ExpireTime <= now.Unix() {
		return nil, errors.New("到期时间必须晚于当前时间")
	}

	// 禁止设置过长时间（超过 10 年）
	maxExpireTime := now.AddDate(10, 0, 0).Unix()
	if req.ExpireTime > maxExpireTime {
		return nil, errors.New("禁止设置超过 10 年的有效期")
	}

	// 2. 开启事务
	tx := s.Orm.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 3. 查询用户是否存在及原有会员信息
	var user struct {
		UserId      int64
		MemberLevel int32
		Status      int32
	}
	err := tx.Table("users").
		Where("user_id = ?", req.UserId).
		First(&user).Error
	if err != nil {
		tx.Rollback()
		return nil, errors.New("用户不存在")
	}

	// 校验用户状态正常
	if user.Status != 1 {
		tx.Rollback()
		return nil, errors.New("用户账号状态异常")
	}

	// 查询原有会员信息
	var oldMember struct {
		ExpireTime time.Time
		Status     int32
	}
	err = tx.Table("user_member").
		Where("user_id = ?", req.UserId).
		First(&oldMember).Error

	oldLevel := user.MemberLevel
	oldExpireTime := int64(0)
	if err == nil {
		oldExpireTime = oldMember.ExpireTime.Unix()
	}

	// 4. 判断操作类型（如果未指定）
	if req.OperationType == 0 {
		if oldExpireTime == 0 {
			req.OperationType = 1 // 开通
		} else if req.MemberLevel > oldLevel {
			req.OperationType = 3 // 升级
		} else if req.MemberLevel < oldLevel {
			req.OperationType = 4 // 降级
		} else if req.ExpireTime > oldExpireTime {
			req.OperationType = 2 // 续费
		} else {
			req.OperationType = 5 // 延长
		}
	}

	// 5. 计算新会员周期
	var newExpireTime int64 = req.ExpireTime

	// 如果是续费/延长操作，且原会员未过期，在原有基础上叠加
	if req.OperationType == 2 || req.OperationType == 5 {
		if oldExpireTime > now.Unix() {
			// 在原到期时间上叠加天数
			if req.Days > 0 {
				newExpireTime = oldExpireTime + req.Days*86400
			}
		}
	}

	// 6. 更新用户会员等级
	err = tx.Table("users").
		Where("user_id = ?", req.UserId).
		Update("member_level", req.MemberLevel).Error
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新用户会员等级失败：%w", err)
	}

	// 7. 更新或创建会员记录
	expireTime := time.Unix(newExpireTime, 0)
	err = tx.Table("user_member").
		Where("user_id = ?", req.UserId).
		Updates(map[string]interface{}{
			"expire_at":  expireTime,
			"status":     0, // 正常
			"updated_at": time.Now(),
			"auto_renew": false, // 默认不自动续费
		}).Error
	if err != nil {
		// 如果记录不存在，创建新记录
		err = tx.Table("user_member").
			Create(map[string]interface{}{
				"user_id":    req.UserId,
				"expire_at":  expireTime,
				"status":     0,
				"created_at": time.Now(),
				"updated_at": time.Now(),
				"auto_renew": false,
			}).Error
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建会员记录失败：%w", err)
		}
	}

	// 8. 记录管理员操作日志
	err = tx.Table("member_operate_log").
		Create(map[string]interface{}{
			"user_id":         req.UserId,
			"operate_admin":   req.OperatorId,
			"operate_type":    req.OperationType,
			"old_level":       oldLevel,
			"new_level":       req.MemberLevel,
			"old_expire_time": time.Unix(oldExpireTime, 0),
			"new_expire_time": expireTime,
			"remark":          req.Remark,
			"operate_time":    time.Now(),
		}).Error
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("记录操作日志失败：%w", err)
	}

	// 9. 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败：%w", err)
	}

	// 10. 同步权益状态（异步刷新 Redis 缓存）
	go s.refreshMemberBenefits(req.UserId, req.MemberLevel)

	// 11. 封装返回结果
	resp := &dto.PlatformMemberUpdateResp{
		UserId:          req.UserId,
		MemberLevel:     req.MemberLevel,
		MemberLevelName: s.getMemberLevelName(req.MemberLevel),
		MemberStatus:    0,
		ExpireTime:      newExpireTime,
		RemainingDays:   (newExpireTime - now.Unix()) / 86400,
		OperationType:   req.OperationType,
		OldLevel:        oldLevel,
		OldExpireTime:   oldExpireTime,
		Remark:          req.Remark,
		UpdateTime:      now.Unix(),
	}

	return resp, nil
}

// getBindDeviceCount 获取用户绑定设备数量
func (s *PlatformMemberListService) getBindDeviceCount(userId int64) (int64, error) {
	var count int64

	err := s.Orm.Table("user_device_bind").
		Where("user_id = ? AND status = 1", userId).
		Count(&count).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}

	return count, nil
}

// getMemberLevelName 获取会员等级名称
func (s *PlatformMemberListService) getMemberLevelName(level int32) string {
	levelNames := map[int32]string{
		0: "普通会员",
		1: "VIP 会员",
		2: "SVIP 会员",
		3: "终身会员",
	}

	if name, ok := levelNames[level]; ok {
		return name
	}
	return "普通会员"
}

// getMemberLevelConfig 获取会员等级配置
func (s *PlatformMemberListService) getMemberLevelConfig(level int32) *dto.MemberLevelConfig {
	configs := map[int32]dto.MemberLevelConfig{
		0: {
			Level:           0,
			Name:            "普通会员",
			Color:           "#909399",
			Description:     "基础会员服务",
			DeviceBindLimit: 1,
		},
		1: {
			Level:           1,
			Name:            "VIP 会员",
			Color:           "#409EFF",
			Description:     "高级会员服务",
			DeviceBindLimit: 3,
		},
		2: {
			Level:           2,
			Name:            "SVIP 会员",
			Color:           "#E6A23C",
			Description:     "至尊会员服务",
			DeviceBindLimit: 10,
		},
		3: {
			Level:           3,
			Name:            "终身会员",
			Color:           "#F56C6C",
			Description:     "终身尊享会员",
			DeviceBindLimit: 20,
		},
	}

	if config, ok := configs[level]; ok {
		return &config
	}
	defaultConfig := configs[0]
	return &defaultConfig
}

// getRenewalRecords 查询续费记录
func (s *PlatformMemberListService) getRenewalRecords(userId int64) []dto.RenewalRecord {
	var records []dto.RenewalRecord

	err := s.Orm.Table("member_renewal_record").
		Where("user_id = ?", userId).
		Order("renewal_time DESC").
		Limit(10).
		Find(&records).Error

	if err != nil {
		log.Errorf("查询续费记录失败：%v", err)
		return []dto.RenewalRecord{}
	}

	return records
}

// fillMemberBenefits 填充会员权益信息
func (s *PlatformMemberListService) fillMemberBenefits(member *dto.PlatformMemberDetailResp, userId int64) {
	// 根据会员等级获取权益列表
	benefits := s.getBenefitListByLevel(member.MemberLevel)

	// 查询用户权益使用情况
	userBenefits := s.getUserBenefits(userId)

	// 分类权益：可用和已用
	now := time.Now().Unix()
	for _, benefit := range benefits {
		userBenefit := userBenefits[benefit.BenefitCode]

		benefitInfo := dto.BenefitInfo{
			BenefitName: benefit.BenefitName,
			BenefitCode: benefit.BenefitCode,
			TotalCount:  benefit.TotalCount,
			ExpireTime:  benefit.ExpireTime,
		}

		if userBenefit != nil {
			benefitInfo.UsedCount = userBenefit.UsedCount
			benefitInfo.RemainingCount = benefit.TotalCount - userBenefit.UsedCount
			if benefitInfo.RemainingCount <= 0 {
				benefitInfo.Status = 1 // 已用完
			} else if benefit.ExpireTime > 0 && benefit.ExpireTime < now {
				benefitInfo.Status = 2 // 已过期
			} else {
				benefitInfo.Status = 0 // 可用
			}
		} else {
			benefitInfo.UsedCount = 0
			benefitInfo.RemainingCount = benefit.TotalCount
			if benefit.ExpireTime > 0 && benefit.ExpireTime < now {
				benefitInfo.Status = 2 // 已过期
			} else {
				benefitInfo.Status = 0 // 可用
			}
		}

		if benefitInfo.Status == 0 {
			member.AvailableBenefits = append(member.AvailableBenefits, benefitInfo)
		} else {
			member.UsedBenefits = append(member.UsedBenefits, benefitInfo)
		}
	}
}

// getBenefitListByLevel 根据会员等级获取权益列表
func (s *PlatformMemberListService) getBenefitListByLevel(level int32) []dto.BenefitInfo {
	// 不同等级的权益配置
	benefitConfigs := map[int32][]dto.BenefitInfo{
		0: { // 普通会员
			{BenefitName: "基础音质", BenefitCode: "basic_audio", TotalCount: 0, ExpireTime: 0},
			{BenefitName: "每日 10 次播放", BenefitCode: "daily_play_10", TotalCount: 10, ExpireTime: 0},
		},
		1: { // VIP
			{BenefitName: "高品质音质", BenefitCode: "hq_audio", TotalCount: 0, ExpireTime: 0},
			{BenefitName: "每日 100 次播放", BenefitCode: "daily_play_100", TotalCount: 100, ExpireTime: 0},
			{BenefitName: "绑定 3 台设备", BenefitCode: "bind_3_devices", TotalCount: 3, ExpireTime: 0},
		},
		2: { // SVIP
			{BenefitName: "无损音质", BenefitCode: "lossless_audio", TotalCount: 0, ExpireTime: 0},
			{BenefitName: "无限播放", BenefitCode: "unlimited_play", TotalCount: 0, ExpireTime: 0},
			{BenefitName: "绑定 10 台设备", BenefitCode: "bind_10_devices", TotalCount: 10, ExpireTime: 0},
			{BenefitName: "离线下载", BenefitCode: "offline_download", TotalCount: 100, ExpireTime: 0},
		},
		3: { // 终身会员
			{BenefitName: "无损音质", BenefitCode: "lossless_audio", TotalCount: 0, ExpireTime: 0},
			{BenefitName: "无限播放", BenefitCode: "unlimited_play", TotalCount: 0, ExpireTime: 0},
			{BenefitName: "绑定 20 台设备", BenefitCode: "bind_20_devices", TotalCount: 20, ExpireTime: 0},
			{BenefitName: "离线下载", BenefitCode: "offline_download", TotalCount: 0, ExpireTime: 0},
			{BenefitName: "专属客服", BenefitCode: "vip_support", TotalCount: 0, ExpireTime: 0},
		},
	}

	if benefits, ok := benefitConfigs[level]; ok {
		return benefits
	}
	return benefitConfigs[0]
}

// getUserBenefits 获取用户权益使用情况
func (s *PlatformMemberListService) getUserBenefits(userId int64) map[string]*dto.BenefitInfo {
	var userBenefits []struct {
		BenefitCode string `gorm:"benefit_code"`
		UsedCount   int64  `gorm:"used_count"`
	}

	err := s.Orm.Table("user_benefit").
		Where("user_id = ?", userId).
		Find(&userBenefits).Error

	if err != nil {
		return make(map[string]*dto.BenefitInfo)
	}

	result := make(map[string]*dto.BenefitInfo)
	for _, ub := range userBenefits {
		result[ub.BenefitCode] = &dto.BenefitInfo{
			BenefitCode: ub.BenefitCode,
			UsedCount:   ub.UsedCount,
		}
	}

	return result
}

// getOrderRecords 查询订单记录
func (s *PlatformMemberListService) getOrderRecords(userId int64) []dto.OrderRecord {
	var orders []dto.OrderRecord

	err := s.Orm.Table("user_member_order").
		Where("user_id = ?", userId).
		Order("created_at DESC").
		Limit(10).
		Find(&orders).Error

	if err != nil {
		log.Errorf("查询订单记录失败：%v", err)
		return []dto.OrderRecord{}
	}

	return orders
}

// getOperateRecords 查询管理员操作记录
func (s *PlatformMemberListService) getOperateRecords(userId int64) []dto.OperateRecord {
	var records []dto.OperateRecord

	err := s.Orm.Table("member_operate_log").
		Where("user_id = ?", userId).
		Order("operate_time DESC").
		Limit(20).
		Find(&records).Error

	if err != nil {
		log.Errorf("查询操作记录失败：%v", err)
		return []dto.OperateRecord{}
	}

	return records
}

// FreezeMember 冻结会员
func (s *PlatformMemberListService) FreezeMember(req *dto.PlatformMemberFreezeReq) (*dto.PlatformMemberFreezeResp, error) {
	// 1. 参数校验
	if req.FreezeReason == "" {
		return nil, errors.New("冻结原因不能为空")
	}

	// 2. 开启事务
	tx := s.Orm.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 3. 查询用户是否存在
	var user struct {
		UserId      int64
		MemberLevel int32
		Status      int32
	}
	err := tx.Table("users").
		Where("user_id = ?", req.UserId).
		First(&user).Error
	if err != nil {
		tx.Rollback()
		return nil, errors.New("用户不存在")
	}

	// 校验用户状态正常
	if user.Status != 1 {
		tx.Rollback()
		return nil, errors.New("用户账号状态异常")
	}

	// 4. 查询会员信息
	var member struct {
		Status     int32
		ExpireTime time.Time
	}
	err = tx.Table("user_member").
		Where("user_id = ?", req.UserId).
		First(&member).Error

	// 检查是否可冻结
	if err != nil {
		tx.Rollback()
		return nil, errors.New("用户未开通会员")
	}

	// 禁止冻结已过期或已冻结的会员
	if member.Status == 1 {
		tx.Rollback()
		return nil, errors.New("会员已过期，无需冻结")
	}
	if member.Status == 2 {
		tx.Rollback()
		return nil, errors.New("会员已冻结")
	}

	// 5. 执行冻结
	now := time.Now()
	err = tx.Table("user_member").
		Where("user_id = ?", req.UserId).
		Updates(map[string]interface{}{
			"status":     2, // 冻结状态
			"updated_at": now,
		}).Error
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("冻结会员失败：%w", err)
	}

	// 6. 记录操作日志
	err = tx.Table("member_operate_log").
		Create(map[string]interface{}{
			"user_id":         req.UserId,
			"operate_admin":   req.OperatorId,
			"operate_type":    2, // 冻结操作
			"old_level":       user.MemberLevel,
			"new_level":       user.MemberLevel,
			"old_expire_time": member.ExpireTime,
			"new_expire_time": member.ExpireTime,
			"remark":          req.FreezeReason,
			"operate_time":    now,
		}).Error
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("记录操作日志失败：%w", err)
	}

	// 7. 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败：%w", err)
	}

	// 8. 立即失效会员权益（异步清除 Redis 缓存）
	go s.clearMemberBenefits(req.UserId)

	// 9. 封装返回结果
	resp := &dto.PlatformMemberFreezeResp{
		UserId:          req.UserId,
		MemberLevel:     user.MemberLevel,
		MemberLevelName: s.getMemberLevelName(user.MemberLevel),
		MemberStatus:    2, // 冻结状态
		ExpireTime:      member.ExpireTime.Unix(),
		FreezeTime:      now.Unix(),
		FreezeReason:    req.FreezeReason,
		OperatorId:      req.OperatorId,
		UpdateTime:      now.Unix(),
	}

	return resp, nil
}

// clearMemberBenefits 清除会员权益缓存
func (s *PlatformMemberListService) clearMemberBenefits(userId int64) {
	// TODO: 实现 Redis 缓存清除逻辑
	// redisKey := fmt.Sprintf("member_benefits:%d", userId)
	// s.Redis.Del(ctx, redisKey)

	log.Infof("清除用户 %d 会员权益缓存", userId)
}

// UnfreezeMember 解冻会员
func (s *PlatformMemberListService) UnfreezeMember(req *dto.PlatformMemberUnfreezeReq) (*dto.PlatformMemberUnfreezeResp, error) {
	// 1. 参数校验
	if req.UnfreezeReason == "" {
		return nil, errors.New("解冻原因不能为空")
	}

	// 2. 开启事务
	tx := s.Orm.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 3. 查询用户是否存在
	var user struct {
		UserId      int64
		MemberLevel int32
		Status      int32
	}
	err := tx.Table("users").
		Where("user_id = ?", req.UserId).
		First(&user).Error
	if err != nil {
		tx.Rollback()
		return nil, errors.New("用户不存在")
	}

	// 校验用户状态正常
	if user.Status != 1 {
		tx.Rollback()
		return nil, errors.New("用户账号状态异常")
	}

	// 4. 查询会员信息
	var member struct {
		Status     int32
		ExpireTime time.Time
	}
	err = tx.Table("user_member").
		Where("user_id = ?", req.UserId).
		First(&member).Error

	// 检查是否可解冻
	if err != nil {
		tx.Rollback()
		return nil, errors.New("用户未开通会员")
	}

	// 仅允许已冻结的会员解冻
	if member.Status != 2 {
		tx.Rollback()
		return nil, errors.New("会员未冻结，无需解冻")
	}

	// 5. 执行解冻
	now := time.Now()
	err = tx.Table("user_member").
		Where("user_id = ?", req.UserId).
		Updates(map[string]interface{}{
			"status":     0, // 正常状态
			"updated_at": now,
		}).Error
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("解冻会员失败：%w", err)
	}

	// 6. 记录操作日志
	err = tx.Table("member_operate_log").
		Create(map[string]interface{}{
			"user_id":         req.UserId,
			"operate_admin":   req.OperatorId,
			"operate_type":    3, // 解冻操作
			"old_level":       user.MemberLevel,
			"new_level":       user.MemberLevel,
			"old_expire_time": member.ExpireTime,
			"new_expire_time": member.ExpireTime,
			"remark":          req.UnfreezeReason,
			"operate_time":    now,
		}).Error
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("记录操作日志失败：%w", err)
	}

	// 7. 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败：%w", err)
	}

	// 8. 恢复会员权益（异步刷新 Redis 缓存）
	go s.refreshMemberBenefits(req.UserId, user.MemberLevel)

	// 9. 封装返回结果
	resp := &dto.PlatformMemberUnfreezeResp{
		UserId:          req.UserId,
		MemberLevel:     user.MemberLevel,
		MemberLevelName: s.getMemberLevelName(user.MemberLevel),
		MemberStatus:    0, // 正常状态
		ExpireTime:      member.ExpireTime.Unix(),
		UnfreezeTime:    now.Unix(),
		UnfreezeReason:  req.UnfreezeReason,
		OperatorId:      req.OperatorId,
		UpdateTime:      now.Unix(),
	}

	return resp, nil
}

// refreshMemberBenefits 刷新会员权益（异步刷新 Redis 缓存）
func (s *PlatformMemberListService) refreshMemberBenefits(userId int64, memberLevel int32) {
	// TODO: 实现 Redis 缓存刷新逻辑
	// 1. 获取新等级的权益列表
	_ = s.getBenefitListByLevel(memberLevel)

	// 2. 刷新 Redis 缓存
	// redisKey := fmt.Sprintf("member_benefits:%d", userId)
	// redisData := map[string]interface{}{
	// 	"level": memberLevel,
	// 	"benefits": benefits,
	// 	"updated_at": time.Now().Unix(),
	// }
	// s.Redis.Set(ctx, redisKey, redisData, 24*time.Hour)

	log.Infof("刷新用户 %d 会员权益缓存，等级：%d", userId, memberLevel)
}

// maskMobile 手机号脱敏
func (s *PlatformMemberListService) maskMobile(mobile string) string {
	if len(mobile) < 11 {
		return mobile
	}
	return mobile[:3] + "****" + mobile[7:]
}

// SaveRightConfig 保存/更新会员权益配置
func (s *PlatformMemberListService) SaveRightConfig(req *dto.PlatformMemberRightConfigReq) (*dto.PlatformMemberRightConfigResp, error) {
	// 1. 参数校验
	if req.LevelId < 0 || req.LevelId > 3 {
		return nil, errors.New("会员等级不合法")
	}

	// 校验权益项合法性
	validRightKeys := map[string]bool{
		"device_bind_limit":  true, // 设备绑定上限
		"vip_content":        true, // 付费内容权限
		"high_quality_audio": true, // 高音质
		"spatial_audio":      true, // 空间音频
		"ota_upgrade":        true, // OTA 升级
		"download_speed":     true, // 下载速度
		"download_parallel":  true, // 下载并发数
		"ad_free":            true, // 免广告
		"cloud_storage":      true, // 云存储空间
		"exclusive_service":  true, // 专属客服
		"vip_avatar":         true, // 会员头像
		"vip_badge":          true, // 会员标识
		"early_access":       true, // 提前收听
	}

	for _, right := range req.Rights {
		if !validRightKeys[right.RightKey] {
			return nil, fmt.Errorf("非法的权益项：%s", right.RightKey)
		}

		// 校验数值合法性
		if right.RightKey == "device_bind_limit" || right.RightKey == "download_parallel" {
			if val, ok := right.RightValue.(float64); ok {
				if val < 0 {
					return nil, fmt.Errorf("%s 必须大于等于 0", right.RightKey)
				}
			}
		}
	}

	// 2. 开启事务
	tx := s.Orm.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 3. 查询原有配置
	var oldConfig struct {
		LevelId   int32
		LevelName string
		Status    int32
		Rights    string
	}
	err := tx.Table("member_level_config").
		Where("level_id = ?", req.LevelId).
		First(&oldConfig).Error

	oldRights := ""
	if err == nil {
		oldRights = oldConfig.Rights
	}

	// 4. 序列化权益配置
	rightsJSON, err := s.serializeRights(req.Rights)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("序列化权益配置失败：%w", err)
	}

	// 5. 保存或更新配置
	now := time.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 新增配置
		err = tx.Table("member_level_config").
			Create(map[string]interface{}{
				"level_id":   req.LevelId,
				"level_name": req.LevelName,
				"status":     req.Status,
				"rights":     rightsJSON,
				"created_at": now,
				"updated_at": now,
			}).Error
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建权益配置失败：%w", err)
		}
	} else {
		// 更新配置
		err = tx.Table("member_level_config").
			Where("level_id = ?", req.LevelId).
			Updates(map[string]interface{}{
				"level_name": req.LevelName,
				"status":     req.Status,
				"rights":     rightsJSON,
				"updated_at": now,
			}).Error
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("更新权益配置失败：%w", err)
		}
	}

	// 6. 记录操作日志
	err = tx.Table("member_operate_log").
		Create(map[string]interface{}{
			"operate_admin":   req.OperatorId,
			"operate_type":    4, // 权益配置操作
			"old_level":       req.LevelId,
			"new_level":       req.LevelId,
			"old_expire_time": time.Time{}, // 不适用于权益配置
			"new_expire_time": time.Time{},
			"remark":          req.Remark,
			"operate_time":    now,
			"old_rights":      oldRights,
			"new_rights":      rightsJSON,
		}).Error
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("记录操作日志失败：%w", err)
	}

	// 7. 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败：%w", err)
	}

	// 8. 刷新全局缓存（异步）
	go s.refreshGlobalRightCache()

	// 9. 封装返回结果
	resp := &dto.PlatformMemberRightConfigResp{
		LevelId:    req.LevelId,
		LevelName:  req.LevelName,
		Status:     req.Status,
		Rights:     req.Rights,
		UpdateTime: now.Unix(),
		OperatorId: req.OperatorId,
	}

	return resp, nil
}

// serializeRights 序列化权益配置为 JSON 字符串
func (s *PlatformMemberListService) serializeRights(rights []dto.MemberRightConfig) (string, error) {
	// TODO: 实现 JSON 序列化
	// return json.Marshal(rights)
	return "rights_json_placeholder", nil
}

// refreshGlobalRightCache 刷新全局权益缓存
func (s *PlatformMemberListService) refreshGlobalRightCache() {
	// TODO: 实现 Redis 全局缓存刷新
	// 清除所有会员等级的权益缓存
	log.Infof("刷新全局会员权益缓存")
}
