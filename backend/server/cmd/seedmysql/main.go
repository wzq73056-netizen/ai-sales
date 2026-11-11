package main

import (
	"os"
	"log"

	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 本地评估用：往 ai_go 库塞一个 admin/123456
func main() {
	// 数据库密码从环境变量读取，不要写进源码
	// 用法：set MYSQL_DSN=user:pass@tcp(127.0.0.1:3306)/ai_go?charset=utf8mb4&parseTime=True&loc=Local
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("请先设置环境变量 MYSQL_DSN")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
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
		NickName:    "管理员",
		AuthorityId: 888,
	}
	if err := db.Create(&u).Error; err != nil {
		log.Fatal(err)
	}
	log.Println("已创建 admin / 123456")
}
