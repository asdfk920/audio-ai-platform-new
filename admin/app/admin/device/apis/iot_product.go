package apis

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"

	"go-admin/app/admin/device/service"
)

// IotProduct 产品线 API
type IotProduct struct {
	api.Api
}

// ListProducts GET /api/v1/platform-device/products
func (e IotProduct) ListProducts(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize < 1 {
		pageSize = 20
	}
	list, total, err := svc.ListIotProducts(service.IotProductListIn{
		Page:     page,
		PageSize: pageSize,
		Keyword:  strings.TrimSpace(c.Query("keyword")),
		Status:   strings.TrimSpace(c.Query("status")),
	})
	if err != nil {
		if errors.Is(err, service.ErrIotProductInvalidStatus) {
			e.Error(400, err, err.Error())
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "查询失败")
		return
	}
	e.PageOK(list, int(total), page, pageSize, "ok")
}

type iotProductCreateReq struct {
	ProductKey    string          `json:"productKey"`
	ProductKeyAlt string          `json:"product_key"`
	Name          string          `json:"name"`
	Category      string          `json:"category"`
	Description   string          `json:"description"`
	Communication json.RawMessage `json:"communication"`
	DeviceType    string          `json:"deviceType"`
	DeviceTypeAlt string          `json:"device_type"`
	Status        string          `json:"status"`
}

// CreateProduct POST /api/v1/platform-device/products 或 POST /api/v1/product（同处理函数）
func (e IotProduct) CreateProduct(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req iotProductCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	pk := strings.TrimSpace(req.ProductKey)
	if pk == "" {
		pk = strings.TrimSpace(req.ProductKeyAlt)
	}
	if pk == "" {
		e.Error(400, errors.New("product_key required"), "产品标识 productKey 或 product_key 必填")
		return
	}
	dt := strings.TrimSpace(req.DeviceType)
	if dt == "" {
		dt = strings.TrimSpace(req.DeviceTypeAlt)
	}
	uid := int64(user.GetUserId(c))
	row, err := svc.CreateIotProduct(service.IotProductCreateIn{
		ProductKey:    pk,
		Name:          req.Name,
		Category:      req.Category,
		Description:   req.Description,
		Communication: req.Communication,
		DeviceType:    dt,
		Status:        req.Status,
		CreateBy:      uid,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIotProductInvalidKey):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrIotProductDuplicate):
			e.Error(400, err, "产品标识已存在")
		case errors.Is(err, service.ErrIotProductInvalidStatus):
			e.Error(400, err, err.Error())
		default:
			if strings.Contains(err.Error(), "JSON") {
				e.Error(400, err, err.Error())
				return
			}
			e.Logger.Error(err)
			e.Error(500, err, "创建失败")
		}
		return
	}
	e.OK(gin.H{
		"id":          row.Id,
		"productKey":  strings.TrimSpace(row.ProductKey),
		"name":        row.Name,
		"category":    row.Category,
		"deviceType":  row.DeviceType,
		"description": row.Description,
		"status":      row.Status,
		"createdAt":   row.CreatedAt.Format("2006-01-02 15:04:05"),
	}, "创建成功")
}

// GetProduct GET /api/v1/platform-device/products/:id
func (e IotProduct) GetProduct(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		e.Error(400, err, "id 无效")
		return
	}
	out, err := svc.GetIotProduct(id)
	if err != nil {
		if errors.Is(err, service.ErrIotProductNotFound) {
			e.Error(404, err, "产品不存在")
			return
		}
		e.Error(500, err, "查询失败")
		return
	}
	e.OK(out, "ok")
}

type iotProductUpdateReq struct {
	Name          string          `json:"name"`
	Category      string          `json:"category"`
	Description   string          `json:"description"`
	Communication json.RawMessage `json:"communication"`
	DeviceType    string          `json:"deviceType"`
}

// UpdateProduct PUT /api/v1/platform-device/products/:id
func (e IotProduct) UpdateProduct(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		e.Error(400, err, "id 无效")
		return
	}
	var req iotProductUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	err = svc.UpdateIotProduct(id, service.IotProductUpdateIn{
		Name:          req.Name,
		Category:      req.Category,
		Description:   req.Description,
		Communication: req.Communication,
		DeviceType:    req.DeviceType,
		UpdateBy:      int64(user.GetUserId(c)),
	})
	if err != nil {
		if errors.Is(err, service.ErrIotProductNotFound) {
			e.Error(404, err, "产品不存在")
			return
		}
		if strings.Contains(err.Error(), "JSON") {
			e.Error(400, err, err.Error())
			return
		}
		e.Error(500, err, "更新失败")
		return
	}
	e.OK(nil, "更新成功")
}

// PublishProduct POST /api/v1/platform-device/products/:id/publish
func (e IotProduct) PublishProduct(c *gin.Context) {
	e.setProductStatus(c, true)
}

// DisableProduct POST /api/v1/platform-device/products/:id/disable
func (e IotProduct) DisableProduct(c *gin.Context) {
	e.setProductStatus(c, false)
}

func (e IotProduct) setProductStatus(c *gin.Context, publish bool) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		e.Error(400, err, "id 无效")
		return
	}
	uid := int64(user.GetUserId(c))
	if publish {
		err = svc.PublishIotProduct(id, uid)
	} else {
		err = svc.DisableIotProduct(id, uid)
	}
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIotProductNotFound):
			e.Error(404, err, "产品不存在")
		case errors.Is(err, service.ErrIotProductInvalidStatus):
			e.Error(400, err, err.Error())
		default:
			e.Error(500, err, "操作失败")
		}
		return
	}
	msg := "已禁用"
	if publish {
		msg = "已发布"
	}
	e.OK(nil, msg)
}
