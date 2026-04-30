package service

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-admin/app/admin/device/models"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

const (
	importMaxFileBytes = 10 << 20 // 10MB
	importMaxRows      = 500
)

type importFailure struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// ImportJobTemplateXLSX 生成批量导入模板（xlsx）
func (e *PlatformDeviceService) ImportJobTemplateXLSX() ([]byte, error) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	headers := []string{"sn", "product_key", "mac", "model", "device_name", "remark", "preset_secret"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	buf, err := f.WriteToBuffer()
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// CreateImportJob 保存任务并异步处理
func (e *PlatformDeviceService) CreateImportJob(tempSourcePath string, createdBy int64) (int64, error) {
	if e.Orm == nil {
		return 0, fmt.Errorf("orm nil")
	}
	f, err := excelize.OpenFile(tempSourcePath)
	if err != nil {
		_ = os.Remove(tempSourcePath)
		return 0, fmt.Errorf("无法解析表格: %w", err)
	}
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	_ = f.Close()
	if err != nil {
		_ = os.Remove(tempSourcePath)
		return 0, err
	}
	if len(rows) < 2 {
		_ = os.Remove(tempSourcePath)
		return 0, fmt.Errorf("表格至少需要表头与一行数据")
	}
	total := len(rows) - 1
	if total > importMaxRows {
		_ = os.Remove(tempSourcePath)
		return 0, fmt.Errorf("单次最多 %d 行", importMaxRows)
	}

	job := models.DeviceImportJob{
		Status:         "pending",
		Total:          total,
		TempSourcePath: tempSourcePath,
		CreatedBy:      createdBy,
	}
	if err := e.Orm.Create(&job).Error; err != nil {
		_ = os.Remove(tempSourcePath)
		return 0, err
	}
	if job.ID <= 0 {
		_ = os.Remove(tempSourcePath)
		return 0, fmt.Errorf("创建任务失败：未返回 id")
	}
	go e.processImportJob(job.ID)
	return job.ID, nil
}

func (e *PlatformDeviceService) processImportJob(jobID int64) {
	defer func() {
		if r := recover(); r != nil {
			_ = e.failImportJob(jobID, fmt.Sprintf("panic: %v", r))
		}
	}()
	if e.Orm == nil {
		return
	}
	var job models.DeviceImportJob
	if err := e.Orm.Where("id = ?", jobID).First(&job).Error; err != nil {
		return
	}
	if job.Status != "pending" {
		return
	}
	_ = e.Orm.Model(&models.DeviceImportJob{}).Where("id = ?", jobID).Updates(map[string]interface{}{
		"status": "running",
	}).Error

	f, err := excelize.OpenFile(job.TempSourcePath)
	if err != nil {
		_ = e.failImportJob(jobID, "打开表格失败: "+err.Error())
		return
	}
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	_ = f.Close()
	if err != nil || len(rows) < 1 {
		_ = e.failImportJob(jobID, "读取表格失败")
		return
	}

	idx := mapHeaderIndex(rows[0])
	if idx["sn"] < 0 || idx["product_key"] < 0 || idx["mac"] < 0 || idx["model"] < 0 {
		_ = e.failImportJob(jobID, "表头必须包含列: sn, product_key, mac, model")
		_ = os.Remove(job.TempSourcePath)
		return
	}

	resultPath := filepath.Join(os.TempDir(), fmt.Sprintf("device_import_%d_result.csv", jobID))
	outF, err := os.Create(resultPath)
	if err != nil {
		_ = e.failImportJob(jobID, "创建结果文件失败: "+err.Error())
		return
	}
	w := csv.NewWriter(outF)
	_ = w.Write([]string{"sn", "product_key", "device_secret", "mac"})

	var failures []importFailure
	success := 0
	failN := 0
	processed := 0

	for lineIdx := 1; lineIdx < len(rows); lineIdx++ {
		row := rows[lineIdx]
		excelRow := lineIdx + 1
		processed++

		sn := cell(row, idx["sn"])
		pk := cell(row, idx["product_key"])
		mac := cell(row, idx["mac"])
		model := cell(row, idx["model"])
		dname := cell(row, idx["device_name"])
		remark := cell(row, idx["remark"])
		preset := cell(row, idx["preset_secret"])

		if sn == "" && pk == "" && mac == "" && model == "" {
			continue
		}

		out, err := e.RegisterDeviceWithOptions(&ProvisionIn{
			Sn:                sn,
			ProductKey:        pk,
			Model:             model,
			Mac:               mac,
			AdminDisplayName:  dname,
			AdminRemark:       remark,
			PlainPresetSecret: preset,
			RequireMAC:        true,
			CreateBy:          int(job.CreatedBy),
		})
		if err != nil {
			failN++
			failures = append(failures, importFailure{Row: excelRow, Reason: err.Error()})
		} else {
			success++
			_ = w.Write([]string{out.Sn, out.ProductKey, out.DeviceSecret, out.Mac})
		}

		_ = e.Orm.Model(&models.DeviceImportJob{}).Where("id = ?", jobID).Updates(map[string]interface{}{
			"processed":    processed,
			"success_count": success,
			"fail_count":   failN,
		}).Error
	}
	w.Flush()
	_ = outF.Close()

	fb, _ := json.Marshal(failures)
	fin := time.Now()
	_ = e.Orm.Model(&models.DeviceImportJob{}).Where("id = ?", jobID).Updates(map[string]interface{}{
		"status":              "success",
		"failure_detail_json": string(fb),
		"result_file_path":    resultPath,
		"processed":           processed,
		"success_count":       success,
		"fail_count":          failN,
		"finished_at":         &fin,
	}).Error
	_ = os.Remove(job.TempSourcePath)
}

func (e *PlatformDeviceService) failImportJob(jobID int64, msg string) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	fin := time.Now()
	return e.Orm.Model(&models.DeviceImportJob{}).Where("id = ?", jobID).Updates(map[string]interface{}{
		"status":        "failed",
		"error_message": msg,
		"finished_at":   &fin,
	}).Error
}

func mapHeaderIndex(header []string) map[string]int {
	out := map[string]int{
		"sn": -1, "product_key": -1, "mac": -1, "model": -1,
		"device_name": -1, "remark": -1, "preset_secret": -1,
	}
	for i, h := range header {
		k := strings.TrimSpace(strings.ToLower(h))
		switch k {
		case "sn":
			out["sn"] = i
		case "product_key", "productkey":
			out["product_key"] = i
		case "mac":
			out["mac"] = i
		case "model":
			out["model"] = i
		case "device_name", "devicename", "name":
			out["device_name"] = i
		case "remark", "note":
			out["remark"] = i
		case "preset_secret", "presetsecret", "secret":
			out["preset_secret"] = i
		}
	}
	return out
}

func cell(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

// GetImportJob 查询任务（仅创建人）
func (e *PlatformDeviceService) GetImportJob(jobID int64, userID int) (*models.DeviceImportJob, []importFailure, error) {
	if e.Orm == nil {
		return nil, nil, fmt.Errorf("orm nil")
	}
	var j models.DeviceImportJob
	if err := e.Orm.Where("id = ?", jobID).First(&j).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrImportJobNotFound
		}
		return nil, nil, err
	}
	if int(j.CreatedBy) != userID {
		return nil, nil, ErrImportJobAccessDenied
	}
	var fails []importFailure
	if strings.TrimSpace(j.FailureDetailJSON) != "" {
		_ = json.Unmarshal([]byte(j.FailureDetailJSON), &fails)
	}
	return &j, fails, nil
}

// ImportResultFilePath 返回结果 CSV 路径（用于下载校验）
func (e *PlatformDeviceService) ImportResultFilePath(jobID int64, userID int) (string, error) {
	j, _, err := e.GetImportJob(jobID, userID)
	if err != nil {
		return "", err
	}
	if j.Status != "success" {
		return "", fmt.Errorf("任务未完成或失败")
	}
	if strings.TrimSpace(j.ResultFilePath) == "" {
		return "", fmt.Errorf("无结果文件")
	}
	return j.ResultFilePath, nil
}

// ImportMaxFileBytes 上传大小上限
func ImportMaxFileBytes() int64 { return importMaxFileBytes }
