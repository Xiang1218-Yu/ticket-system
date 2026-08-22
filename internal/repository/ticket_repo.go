package repository

import (
	"time"

	"ticket-system/internal/model"

	"gorm.io/gorm"
)

// TicketFilter 封装工单列表的筛选/分页参数。
// 单一职责：仅承载查询条件，由 TicketRepository 解释。
type TicketFilter struct {
	Page     int
	PageSize int
	Status   string
	Category string // 类型名称
	Urgency  string
	Group    string // 处理组范围（管理员为空表示全部）
	Search   string // 编号/标题/提交人
}

// TicketRepository 封装工单表的数据访问。
// 单一职责：仅读写 tickets（及关联），不含状态流转业务规则。
type TicketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

func (r *TicketRepository) Create(t *model.Ticket) error {
	return r.db.Create(t).Error
}

func (r *TicketRepository) FindByID(id uint) (*model.Ticket, error) {
	var t model.Ticket
	err := r.db.Preload("Submitter").Preload("Assignee").Preload("Category").
		Preload("Comments.User").Preload("Attachments").Preload("Review").
		First(&t, id).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// applyFilter 把 Filter 应用到查询，返回总数与结果列表。
func (r *TicketRepository) applyFilter(f TicketFilter) *gorm.DB {
	q := r.db.Model(&model.Ticket{}).
		Preload("Submitter").Preload("Assignee").Preload("Category")
	if f.Status != "" {
		q = q.Where("tickets.status = ?", f.Status)
	}
	if f.Urgency != "" {
		q = q.Where("tickets.urgency = ?", f.Urgency)
	}
	if f.Category != "" {
		// Category 是名称，需要 join categories 表
		q = q.Joins("JOIN categories ON categories.id = tickets.category_id").
			Where("categories.name = ?", f.Category)
	}
	if f.Group != "" {
		q = q.Where("tickets.`group` = ?", f.Group)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Joins("LEFT JOIN users AS sub ON sub.id = tickets.submitter_id").
			Where("tickets.ticket_no LIKE ? OR tickets.title LIKE ? OR sub.name LIKE ?", like, like, like)
	}
	return q
}

// List 按筛选分页返回工单（不含评论/附件，列表轻量）。
func (r *TicketRepository) List(f TicketFilter) ([]model.Ticket, int64, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 10
	}
	q := r.applyFilter(f)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tickets []model.Ticket
	err := q.Order("tickets.created_at DESC").
		Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).
		Find(&tickets).Error
	return tickets, total, err
}

// FindBySubmitter 我提交的工单。
func (r *TicketRepository) FindBySubmitter(submitterID uint, f TicketFilter) ([]model.Ticket, int64, error) {
	q := r.applyFilter(f).Where("submitter_id = ?", submitterID)
	return r.listWithCount(q, f)
}

// FindPending 待某处理人处理的工单（指派给他且未完成）。
func (r *TicketRepository) FindPending(assigneeID uint, f TicketFilter) ([]model.Ticket, int64, error) {
	q := r.applyFilter(f).
		Where("assignee_id = ? AND status IN ?", assigneeID, []string{model.StatusPending, model.StatusProcessing})
	return r.listWithCount(q, f)
}

// FindProcessed 某处理人处理过的工单历史（指派过他且已完成或关闭）。
func (r *TicketRepository) FindProcessed(assigneeID uint, f TicketFilter) ([]model.Ticket, int64, error) {
	q := r.applyFilter(f).
		Where("assignee_id = ? AND status IN ?", assigneeID, []string{model.StatusDone, model.StatusClosed})
	return r.listWithCount(q, f)
}

func (r *TicketRepository) listWithCount(q *gorm.DB, f TicketFilter) ([]model.Ticket, int64, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 10
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tickets []model.Ticket
	err := q.Order("tickets.created_at DESC").
		Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).
		Find(&tickets).Error
	return tickets, total, err
}

// ListOverdue 返回符合超时规则的未完成工单，并在数据库层完成筛选和分页。
// 这样不会因为默认分页大小而漏掉第 101 条及之后的超时工单。
func (r *TicketRepository) ListOverdue(f TicketFilter, now time.Time) ([]model.Ticket, int64, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 10
	}
	q := r.applyFilter(f).Where(`
		tickets.status IN ? AND
		((tickets.urgency = ? AND tickets.submitted_at < ?) OR
		 (tickets.urgency = ? AND tickets.submitted_at < ?))`,
		[]string{model.StatusPending, model.StatusProcessing},
		model.UrgencyUrgent, now.Add(-12*time.Hour),
		model.UrgencyNormal, now.Add(-48*time.Hour),
	)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tickets []model.Ticket
	err := q.Order("tickets.submitted_at ASC").
		Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&tickets).Error
	return tickets, total, err
}

// CountOverdue 统计当前超时的未完成工单数量。
func (r *TicketRepository) CountOverdue(now time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.Ticket{}).Where(`
		status IN ? AND
		((urgency = ? AND submitted_at < ?) OR
		 (urgency = ? AND submitted_at < ?))`,
		[]string{model.StatusPending, model.StatusProcessing},
		model.UrgencyUrgent, now.Add(-12*time.Hour),
		model.UrgencyNormal, now.Add(-48*time.Hour),
	).Count(&count).Error
	return count, err
}

// Update 保存工单字段变更。
func (r *TicketRepository) Update(t *model.Ticket) error {
	return r.db.Save(t).Error
}

// UpdateAssignee 仅更新指派处理人。
func (r *TicketRepository) UpdateAssignee(id, assigneeID uint) error {
	return r.db.Model(&model.Ticket{}).Where("id = ?", id).
		Update("assignee_id", assigneeID).Error
}

func (r *TicketRepository) TransitionStatus(tx *gorm.DB, id uint, updates map[string]interface{}) error {
	return tx.Model(&model.Ticket{}).Where("id = ?", id).Updates(updates).Error
}

// CountByStatus 按状态聚合工单数。
func (r *TicketRepository) CountByStatus() (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	err := r.db.Model(&model.Ticket{}).Select("status, count(*) as count").Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[string]int64, 4)
	for _, rw := range rows {
		m[rw.Status] = rw.Count
	}
	return m, nil
}

// CountByCategory 按类型名称聚合工单数。
func (r *TicketRepository) CountByCategory() ([]CategoryCount, error) {
	var rows []CategoryCount
	err := r.db.Table("tickets").
		Select("categories.name as category, count(*) as count").
		Joins("JOIN categories ON categories.id = tickets.category_id").
		Group("categories.name").
		Scan(&rows).Error
	return rows, err
}

// CountSince 统计某时间点之后新增的工单数。
func (r *TicketRepository) CountSince(since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.Ticket{}).Where("submitted_at >= ?", since).Count(&count).Error
	return count, err
}

// AvgProcessingMinutes 计算已完成工单的平均处理时长（分钟）。
func (r *TicketRepository) AvgProcessingMinutes() (float64, error) {
	var avg float64
	err := r.db.Model(&model.Ticket{}).
		Where("status IN ? AND completed_at IS NOT NULL", []string{model.StatusDone, model.StatusClosed}).
		Select("COALESCE(AVG(strftime('%s', completed_at) - strftime('%s', submitted_at)) / 60.0, 0)").
		Scan(&avg).Error
	return avg, err
}

// AssigneeWorkload 每个处理人已处理/进行中的工单数。
func (r *TicketRepository) AssigneeWorkload() ([]Workload, error) {
	var rows []Workload
	err := r.db.Table("tickets").
		Select("assignee_id, count(*) as count").
		Where("assignee_id IS NOT NULL").
		Group("assignee_id").
		Scan(&rows).Error
	return rows, err
}

// CategoryCount 类型计数结果。
type CategoryCount struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// Workload 处理人工作量结果。
type Workload struct {
	AssigneeID uint  `json:"assignee_id"`
	Count      int64 `json:"count"`
}
