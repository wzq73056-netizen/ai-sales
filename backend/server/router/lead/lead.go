package lead

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type LeadRouter struct {}

// InitLeadRouter 初始化 获客线索 路由信息
func (s *LeadRouter) InitLeadRouter(Router *gin.RouterGroup,PublicRouter *gin.RouterGroup) {
	leadRouter := Router.Group("lead").Use(middleware.OperationRecord())
	leadRouterWithoutRecord := Router.Group("lead")
	leadRouterWithoutAuth := PublicRouter.Group("lead")

	// 微信语音/图片的本地文件（详情里直接播放）
	PublicRouter.Static("/media", `E:\ai\data\media`)
	{
		leadRouter.POST("createLead", leadApi.CreateLead)   // 新建获客线索
		leadRouter.DELETE("deleteLead", leadApi.DeleteLead) // 删除获客线索
		leadRouter.DELETE("deleteLeadByIds", leadApi.DeleteLeadByIds) // 批量删除获客线索
		leadRouter.PUT("updateLead", leadApi.UpdateLead)    // 更新获客线索
	}
	{
		leadRouterWithoutRecord.GET("findLead", leadApi.FindLead)        // 根据ID获取获客线索
		leadRouterWithoutRecord.GET("getLeadList", leadApi.GetLeadList)  // 获取获客线索列表
	leadRouterWithoutRecord.POST("createDouyinTask", leadApi.CreateDouyinTask)   // 新建任务
	leadRouterWithoutRecord.GET("getDouyinTaskList", leadApi.GetDouyinTaskList)   // 任务列表
	leadRouterWithoutRecord.POST("saveDouyinAccounts", leadApi.SaveDouyinAccounts)         // 批量保存账号线索
	leadRouterWithoutRecord.POST("classifyComments", leadApi.ClassifyComments)          // 评论意向判定
	leadRouterWithoutRecord.POST("syncAccountsToLeads", leadApi.SyncAccountsToLeads)    // 账号线索同步
	leadRouterWithoutRecord.POST("checkDouyinDm", leadApi.CheckDouyinDm)               // 私信查重
	leadRouterWithoutRecord.POST("saveDouyinDm", leadApi.SaveDouyinDm)                 // 私信记录
	leadRouterWithoutRecord.GET("getDouyinDmList", leadApi.GetDouyinDmList)            // 私信列表
	leadRouterWithoutRecord.POST("updateDmReply", leadApi.UpdateDmReply)                 // 记录私信回复
	leadRouterWithoutRecord.POST("genDmReply", leadApi.GenDmReply)                     // AI 生成私信回复
	leadRouterWithoutRecord.GET("getDmControl", leadApi.GetDmControl)                 // 私信自动化状态
	leadRouterWithoutRecord.POST("checkWxIntent", leadApi.CheckWxIntent)             // 转人工判定
	leadRouterWithoutRecord.POST("createWxHandoff", leadApi.CreateWxHandoff)          // 建转人工工单
	leadRouterWithoutRecord.GET("getWxHandoffList", leadApi.GetWxHandoffList)         // 待接管列表
	leadRouterWithoutRecord.GET("getWxHandoffCount", leadApi.GetWxHandoffCount)        // 待接管角标
	leadRouterWithoutRecord.POST("claimWxHandoff", leadApi.ClaimWxHandoff)            // 认领
	leadRouterWithoutRecord.POST("resolveWxHandoff", leadApi.ResolveWxHandoff)        // 处理
	leadRouterWithoutRecord.POST("updateWxHandoffResult", leadApi.UpdateWxHandoffResult) // 转微信跟进
	leadRouterWithoutRecord.GET("getAiPausedChats", leadApi.GetAiPausedChats)                         // AI 暂停会话
	leadRouterWithoutRecord.POST("resumeAi", leadApi.ResumeAi)                                      // 手动恢复AI
	leadRouterWithoutRecord.GET("getFunnelStats", leadApi.GetFunnelStats)                             // 转化漏斗
	leadRouterWithoutRecord.POST("extractLeadContacts", leadApi.ExtractLeadContacts)                       // 抽取联系方式
	leadRouterWithoutRecord.GET("getSystemAlerts", leadApi.GetSystemAlerts)                                  // 系统告警
	leadRouterWithoutRecord.GET("getDaemons", leadApi.GetDaemons)                                       // 监听进程状态
	leadRouterWithoutRecord.POST("startDaemon", leadApi.StartDaemon)                                    // 启动监听进程
	leadRouterWithoutRecord.POST("stopDaemon", leadApi.StopDaemon)                                      // 停止监听进程
	leadRouterWithoutRecord.GET("getWorkbench", leadApi.GetWorkbench)                                  // 今日工作台
	leadRouterWithoutRecord.GET("getFollowupReminders", leadApi.GetFollowupReminders)                  // 跟进提醒
	leadRouterWithoutRecord.GET("getLLMConfig", leadApi.GetLLMConfig)                                 // 大模型配置读取
	leadRouterWithoutRecord.POST("saveLLMConfig", leadApi.SaveLLMConfig)                              // 大模型配置保存
	leadRouterWithoutRecord.POST("testLLMConfig", leadApi.TestLLMConfig)                              // 大模型连通性测试
	leadRouterWithoutRecord.GET("tableViewList", leadApi.TableViewList)                                 // 表格视图列表
	leadRouterWithoutRecord.POST("tableViewSave", leadApi.TableViewSave)                                // 保存表格视图
	leadRouterWithoutRecord.DELETE("tableViewDelete", leadApi.TableViewDelete)                           // 删除表格视图
	leadRouterWithoutRecord.GET("getInbox", leadApi.GetInbox)                                        // 统一收件箱
	leadRouterWithoutRecord.POST("recordDmMessages", leadApi.RecordDmMessages)                          // 私信消息入库
	leadRouterWithoutRecord.GET("getDmConversations", leadApi.GetDmConversations)                      // 私信会话
	leadRouterWithoutRecord.GET("getScrapeKeywords", leadApi.GetScrapeKeywords)                        // 关键词库
	leadRouterWithoutRecord.POST("saveScrapeKeyword", leadApi.SaveScrapeKeyword)                       // 新增关键词
	leadRouterWithoutRecord.POST("deleteScrapeKeyword", leadApi.DeleteScrapeKeyword)                   // 删除关键词
	leadRouterWithoutRecord.GET("getScrapeBlacklist", leadApi.GetScrapeBlacklist)                      // 黑名单
	leadRouterWithoutRecord.POST("saveScrapeBlacklist", leadApi.SaveScrapeBlacklist)                   // 新增黑名单
	leadRouterWithoutRecord.POST("deleteScrapeBlacklist", leadApi.DeleteScrapeBlacklist)               // 删除黑名单
	leadRouterWithoutRecord.GET("exportLeads", leadApi.ExportLeads)                                    // 导出线索
	leadRouterWithoutRecord.POST("convertLeads", leadApi.ConvertLeads)                                  // 批量转客户
	leadRouterWithoutRecord.GET("getLeadAnalytics", leadApi.GetLeadAnalytics)                          // 报表分析
	leadRouterWithoutRecord.POST("importLeads", leadApi.ImportLeads)                                   // 导入线索
	leadRouterWithoutRecord.POST("convertLead", leadApi.ConvertLead)                                    // 线索转客户
	leadRouterWithoutRecord.POST("addWxFriend", leadApi.AddWxFriend)                                    // 微信加好友
	leadRouterWithoutRecord.GET("getWxAddConfig", leadApi.GetWxAddConfig)                                // 加好友配置
	leadRouterWithoutRecord.POST("saveWxAddConfig", leadApi.SaveWxAddConfig)                            // 保存加好友配置
	leadRouterWithoutRecord.POST("enqueueWxAdd", leadApi.EnqueueWxAdd)                                  // 加入加好友队列
	leadRouterWithoutRecord.GET("getWxAddQueue", leadApi.GetWxAddQueue)                                 // 加好友队列
	leadRouterWithoutRecord.POST("runWxAddOnce", leadApi.RunWxAddOnce)                                  // 手动执行一次
	leadRouterWithoutRecord.GET("claimWxAddJob", leadApi.ClaimWxAddJob)                                  // 领取加好友任务
	leadRouterWithoutRecord.POST("finishWxAddJob", leadApi.FinishWxAddJob)                              // 回报加好友结果
	leadRouterWithoutRecord.GET("getScriptStats", leadApi.GetScriptStats)                              // 话术效果
	leadRouterWithoutRecord.POST("gradeAllLeads", leadApi.GradeAllLeads)                                // 线索自动分级
	leadRouterWithoutRecord.POST("applyLeadContact", leadApi.ApplyLeadContact)                        // 回填联系方式
	leadRouterWithoutRecord.POST("genLeadDmScript", leadApi.GenLeadDmScript)                          // 生成私信话术
	leadRouterWithoutRecord.POST("genLeadDmScripts", leadApi.GenLeadDmScripts)                        // 批量生成话术
	leadRouterWithoutRecord.POST("startXhsCollect", leadApi.StartXhsCollect)                            // 小红书采集
	leadRouterWithoutRecord.GET("getXhsCollectStatus", leadApi.GetXhsCollectStatus)                    // 采集状态
	leadRouterWithoutRecord.GET("getXhsLeadStats", leadApi.GetXhsLeadStats)                            // 小红书线索统计
	leadRouterWithoutRecord.GET("getXhsRuns", leadApi.GetXhsRuns)                                      // 小红书采集历史
	leadRouterWithoutRecord.POST("sendXhsDm", leadApi.SendXhsDm)                                        // 小红书主动私信
	leadRouterWithoutRecord.POST("sendXhsProactiveDm", leadApi.SendXhsProactiveDm)                      // 小红书线索主动私信
	leadRouterWithoutRecord.POST("sendDyDm", leadApi.SendDyDm)                                          // 抖音主动私信
	leadRouterWithoutRecord.GET("getXhsSchedule", leadApi.GetXhsSchedule)                              // 小红书定时配置
	leadRouterWithoutRecord.POST("saveXhsSchedule", leadApi.SaveXhsSchedule)                            // 保存定时配置
	leadRouterWithoutRecord.GET("getDouyinDevices", leadApi.GetDouyinDevices)                           // 抖音账号列表
	leadRouterWithoutRecord.POST("saveDouyinDevice", leadApi.SaveDouyinDevice)                        // 保存抖音账号
	leadRouterWithoutRecord.POST("deleteDouyinDevice", leadApi.DeleteDouyinDevice)                    // 删除抖音账号
	leadRouterWithoutRecord.GET("getDouyinWhitelist", leadApi.GetDouyinWhitelist)                     // 私信白名单
	leadRouterWithoutRecord.POST("saveDouyinWhitelist", leadApi.SaveDouyinWhitelist)                  // 保存白名单
	leadRouterWithoutRecord.POST("deleteDouyinWhitelist", leadApi.DeleteDouyinWhitelist)              // 删除白名单
	leadRouterWithoutRecord.POST("touchDouyinDevice", leadApi.TouchDouyinDevice)                      // 账号心跳
	leadRouterWithoutRecord.POST("syncWxCustomer", leadApi.SyncWxCustomer)             // 同步客户价值
	leadRouterWithoutRecord.GET("getWxCustomerList", leadApi.GetWxCustomerList)       // 客户跟进列表
	leadRouterWithoutRecord.POST("updateWxCustomer", leadApi.UpdateWxCustomer)                            // 更新客户
	leadRouterWithoutRecord.GET("getWxCustomerGrouped", leadApi.GetWxCustomerGrouped)      // 客户列表(合并)
	leadRouterWithoutRecord.GET("getWxCustomerDetail", leadApi.GetWxCustomerDetail)        // 客户详情
	leadRouterWithoutRecord.POST("addWxFollowup", leadApi.AddWxFollowup)                   // 新增跟进
	leadRouterWithoutRecord.POST("markWxCustomerFollow", leadApi.MarkWxCustomerFollow) // 标记跟进
	leadRouterWithoutRecord.POST("setDmControl", leadApi.SetDmControl)                // 私信自动化开关
	leadRouterWithoutRecord.GET("getPendingDmList", leadApi.GetPendingDmList)          // 待检查回复
	leadRouterWithoutRecord.GET("getDouyinAccountList", leadApi.GetDouyinAccountList)     // 账号线索列表
	leadRouterWithoutRecord.PUT("updateDouyinAccountStatus", leadApi.UpdateDouyinAccountStatus) // 更新状态
	leadRouterWithoutRecord.DELETE("deleteDouyinAccount", leadApi.DeleteDouyinAccount)   // 删除账号线索
	leadRouterWithoutRecord.POST("startDouyinTask", leadApi.StartDouyinTask)      // 启动任务
	leadRouterWithoutRecord.POST("stopDouyinTask", leadApi.StopDouyinTask)        // 停止任务
	leadRouterWithoutRecord.DELETE("deleteDouyinTask", leadApi.DeleteDouyinTask)  // 删除任务
	leadRouterWithoutRecord.GET("runningLog", leadApi.GetRunningLog)              // 实时日志
	leadRouterWithoutRecord.POST("genKeywords", leadApi.GenKeywords)              // AI 生成关键词
	}
	{
	    leadRouterWithoutAuth.GET("getLeadPublic", leadApi.GetLeadPublic)  // 获客线索开放接口
	}
}
