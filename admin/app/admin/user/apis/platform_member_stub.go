package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/api"
)

// PlatformMemberStub 会员等级/权益等扩展接口占位，避免前端 404（数据为空时可先运营配置数据库后再接实装）
type PlatformMemberStub struct {
	api.Api
}

func (e PlatformMemberStub) emptyList(c *gin.Context, msg string) {
	e.MakeContext(c)
	e.OK(gin.H{"list": []interface{}{}, "total": 0}, msg)
}

// UserMembersList GET /platform-member/user-members
func (e PlatformMemberStub) UserMembersList(c *gin.Context) {
	e.emptyList(c, "ok")
}

// UserMembersSummary GET /platform-member/user-members/summary
func (e PlatformMemberStub) UserMembersSummary(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{
		"total_users":   0,
		"active_members": 0,
		"expiring_soon": 0,
	}, "ok")
}

// Levels GET /platform-member/levels
func (e PlatformMemberStub) Levels(c *gin.Context) {
	e.emptyList(c, "ok")
}

// Benefits GET /platform-member/benefits
func (e PlatformMemberStub) Benefits(c *gin.Context) {
	e.emptyList(c, "ok")
}

// LevelBenefits GET /platform-member/level-benefits
func (e PlatformMemberStub) LevelBenefits(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{"benefit_codes": []string{}}, "ok")
}

// SetLevelBenefits PUT /platform-member/level-benefits/:levelCode
func (e PlatformMemberStub) SetLevelBenefits(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{"saved": false}, "ok")
}

// UserMember GET /platform-member/user-member
func (e PlatformMemberStub) UserMember(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{}, "ok")
}

// UpsertUserMember PUT /platform-member/user-member
func (e PlatformMemberStub) UpsertUserMember(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{}, "ok")
}

// UserMemberDetail GET /platform-member/user-member/detail
func (e PlatformMemberStub) UserMemberDetail(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{}, "ok")
}

// BatchUserMembers POST /platform-member/user-member/batch
func (e PlatformMemberStub) BatchUserMembers(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{"ok": true}, "ok")
}

// UpsertLevel POST/PUT /platform-member/levels
func (e PlatformMemberStub) UpsertLevel(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{"id": 0}, "ok")
}

// DeleteLevel DELETE /platform-member/levels/:id
func (e PlatformMemberStub) DeleteLevel(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{}, "ok")
}

// BatchLevels POST /platform-member/levels/batch
func (e PlatformMemberStub) BatchLevels(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{}, "ok")
}

// ReorderLevels POST /platform-member/levels/reorder
func (e PlatformMemberStub) ReorderLevels(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{}, "ok")
}

// UpsertBenefit POST/PUT /platform-member/benefits
func (e PlatformMemberStub) UpsertBenefit(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{"id": 0}, "ok")
}

// DeleteBenefit DELETE /platform-member/benefits/:id
func (e PlatformMemberStub) DeleteBenefit(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{}, "ok")
}

// BatchBenefits POST /platform-member/benefits/batch
func (e PlatformMemberStub) BatchBenefits(c *gin.Context) {
	e.MakeContext(c)
	e.OK(gin.H{}, "ok")
}
