// 用于在已有 sys_user 等平台表、但未跑完 go-admin 全量迁移时，补齐 RBAC 相关表。
// 含：部门/字典、岗位(sys_post)、角色菜单关联表 sys_role_menu（不迁移 sys_role 本体，避免与已有表冲突）。
// 用法：在 admin 目录执行 go run ./cmd/ensure-admin-tables -c config/settings.yml
package main

import (
	"fmt"
	"os"

	"github.com/go-admin-team/go-admin-core/config/source/file"
	"github.com/go-admin-team/go-admin-core/sdk"
	"github.com/go-admin-team/go-admin-core/sdk/config"
	"go-admin/app/admin/models"
	"go-admin/common/database"

	"gorm.io/gorm"
)

// sysRoleMenu 角色-菜单关联表；SysRole.GetPage 会 Preload("SysMenu")，缺表会导致 GET /api/v1/role 500。
type sysRoleMenu struct {
	RoleId int `gorm:"primaryKey;column:role_id"`
	MenuId int `gorm:"primaryKey;column:menu_id"`
}

func (sysRoleMenu) TableName() string { return "sys_role_menu" }

func main() {
	configYml := "config/settings.yml"
	for i, a := range os.Args {
		if a == "-c" && i+1 < len(os.Args) {
			configYml = os.Args[i+1]
		}
	}
	config.Setup(
		file.NewSource(file.WithPath(configYml)),
		func() {
			database.Setup()
		},
	)
	var db *gorm.DB
	if d := sdk.Runtime.GetDbByKey("*"); d != nil {
		db = d
	} else {
		for _, v := range sdk.Runtime.GetDb() {
			db = v
			break
		}
	}
	if db == nil {
		fmt.Println("no db")
		os.Exit(1)
	}
	if config.DatabaseConfig.Driver == "mysql" {
		db = db.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4")
	}
	err := db.AutoMigrate(
		&models.SysDept{},
		&models.SysDictType{},
		&models.SysDictData{},
		&models.SysPost{},
		&sysRoleMenu{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("ok: sys_dept, sys_dict_type, sys_dict_data, sys_post, sys_role_menu")
}
