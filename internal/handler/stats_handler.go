package handler

import (
	"github.com/gin-gonic/gin"

	"ticket-system/internal/service"
)

// StatsHandler 处理统计看板相关 HTTP 接口。
// 单一职责：仅做 HTTP I/O，聚合逻辑在 StatsService。
type StatsHandler struct {
	stats *service.StatsService
}

func NewStatsHandler(stats *service.StatsService) *StatsHandler {
	return &StatsHandler{stats: stats}
}

// Dashboard 返回看板数据。
func (h *StatsHandler) Dashboard(c *gin.Context) {
	d, err := h.stats.Stats()
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{
		"by_category":  d.ByCategory,
		"by_status":    d.ByStatus,
		"today_new":    d.TodayNew,
		"week_new":     d.WeekNew,
		"pending":      d.Pending,
		"overdue":      d.Overdue,
		"avg_minutes":  d.AvgMinutes,
		"avg_text":     d.AvgDurationText(),
		// metric_scope 描述平均处理时长的生命周期口径：仅统计已完成状态（done/closed）的工单，
		// 与详情页“完成时间生效”规则一致。
		"metric_scope": "done,closed",
		"workload":     d.Workload,
	})
}
