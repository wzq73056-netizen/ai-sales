package initialize

import (
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// bizModel 注册业务表
//
// 只建「不存在的表」，已存在的表不动——
// GORM 的 AutoMigrate 会尝试改列类型，历史数据长一点就会报
// 「Data truncated for column xxx」把启动流程打断。
func bizModel() error {
	db := global.GVA_DB

	models := []any{
		// 线索
		&leadModel.Lead{},

		// 抖音
		&leadModel.DouyinAccount{},
		&leadModel.DouyinDevice{},
		&leadModel.DouyinWhitelist{},
		&leadModel.DouyinTask{},
		&leadModel.DouyinDm{},

		// 微信客户与工单
		&leadModel.WxCustomer{},
		&leadModel.WxFollowup{},
		&leadModel.WxHandoff{},
	}

	for _, model := range models {
		if db.Migrator().HasTable(model) {
			continue
		}

		if err := db.AutoMigrate(model); err != nil {
			return err
		}
	}

	return nil
}
