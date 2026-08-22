package service

import (
	"fmt"
	"time"

	"ticket-system/internal/model"
	"ticket-system/internal/repository"
)

// StatsService 负责统计看板数据。
// 单一职责：仅聚合统计，不修改任何数据。
type StatsService struct {
	ticketRepo *repository.TicketRepository
	userRepo   *repository.UserRepository
}

func NewStatsService(ticketRepo *repository.TicketRepository, userRepo *repository.UserRepository) *StatsService {
	return &StatsService{ticketRepo: ticketRepo, userRepo: userRepo}
}

// Dashboard 看板数据。
type Dashboard struct {
	ByCategory []repository.CategoryCount `json:"by_category"`
	ByStatus   map[string]int64           `json:"by_status"`
	TodayNew   int64                      `json:"today_new"`
	WeekNew    int64                      `json:"week_new"`
	Pending    int64                      `json:"pending"`
	Overdue    int64                      `json:"overdue"`
	AvgMinutes float64                    `json:"avg_minutes"`
	Workload   []HandlerLoad              `json:"workload"`
}

// HandlerLoad 处理人工作量。
type HandlerLoad struct {
	UserID uint   `json:"user_id"`
	Name   string `json:"name"`
	Group  string `json:"group"`
	Count  int64  `json:"count"`
}

// Stats 计算看板数据。
func (s *StatsService) Stats() (*Dashboard, error) {
	d := &Dashboard{ByStatus: map[string]int64{}}
	cats, err := s.ticketRepo.CountByCategory()
	if err != nil {
		return nil, err
	}
	d.ByCategory = cats
	statusCounts, err := s.ticketRepo.CountByStatus()
	if err != nil {
		return nil, err
	}
	for _, st := range model.AllStatuses() {
		d.ByStatus[st] = statusCounts[st]
	}

	now := time.Now()
	startToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startWeek := startToday.AddDate(0, 0, -int(now.Weekday())) // 本周日开始
	if d.TodayNew, err = s.ticketRepo.CountSince(startToday); err != nil {
		return nil, err
	}
	if d.WeekNew, err = s.ticketRepo.CountSince(startWeek); err != nil {
		return nil, err
	}
	for status, count := range statusCounts {
		if model.IsOpenStatus(status) {
			d.Pending += count
		}
	}
	if d.Overdue, err = s.ticketRepo.CountOverdue(now); err != nil {
		return nil, err
	}
	if d.AvgMinutes, err = s.ticketRepo.AvgProcessingMinutes(); err != nil {
		return nil, err
	}
	loads, err := s.ticketRepo.AssigneeWorkload()
	if err != nil {
		return nil, err
	}
	for _, l := range loads {
		u, findErr := s.userRepo.FindByID(l.AssigneeID)
		if findErr != nil {
			return nil, findErr
		}
		d.Workload = append(d.Workload, HandlerLoad{
			UserID: u.ID, Name: u.Name, Group: u.Group, Count: l.Count,
		})
	}
	return d, nil
}

// AvgDurationText 返回平均处理时长的人类可读描述。
func (d Dashboard) AvgDurationText() string {
	if d.AvgMinutes <= 0 {
		return "暂无数据"
	}
	h := int(d.AvgMinutes) / 60
	m := int(d.AvgMinutes) % 60
	return fmt.Sprintf("%d 小时 %d 分钟", h, m)
}
