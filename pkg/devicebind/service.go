package devicebind

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

// generateSN 生成符合规范的设备序列号
func generateSN(vendorCode, productLine string) (string, error) {
	// 验证厂商码
	if len(vendorCode) != 3 {
		return "", fmt.Errorf("厂商码必须为 3 位，当前：%s", vendorCode)
	}

	// 验证产品线
	if len(productLine) != 2 {
		return "", fmt.Errorf("产品线必须为 2 位，当前：%s", productLine)
	}

	// 生成年月（YYMM 格式）
	now := time.Now()
	year := now.Year() % 100
	month := int(now.Month())
	yearMonth := fmt.Sprintf("%02d%02d", year, month)

	// 生成流水号（00001-99999）
	serialNum, err := rand.Int(rand.Reader, big.NewInt(99999))
	if err != nil {
		return "", fmt.Errorf("生成流水号失败：%w", err)
	}
	serialNum = serialNum.Add(serialNum, big.NewInt(1))
	serialStr := fmt.Sprintf("%05d", serialNum.Int64())

	// 拼接前缀（不含校验位）
	prefix := fmt.Sprintf("%s-%s-%s-%s",
		strings.ToUpper(vendorCode),
		strings.ToUpper(productLine),
		yearMonth,
		serialStr,
	)

	// 生成校验位
	checkDigit := generateCheckDigit(prefix)

	sn := prefix + "-" + string(checkDigit)
	return sn, nil
}

// validateSN 仅校验非空；若为标准五段式 SN 则解析各字段，不做格式/校验位限制。
func validateSN(sn string) (bool, string, string, string, string, string, error) {
	sn = strings.TrimSpace(sn)
	if sn == "" {
		return false, "", "", "", "", "", fmt.Errorf("序列号不能为空")
	}
	parts := strings.Split(sn, "-")
	if len(parts) == 5 {
		return true, parts[0], parts[1], parts[2], parts[3], parts[4], nil
	}
	return true, "", "", "", "", "", nil
}

// parseSN 解析序列号，返回详细信息
func parseSN(sn string) (map[string]interface{}, error) {
	valid, vendorCode, productLine, yearMonth, serialNum, checkDigit, err := validateSN(sn)
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, fmt.Errorf("序列号无效")
	}

	result := map[string]interface{}{
		"sn":    sn,
		"valid": true,
	}
	if vendorCode != "" && productLine != "" && len(yearMonth) >= 4 {
		result["vendor_code"] = vendorCode
		result["vendor_name"] = getVendorName(vendorCode)
		result["product_line"] = productLine
		result["product_line_name"] = getProductLineName(productLine)
		result["year_month"] = yearMonth
		result["year"] = "20" + yearMonth[:2]
		result["month"] = yearMonth[2:]
		result["serial_number"] = serialNum
		result["check_digit"] = checkDigit
	}

	return result, nil
}

// getVendorName 根据厂商码获取厂商名称
func getVendorName(vendorCode string) string {
	vendorCodes := map[string]string{
		"AUD": "Audio Tech",
		"SND": "Sound Pro",
		"SPK": "Speaker Co",
		"HPH": "Headphone Inc",
		"MIC": "Mic Master",
	}
	if name, ok := vendorCodes[strings.ToUpper(vendorCode)]; ok {
		return name
	}
	return "未知厂商"
}

// getProductLineName 根据产品线代码获取产品线名称
func getProductLineName(productLine string) string {
	productLines := map[string]string{
		"SP": "Speaker",
		"HP": "Headphone",
		"SB": "SoundBar",
		"MI": "Mini",
		"PR": "Pro",
		"X1": "X1 Series",
		"X2": "X2 Series",
	}
	if name, ok := productLines[strings.ToUpper(productLine)]; ok {
		return name
	}
	return "未知产品线"
}

type Options struct {
	MaxDeviceBinds int
}

// BindError 绑定错误类型
type BindError struct {
	Code int
	Msg  string
}

func (e *BindError) Error() string {
	return e.Msg
}

func newError(code int, msg string) error {
	return &BindError{Code: code, Msg: msg}
}

type Result struct {
	UserID     int64
	DeviceID   int64
	DeviceSN   string
	DeviceName string
	BindTime   time.Time
}

type userRow struct {
	ID     int64
	Status int16
}

type deviceRow struct {
	ID     int64
	SN     string
	Status int16
}

type userDeviceBindRow struct {
	ID         int64
	UserID     int64
	DeviceID   int64
	SN         string
	Alias      string
	Status     int16
	BoundAt    time.Time
	UnboundAt  sql.NullTime
	DeviceName string
}

type deviceTableColumns struct {
	BoundUserID bool
	BoundAt     bool
	BindStatus  bool
	UpdatedAt   bool
}

type userProfileTableColumns struct {
	DeviceCount  bool
	LastBindTime bool
	UpdatedAt    bool
}

var (
	deviceColsMu  sync.RWMutex
	deviceCols    *deviceTableColumns
	profileColsMu sync.RWMutex
	profileCols   *userProfileTableColumns
)

func BindUserDevice(ctx context.Context, db *sql.DB, userID int64, deviceSN string, opts Options) (*Result, error) {
	if userID <= 0 {
		return nil, newError(401, "登录已过期或无效，请重新登录")
	}

	snNorm := normalizeSN(deviceSN)
	if snNorm == "" {
		return nil, newError(400, "设备序列号不能为空")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, newError(500, "系统繁忙，请稍后重试")
	}
	defer func() { _ = tx.Rollback() }()

	user, err := findUserByID(ctx, tx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(404, "用户不存在")
		}
		return nil, newError(500, "查询用户失败")
	}
	if user.Status != 1 {
		if user.Status == 2 {
			return nil, newError(403, "用户账号已被禁用")
		}
		if user.Status == 3 {
			return nil, newError(403, "用户账号已被封禁")
		}
		return nil, newError(403, "用户账号状态异常")
	}

	device, err := findDeviceBySN(ctx, tx, snNorm)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newError(404, "设备不存在")
		}
		return nil, newError(500, "查询设备失败")
	}
	if device.Status != 1 {
		if device.Status == 2 {
			return nil, newError(403, "设备已被禁用")
		}
		if device.Status == 3 {
			return nil, newError(403, "设备未激活，请先激活设备")
		}
		return nil, newError(403, "设备状态异常")
	}

	activeBind, err := findActiveBindByDeviceID(ctx, tx, device.ID)
	if err != nil {
		return nil, newError(500, "查询绑定状态失败")
	}
	if activeBind != nil && activeBind.UserID != userID {
		return nil, newError(403, "该设备已被其他用户绑定")
	}
	if activeBind != nil && activeBind.UserID == userID {
		deviceName := strings.TrimSpace(activeBind.DeviceName)
		if deviceName == "" {
			deviceName = snNorm
		}
		return &Result{
			UserID:     userID,
			DeviceID:   device.ID,
			DeviceSN:   snNorm,
			DeviceName: deviceName,
			BindTime:   activeBind.BoundAt,
		}, nil
	}

	maxBinds := opts.MaxDeviceBinds
	if maxBinds <= 0 {
		maxBinds = 10
	}
	bindCount, err := countUserBinds(ctx, tx, userID)
	if err != nil {
		return nil, newError(500, "查询绑定数失败")
	}
	if bindCount >= int64(maxBinds) {
		return nil, newError(400, fmt.Sprintf("已达到最大绑定设备数限制（%d 台）", maxBinds))
	}

	bindTime := time.Now()
	historyBind, err := findBindByUserAndDevice(ctx, tx, userID, device.ID)
	if err != nil {
		return nil, newError(500, "查询绑定状态失败")
	}
	if historyBind != nil {
		if err := reactivateBind(ctx, tx, userID, device.ID, snNorm, snNorm, bindTime); err != nil {
			return nil, newError(500, "创建绑定关系失败")
		}
	} else {
		if err := insertBind(ctx, tx, userID, device.ID, snNorm, snNorm); err != nil {
			return nil, newError(500, "创建绑定关系失败")
		}
	}

	if err := updateDeviceBindStatus(ctx, tx, device.ID, userID, bindTime); err != nil {
		return nil, newError(500, "更新设备状态失败")
	}
	if err := incrementUserDeviceCount(ctx, tx, userID, bindTime); err != nil {
		return nil, newError(500, "更新用户信息失败")
	}
	_ = insertBindLog(ctx, tx, userID, device.ID, snNorm, "", "bind", bindTime)

	if err := tx.Commit(); err != nil {
		return nil, newError(500, "绑定失败，请稍后重试")
	}

	return &Result{
		UserID:     userID,
		DeviceID:   device.ID,
		DeviceSN:   snNorm,
		DeviceName: snNorm,
		BindTime:   bindTime,
	}, nil
}

func normalizeSN(sn string) string {
	return strings.ToUpper(strings.TrimSpace(sn))
}

// validateSNFormat 仅校验非空（不做格式限制）
func validateSNFormat(sn string) error {
	if strings.TrimSpace(sn) == "" {
		return fmt.Errorf("设备序列号不能为空")
	}
	return nil
}

// generateCheckDigit 根据前缀生成校验位（第 15 位）
// 算法：Luhn 算法变体，将字母转换为数字后计算
func generateCheckDigit(prefix string) byte {
	sum := 0
	for i, ch := range prefix {
		var val int
		if ch >= '0' && ch <= '9' {
			val = int(ch - '0')
		} else if ch >= 'A' && ch <= 'Z' {
			val = int(ch - 'A' + 10)
		} else {
			val = i // 其他字符（如横杠）使用位置索引
		}
		sum += val
	}

	// 校验位范围：0-9, A-Z（36 进制）
	checkVal := (36 - (sum % 36)) % 36
	if checkVal < 10 {
		return byte('0' + checkVal)
	}
	return byte('A' + checkVal - 10)
}

func findUserByID(ctx context.Context, tx *sql.Tx, userID int64) (*userRow, error) {
	if userID <= 0 {
		return nil, sql.ErrNoRows
	}
	var row userRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, status
		FROM public.users
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1`, userID,
	).Scan(&row.ID, &row.Status)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func findDeviceBySN(ctx context.Context, tx *sql.Tx, sn string) (*deviceRow, error) {
	if sn == "" {
		return nil, sql.ErrNoRows
	}
	var row deviceRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, sn, status
		FROM public.device
		WHERE sn = $1
		LIMIT 1`, sn,
	).Scan(&row.ID, &row.SN, &row.Status)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func countUserBinds(ctx context.Context, tx *sql.Tx, userID int64) (int64, error) {
	var count int64
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM public.user_device_bind
		WHERE user_id = $1 AND status = 1
	`, userID).Scan(&count)
	return count, err
}

func findBindByUserAndDevice(ctx context.Context, tx *sql.Tx, userID, deviceID int64) (*userDeviceBindRow, error) {
	var row userDeviceBindRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE user_id = $1 AND device_id = $2
		LIMIT 1
	`, userID, deviceID).Scan(
		&row.ID, &row.UserID, &row.DeviceID, &row.SN, &row.Alias,
		&row.Status, &row.BoundAt, &row.UnboundAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.DeviceName = row.Alias
	return &row, nil
}

func findActiveBindByDeviceID(ctx context.Context, tx *sql.Tx, deviceID int64) (*userDeviceBindRow, error) {
	var row userDeviceBindRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE device_id = $1 AND status = 1
		LIMIT 1
	`, deviceID).Scan(
		&row.ID, &row.UserID, &row.DeviceID, &row.SN, &row.Alias,
		&row.Status, &row.BoundAt, &row.UnboundAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.DeviceName = row.Alias
	return &row, nil
}

func insertBind(ctx context.Context, tx *sql.Tx, userID, deviceID int64, sn, deviceName string) error {
	alias := truncateAlias(deviceName)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.user_device_bind
		  (user_id, device_id, sn, alias, is_default, bind_type, status, bound_at)
		VALUES ($1, $2, $3, $4, 0, 1, 1, CURRENT_TIMESTAMP)
	`, userID, deviceID, sn, alias)
	return err
}

func reactivateBind(ctx context.Context, tx *sql.Tx, userID, deviceID int64, sn, deviceName string, bindTime time.Time) error {
	alias := truncateAlias(deviceName)
	_, err := tx.ExecContext(ctx, `
		UPDATE public.user_device_bind
		SET sn = $1,
		    alias = $2,
		    status = 1,
		    bound_at = $3,
		    unbound_at = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $4 AND device_id = $5
	`, sn, alias, bindTime, userID, deviceID)
	return err
}

func insertBindLog(ctx context.Context, tx *sql.Tx, userID, deviceID int64, sn, operator, action string, actionTime time.Time) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.user_device_bind_log
		  (user_id, device_id, sn, operator, action, action_time)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, deviceID, sn, operator, action, actionTime)
	return err
}

func updateDeviceBindStatus(ctx context.Context, tx *sql.Tx, deviceID, userID int64, bindTime time.Time) error {
	cols, err := getDeviceTableColumns(ctx, tx)
	if err != nil {
		return err
	}

	sets := make([]string, 0, 4)
	args := make([]interface{}, 0, 3)
	idx := 1
	if cols.BoundUserID {
		sets = append(sets, fmt.Sprintf("bound_user_id = $%d", idx))
		args = append(args, userID)
		idx++
	}
	if cols.BoundAt {
		sets = append(sets, fmt.Sprintf("bound_at = $%d", idx))
		args = append(args, bindTime)
		idx++
	}
	if cols.BindStatus {
		sets = append(sets, "bind_status = 1")
	}
	if cols.UpdatedAt {
		sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	}
	if len(sets) == 0 {
		return nil
	}

	args = append(args, deviceID)
	query := fmt.Sprintf("UPDATE public.device SET %s WHERE id = $%d", strings.Join(sets, ", "), idx)
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

func incrementUserDeviceCount(ctx context.Context, tx *sql.Tx, userID int64, bindTime time.Time) error {
	cols, err := getUserProfileTableColumns(ctx, tx)
	if err != nil {
		return err
	}

	sets := make([]string, 0, 3)
	args := make([]interface{}, 0, 2)
	idx := 1
	if cols.DeviceCount {
		sets = append(sets, "device_count = COALESCE(device_count, 0) + 1")
	}
	if cols.LastBindTime {
		sets = append(sets, fmt.Sprintf("last_bind_time = $%d", idx))
		args = append(args, bindTime)
		idx++
	}
	if cols.UpdatedAt {
		sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	}
	if len(sets) == 0 {
		return nil
	}

	args = append(args, userID)
	query := fmt.Sprintf("UPDATE public.user_profile SET %s WHERE user_id = $%d", strings.Join(sets, ", "), idx)
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

func getDeviceTableColumns(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}) (*deviceTableColumns, error) {
	deviceColsMu.RLock()
	cached := deviceCols
	deviceColsMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	deviceColsMu.Lock()
	defer deviceColsMu.Unlock()
	if deviceCols != nil {
		return deviceCols, nil
	}

	rows, err := q.QueryContext(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'device'`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	cols := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	deviceCols = &deviceTableColumns{
		BoundUserID: hasColumn(cols, "bound_user_id"),
		BoundAt:     hasColumn(cols, "bound_at"),
		BindStatus:  hasColumn(cols, "bind_status"),
		UpdatedAt:   hasColumn(cols, "updated_at"),
	}
	return deviceCols, nil
}

func getUserProfileTableColumns(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}) (*userProfileTableColumns, error) {
	profileColsMu.RLock()
	cached := profileCols
	profileColsMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	profileColsMu.Lock()
	defer profileColsMu.Unlock()
	if profileCols != nil {
		return profileCols, nil
	}

	rows, err := q.QueryContext(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'user_profile'`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	cols := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	profileCols = &userProfileTableColumns{
		DeviceCount:  hasColumn(cols, "device_count"),
		LastBindTime: hasColumn(cols, "last_bind_time"),
		UpdatedAt:    hasColumn(cols, "updated_at"),
	}
	return profileCols, nil
}

func hasColumn(m map[string]struct{}, key string) bool {
	_, ok := m[key]
	return ok
}

func truncateAlias(alias string) string {
	runes := []rune(alias)
	if len(runes) > 32 {
		return string(runes[:32])
	}
	return alias
}

// GenerateSN 导出 SN 生成函数（供设备微服务使用）
// 格式：厂商码 (3 位) + 产品线 (2 位) + 年月 (4 位) + 流水号 (5 位) + 校验位 (1 位)
// 示例：AUD-SP-2605-00001-X
func GenerateSN(vendorCode, productLine string) (string, error) {
	return generateSN(vendorCode, productLine)
}

// ValidateSN 导出 SN 验证函数（供设备微服务使用）
func ValidateSN(sn string) (bool, string, string, string, string, string, error) {
	return validateSN(sn)
}

// ParseSN 导出 SN 解析函数（供设备微服务使用）
func ParseSN(sn string) (map[string]interface{}, error) {
	return parseSN(sn)
}
