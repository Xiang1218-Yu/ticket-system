package handler

import (
	"github.com/gin-gonic/gin"

	"ticket-system/internal/model"
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
		"by_category":   d.ByCategory,
		"by_status":     d.ByStatus,
		"today_new":     d.TodayNew,
		"week_new":      d.WeekNew,
		"pending":       d.Pending,
		"overdue":       d.Overdue,
		"avg_minutes":   d.AvgMinutes,
		"avg_text":      d.AvgDurationText(),
		"workload":      d.Workload,
		"open_statuses": []string{model.StatusPending, model.StatusProcessing},
	})
}
