package initialize

import (
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// bizModel 注册业务表，首次启动自动建表
//
// 之前这里是空的 AutoMigrate()，新机器必须手工导入 SQL 才有业务表；
// 现在把业务模型都注册进来，配合 gin-vue-admin 自带的 /init/initdb 就能一次建库。
func bizModel() error {
	db := global.GVA_DB

	return db.AutoMigrate(
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


	)
}
