package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-admin/app/admin/device/models"

	"gorm.io/gorm"
)

const maxDeviceInfoUpdateFields = 10

// ErrDeviceAdminInfoInvalid 参数或业务校验失败（对应 INVALID_FIELD / VALIDATION_ERROR）
var ErrDeviceAdminInfoInvalid = errors.New("VALIDATION_ERROR")

// DeviceInfoUpdatesPayload 管理员可修改字段（与 API updates 对齐）
type DeviceInfoUpdatesPayload struct {
	DeviceName *string                `json:"device_name"`
	Location   *string                `json:"location"`
	GroupID    *string                `json:"group_id"`
	Config     map[string]interface{} `json:"config"`
	Tags       []string               `json:"tags"`
	Remark     *string                `json:"remark"`
	Status     *int                   `json:"status"` // 0 禁用 1 启用 2 维护（维护写入 admin_config.maintenance_mode）
}

// AdminUpdateDeviceInfoIn 单设备更新入参
type AdminUpdateDeviceInfoIn struct {
	DeviceID int64  // >0 时优先
	Sn       string // 与 DeviceID 二选一
	Updates  DeviceInfoUpdatesPayload
	Operator string
	UserID   int64
	ClientIP string
}

// AdminUpdateDeviceInfoOut 单设备更新出参
type AdminUpdateDeviceInfoOut struct {
	DeviceID      int64    `json:"device_id"`
	DeviceSn      string   `json:"device_sn"`
	UpdatedFields []string `json:"updated_fields"`
	UpdatedAt     string   `json:"updated_at"`
}

func countNonEmptyUpdates(u DeviceInfoUpdatesPayload) int {
	n := 0
	if u.DeviceName != nil {
		n++
	}
	if u.Location != nil {
		n++
	}
	if u.GroupID != nil {
		n++
	}
	if u.Config != nil && len(u.Config) > 0 {
		n++
	}
	if u.Tags != nil {
		n++
	}
	if u.Remark != nil {
		n++
	}
	if u.Status != nil {
		n++
	}
	return n
}

// AdminUpdateDeviceInfo 管理员更新设备扩展信息（单台）
func (e *PlatformDeviceService) AdminUpdateDeviceInfo(in *AdminUpdateDeviceInfoIn) (*AdminUpdateDeviceInfoOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if countNonEmptyUpdates(in.Updates) == 0 {
		return nil, fmt.Errorf("updates 不能为空: %w", ErrDeviceAdminInfoInvalid)
	}
	if countNonEmptyUpdates(in.Updates) > maxDeviceInfoUpdateFields {
		return nil, fmt.Errorf("单次最多修改 %d 个顶层字段: %w", maxDeviceInfoUpdateFields, ErrDeviceAdminInfoInvalid)
	}

	var dev models.Device
	if in.DeviceID > 0 {
		if err := e.Orm.Where("id = ?", in.DeviceID).Take(&dev).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrPlatformDeviceNotFound
			}
			return nil, err
		}
	} else {
		sn := strings.TrimSpace(in.Sn)
		if sn == "" {
			return nil, ErrPlatformDeviceInvalid
		}
		snNorm := strings.ToUpper(sn)
		if err := e.Orm.Where("UPPER(TRIM(sn)) = ?", snNorm).Take(&dev).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrPlatformDeviceNotFound
			}
			return nil, err
		}
	}
	if strings.TrimSpace(dev.AdminTags) == "" {
		dev.AdminTags = "[]"
	}
	if strings.TrimSpace(dev.AdminConfig) == "" {
		dev.AdminConfig = "{}"
	}

	before := deviceAdminSnapshot(&dev)

	var updated []string
	u := in.Updates

	if u.DeviceName != nil {
		dev.AdminDisplayName = strings.TrimSpace(*u.DeviceName)
		updated = append(updated, "device_name")
	}
	if u.Location != nil {
		dev.AdminLocation = strings.TrimSpace(*u.Location)
		updated = append(updated, "location")
	}
	if u.GroupID != nil {
		dev.AdminGroupID = strings.TrimSpace(*u.GroupID)
		updated = append(updated, "group_id")
	}
	if u.Remark != nil {
		dev.AdminRemark = strings.TrimSpace(*u.Remark)
		updated = append(updated, "remark")
	}
	if u.Tags != nil {
		b, err := json.Marshal(u.Tags)
		if err != nil {
			return nil, fmt.Errorf("tags 格式无效: %w", ErrDeviceAdminInfoInvalid)
		}
		dev.AdminTags = string(b)
		updated = append(updated, "tags")
	}
	if u.Config != nil && len(u.Config) > 0 {
		merged, err := mergeAdminConfigJSON(dev.AdminConfig, u.Config)
		if err != nil {
			return nil, err
		}
		dev.AdminConfig = merged
		updated = append(updated, "config")
	}
	if u.Status != nil {
		st := *u.Status
		switch st {
		case 0:
			dev.Status = 2
			merged, err := mergeAdminConfigJSON(dev.AdminConfig, map[string]interface{}{"maintenance_mode": false})
			if err != nil {
				return nil, err
			}
			dev.AdminConfig = merged
			updated = append(updated, "status")
			updated = append(updated, "config")
		case 1:
			dev.Status = 1
			merged, err := mergeAdminConfigJSON(dev.AdminConfig, map[string]interface{}{"maintenance_mode": false})
			if err != nil {
				return nil, err
			}
			dev.AdminConfig = merged
			updated = append(updated, "status")
			updated = append(updated, "config")
		case 2:
			merged, err := mergeAdminConfigJSON(dev.AdminConfig, map[string]interface{}{"maintenance_mode": true})
			if err != nil {
				return nil, err
			}
			dev.AdminConfig = merged
			if dev.Status != 1 && dev.Status != 2 {
				dev.Status = 1
			}
			updated = append(updated, "status")
			updated = append(updated, "config")
		default:
			return nil, fmt.Errorf("status 仅支持 0/1/2: %w", ErrDeviceAdminInfoInvalid)
		}
	}

	if len(updated) == 0 {
		return nil, fmt.Errorf("无有效更新: %w", ErrDeviceAdminInfoInvalid)
	}

	after := deviceAdminSnapshot(&dev)

	if err := e.Orm.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
UPDATE device SET
  admin_display_name = ?,
  admin_remark = ?,
  admin_location = ?,
  admin_group_id = ?,
  admin_tags = ?::jsonb,
  admin_config = ?::jsonb,
  status = ?,
  update_by = ?,
  updated_at = CURRENT_TIMESTAMP
WHERE id = ?`,
			dev.AdminDisplayName,
			dev.AdminRemark,
			dev.AdminLocation,
			dev.AdminGroupID,
			dev.AdminTags,
			dev.AdminConfig,
			dev.Status,
			int(in.UserID),
			dev.Id,
		).Error; err != nil {
			return err
		}
		beforeJ, _ := json.Marshal(before)
		afterJ, _ := json.Marshal(after)
		uf, _ := json.Marshal(updated)
		log := models.DeviceAdminEditLog{
			DeviceId:      int64(dev.Id),
			Sn:            dev.Sn,
			AdminUserId:   in.UserID,
			AdminAccount:  in.Operator,
			BeforeData:    string(beforeJ),
			AfterData:     string(afterJ),
			UpdatedFields: string(uf),
			IpAddress:     in.ClientIP,
		}
		return tx.Create(&log).Error
	}); err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	return &AdminUpdateDeviceInfoOut{
		DeviceID:      int64(dev.Id),
		DeviceSn:      dev.Sn,
		UpdatedFields: updated,
		UpdatedAt:     now,
	}, nil
}

func deviceAdminSnapshot(d *models.Device) map[string]interface{} {
	var tags interface{}
	_ = json.Unmarshal([]byte(d.AdminTags), &tags)
	var cfg interface{}
	_ = json.Unmarshal([]byte(d.AdminConfig), &cfg)
	return map[string]interface{}{
		"device_name": d.AdminDisplayName,
		"location":    d.AdminLocation,
		"group_id":    d.AdminGroupID,
		"remark":      d.AdminRemark,
		"tags":        tags,
		"config":      cfg,
		"status":      d.Status,
	}
}

func mergeAdminConfigJSON(existingJSON string, patch map[string]interface{}) (string, error) {
	m := map[string]interface{}{}
	if strings.TrimSpace(existingJSON) != "" {
		if err := json.Unmarshal([]byte(existingJSON), &m); err != nil {
			return "", fmt.Errorf("现有 admin_config 非法: %w", ErrDeviceAdminInfoInvalid)
		}
	}
	for k, v := range patch {
		m[k] = v
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// AdminUpdateDeviceInfoBatchItem 批量单项
type AdminUpdateDeviceInfoBatchItem struct {
	DeviceID int64  `json:"device_id"`
	Sn       string `json:"sn"`
	Updates  DeviceInfoUpdatesPayload `json:"updates"`
}

// AdminUpdateDeviceInfoBatchOut 批量结果
type AdminUpdateDeviceInfoBatchOut struct {
	Results []AdminUpdateDeviceInfoBatchResult `json:"results"`
}

// AdminUpdateDeviceInfoBatchResult 单条结果
type AdminUpdateDeviceInfoBatchResult struct {
	DeviceID  int64                  `json:"device_id,omitempty"`
	DeviceSn  string                 `json:"device_sn,omitempty"`
	Success   bool                   `json:"success"`
	Message   string                 `json:"message,omitempty"`
	ErrorCode string                 `json:"error_code,omitempty"`
	Data      *AdminUpdateDeviceInfoOut `json:"data,omitempty"`
}

// AdminUpdateDeviceInfoBatch 批量更新（共享同一 Operator/UserID/IP）
func (e *PlatformDeviceService) AdminUpdateDeviceInfoBatch(items []AdminUpdateDeviceInfoBatchItem, operator string, userID int64, clientIP string) *AdminUpdateDeviceInfoBatchOut {
	out := &AdminUpdateDeviceInfoBatchOut{Results: make([]AdminUpdateDeviceInfoBatchResult, 0, len(items))}
	for _, it := range items {
		r := AdminUpdateDeviceInfoBatchResult{}
		o, err := e.AdminUpdateDeviceInfo(&AdminUpdateDeviceInfoIn{
			DeviceID: it.DeviceID,
			Sn:       it.Sn,
			Updates:  it.Updates,
			Operator: operator,
			UserID:   userID,
			ClientIP: clientIP,
		})
		if err != nil {
			r.Success = false
			r.Message = err.Error()
			if errors.Is(err, ErrPlatformDeviceNotFound) {
				r.ErrorCode = "DEVICE_NOT_FOUND"
			} else if errors.Is(err, ErrDeviceAdminInfoInvalid) || errors.Is(err, ErrPlatformDeviceInvalid) {
				r.ErrorCode = "VALIDATION_ERROR"
			} else {
				r.ErrorCode = "INTERNAL"
			}
		} else {
			r.Success = true
			r.DeviceID = o.DeviceID
			r.DeviceSn = o.DeviceSn
			r.Data = o
			r.Message = "设备信息更新成功"
		}
		out.Results = append(out.Results, r)
	}
	return out
}
