package main

import (
	"log"

	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// 仅用于本地评估：给 sqlite 库塞一个 admin/123456 账号。
func main() {
	db, err := gorm.Open(sqlite.Open("./gva.db.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := db.AutoMigrate(&system.SysAuthority{}, &system.SysUser{}); err != nil {
		log.Fatal(err)
	}

	var cnt int64
	db.Model(&system.SysUser{}).Where("username = ?", "admin").Count(&cnt)
	if cnt > 0 {
		log.Println("admin 已存在，跳过")
		return
	}

	db.FirstOrCreate(&system.SysAuthority{}, system.SysAuthority{AuthorityId: 888, AuthorityName: "超级管理员"})
	u := system.SysUser{
		Username:    "admin",
		Password:    utils.BcryptHash("123456"),
		NickName:    "超级管理员",
		AuthorityId: 888,
	}
	if err := db.Create(&u).Error; err != nil {
		log.Fatal(err)
	}
	log.Println("已创建 admin / 123456")
}
