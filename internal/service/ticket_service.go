package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"ticket-system/internal/model"
	"ticket-system/internal/repository"

	"gorm.io/gorm"
)

// Actor 表示一次业务操作的发起者（从 JWT 还原的当前用户最小信息）。
// 单一职责：仅承载操作上下文，业务逻辑在各 service 内。
type Actor struct {
	ID       uint
	Role     string
	Name     string
	Group    string
	IsLeader bool
}

func (a Actor) IsAdmin() bool    { return a.Role == model.RoleAdmin }
func (a Actor) IsHandler() bool  { return a.Role == model.RoleHandler }
func (a Actor) IsEmployee() bool { return a.Role == model.RoleEmployee }

// TicketService 负责工单业务：提交、查询、状态流转、超时判定。
// 单一职责：仅承载工单业务规则，不处理 HTTP、不直接管理文件存储以外的事。
type TicketService struct {
	db         *gorm.DB
	ticketRepo *repository.TicketRepository
	catRepo    *repository.CategoryRepository
	cmtRepo    *repository.CommentRepository
	attRepo    *repository.AttachmentRepository
	uploadDir  string
}

func NewTicketService(
	db *gorm.DB,
	ticketRepo *repository.TicketRepository,
	catRepo *repository.CategoryRepository,
	cmtRepo *repository.CommentRepository,
	attRepo *repository.AttachmentRepository,
	uploadDir string,
) *TicketService {
	return &TicketService{
		db: db, ticketRepo: ticketRepo, catRepo: catRepo,
		cmtRepo: cmtRepo, attRepo: attRepo, uploadDir: uploadDir,
	}
}

// CreateInput 提交工单入参。
type CreateInput struct {
	Title       string
	Description string
	CategoryID  uint
	Urgency     string
	SubmitterID uint
	// Attachments：已由 handler 落盘后的 (原始名, 相对路径, 大小) 列表
	Attachments []SavedAttachment
}

// SavedAttachment 表示已落盘的附件（handler 负责存储，service 仅记录元数据）。
type SavedAttachment struct {
	Filename string
	Path     string
	Size     int64
}

// Create 提交工单：查类型取组 → 生成编号 → 入库 → 写系统备注 → 存附件。
func (s *TicketService) Create(in CreateInput) (*model.Ticket, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	if in.Title == "" {
		return nil, errors.New("标题不能为空")
	}
	if len([]rune(in.Title)) > 200 {
		return nil, errors.New("标题长度不能超过 200 个字符")
	}
	if len([]rune(in.Description)) > 10000 {
		return nil, errors.New("描述长度不能超过 10000 个字符")
	}
	cat, err := s.catRepo.FindByID(in.CategoryID)
	if err != nil {
		return nil, errors.New("工单类型不存在")
	}
	if in.Urgency != model.UrgencyNormal && in.Urgency != model.UrgencyUrgent {
		in.Urgency = model.UrgencyNormal
	}

	var ticket *model.Ticket
	err = s.db.Transaction(func(tx *gorm.DB) error {
		no, err := s.nextTicketNo(tx)
		if err != nil {
			return err
		}
		now := time.Now()
		t := &model.Ticket{
			TicketNo:    no,
			Title:       in.Title,
			Description: in.Description,
			CategoryID:  in.CategoryID,
			Urgency:     in.Urgency,
			Status:      model.StatusPending,
			SubmitterID: in.SubmitterID,
			Group:       cat.Group,
			SubmittedAt: now,
		}
		if err := tx.Create(t).Error; err != nil {
			return err
		}
		// 系统备注：提交事件
		if err := tx.Create(&model.Comment{
			TicketID: t.ID, UserID: in.SubmitterID,
			Content: "工单已提交", Type: model.CommentTypeSystem,
			CreatedAt: now,
		}).Error; err != nil {
			return err
		}
		// 附件
		for _, a := range in.Attachments {
			if err := tx.Create(&model.Attachment{
				TicketID: t.ID, Filename: a.Filename, FilePath: a.Path, FileSize: a.Size,
			}).Error; err != nil {
				return err
			}
		}
		ticket = t
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.ticketRepo.FindByID(ticket.ID)
}

// nextTicketNo 生成当日序号：T + YYYYMMDD + 4 位序号。
func (s *TicketService) nextTicketNo(tx *gorm.DB) (string, error) {
	prefix := "T" + time.Now().Format("20060102")
	var count int64
	if err := tx.Model(&model.Ticket{}).
		Where("ticket_no LIKE ?", prefix+"%").Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%04d", prefix, count+1), nil
}

// Get 详情（含超时标记由调用方或 DTO 计算）。
func (s *TicketService) Get(id uint) (*model.Ticket, error) {
	t, err := s.ticketRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("工单不存在")
	}
	return t, nil
}

// GetForActor 获取工单详情并校验数据可见范围。
func (s *TicketService) GetForActor(id uint, actor Actor) (*model.Ticket, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if !canViewTicket(t, actor) {
		return nil, errors.New("无权查看此工单")
	}
	return t, nil
}

func canViewTicket(t *model.Ticket, actor Actor) bool {
	return actor.IsAdmin() || t.SubmitterID == actor.ID ||
		(t.AssigneeID != nil && *t.AssigneeID == actor.ID) ||
		(actor.IsHandler() && actor.Group != "" && actor.Group == t.Group)
}

func normalizedPage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func normalizedPageSize(size int) int {
	if size < 1 || size > 100 {
		return 10
	}
	return size
}

// ListFilter 列表筛选。
type ListFilter struct {
	Page     int
	PageSize int
	Status   string
	Category string
	Urgency  string
	Search   string
	Group    string
}

// ListResult 列表结果（含分页元信息）。
type ListResult struct {
	Items    []model.Ticket `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

func validateListFilter(f ListFilter) error {
	if f.Status != "" {
		valid := false
		for _, status := range model.AllStatuses() {
			if f.Status == status {
				valid = true
				break
			}
		}
		if !valid {
			return errors.New("状态筛选值无效")
		}
	}
	if f.Urgency != "" && f.Urgency != model.UrgencyNormal && f.Urgency != model.UrgencyUrgent {
		return errors.New("紧急程度筛选值无效")
	}
	return nil
}

func (s *TicketService) toRepoFilter(f ListFilter) repository.TicketFilter {
	return repository.TicketFilter{
		Page: f.Page, PageSize: f.PageSize,
		Status: f.Status, Category: f.Category, Urgency: f.Urgency, Group: f.Group, Search: f.Search,
	}
}

// List 所有工单（保留给内部调用；HTTP 层使用 ListFor 做权限隔离）。
func (s *TicketService) List(f ListFilter) (*ListResult, error) {
	if err := validateListFilter(f); err != nil {
		return nil, err
	}
	items, total, err := s.ticketRepo.List(s.toRepoFilter(f))
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: total, Page: normalizedPage(f.Page), PageSize: normalizedPageSize(f.PageSize)}, nil
}

// ListFor 返回当前用户有权看到的工单。管理员看全部，组长只看所属处理组。
// 授权收敛原则：身份归属始终覆盖外来条件——组长的可见组恒为其所属组，
// 客户端传入的 group 筛选一律忽略并强制收敛到 actor.Group，避免组长借 ?group=
// 读取其他组数据。管理员不受组约束，保留其合理的筛选能力。
func (s *TicketService) ListFor(actor Actor, f ListFilter) (*ListResult, error) {
	if actor.IsAdmin() {
		return s.List(f)
	}
	if !actor.IsHandler() || !actor.IsLeader || actor.Group == "" {
		return nil, errors.New("无权查看所有工单")
	}
	f.Group = actor.Group
	return s.List(f)
}

// MySubmitted 我提交的。
func (s *TicketService) MySubmitted(actor Actor, f ListFilter) (*ListResult, error) {
	if err := validateListFilter(f); err != nil {
		return nil, err
	}
	items, total, err := s.ticketRepo.FindBySubmitter(actor.ID, s.toRepoFilter(f))
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, nil
}

// Pending 待我处理。
func (s *TicketService) Pending(actor Actor, f ListFilter) (*ListResult, error) {
	if err := validateListFilter(f); err != nil {
		return nil, err
	}
	items, total, err := s.ticketRepo.FindPending(actor.ID, s.toRepoFilter(f))
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, nil
}

// Processed 我处理过的。
func (s *TicketService) Processed(actor Actor, f ListFilter) (*ListResult, error) {
	if err := validateListFilter(f); err != nil {
		return nil, err
	}
	items, total, err := s.ticketRepo.FindProcessed(actor.ID, s.toRepoFilter(f))
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, nil
}

// Overdue 超时工单列表（保留给内部调用）。
func (s *TicketService) Overdue(f ListFilter) (*ListResult, error) {
	if err := validateListFilter(f); err != nil {
		return nil, err
	}
	items, total, err := s.ticketRepo.ListOverdue(s.toRepoFilter(f), time.Now())
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: total, Page: normalizedPage(f.Page), PageSize: normalizedPageSize(f.PageSize)}, nil
}

// OverdueFor 返回当前用户有权看到的超时工单。
// 授权收敛原则同 ListFor：组长可见组恒为所属组，忽略客户端 group 筛选；
// 管理员看全部并保留合理筛选。
func (s *TicketService) OverdueFor(actor Actor, f ListFilter) (*ListResult, error) {
	if actor.IsAdmin() {
		return s.Overdue(f)
	}
	if !actor.IsHandler() || !actor.IsLeader || actor.Group == "" {
		return nil, errors.New("无权查看超时工单")
	}
	f.Group = actor.Group
	return s.Overdue(f)
}

// UpdateInput 更新工单字段（标题/描述）。
type UpdateInput struct {
	Title       *string
	Description *string
}

// Update 更新工单基本信息，仅提交人可改且未关闭前。
func (s *TicketService) Update(id uint, actor Actor, in UpdateInput) (*model.Ticket, error) {
	t, err := s.ticketRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("工单不存在")
	}
	if t.SubmitterID != actor.ID && !actor.IsAdmin() {
		return nil, errors.New("无权修改他人工单")
	}
	if t.Status == model.StatusClosed {
		return nil, errors.New("工单已关闭，不可修改")
	}
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return nil, errors.New("标题不能为空")
		}
		if len([]rune(title)) > 200 {
			return nil, errors.New("标题长度不能超过 200 个字符")
		}
		t.Title = title
	}
	if in.Description != nil {
		description := strings.TrimSpace(*in.Description)
		if len([]rune(description)) > 10000 {
			return nil, errors.New("描述长度不能超过 10000 个字符")
		}
		t.Description = description
	}
	if err := s.ticketRepo.Update(t); err != nil {
		return nil, err
	}
	return s.ticketRepo.FindByID(id)
}

// UpdateStatus 推进状态并记录系统备注。
func (s *TicketService) UpdateStatus(id uint, actor Actor, to string) (*model.Ticket, error) {
	t, err := s.ticketRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("工单不存在")
	}
	if to == t.Status {
		return nil, errors.New("状态未变更")
	}
	if !model.LegalTransition(t.Status, to) {
		return nil, fmt.Errorf("不允许从 %s 迁移到 %s", t.Status, to)
	}
	// 权限：接单需指派给该处理人；完成需指派处理人/组长/管理员；关闭需提交人或管理员。
	can := false
	switch to {
	case model.StatusProcessing:
		can = (t.AssigneeID != nil && *t.AssigneeID == actor.ID) || actor.IsAdmin()
	case model.StatusDone:
		can = (t.AssigneeID != nil && *t.AssigneeID == actor.ID) ||
			(t.Group == actor.Group && actor.IsLeader) || actor.IsAdmin()
	case model.StatusClosed:
		can = t.SubmitterID == actor.ID || actor.IsAdmin()
	}
	if !can {
		return nil, errors.New("无权执行此状态变更")
	}

	now := time.Now()
	err = s.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{"status": to}
		if to == model.StatusDone {
			updates["completed_at"] = now
		}
		if to == model.StatusClosed {
			updates["closed_at"] = now
		}
		result := tx.Model(&model.Ticket{}).Where("id = ? AND status = ?", id, t.Status).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("工单状态已被其他操作更新，请刷新后重试")
		}
		return tx.Create(&model.Comment{
			TicketID: id, UserID: actor.ID,
			Content: fmt.Sprintf("状态由 %s 变更为 %s", t.Status, to),
			Type:    model.CommentTypeSystem, CreatedAt: now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.ticketRepo.FindByID(id)
}

// AddComment 添加处理备注。
func (s *TicketService) AddComment(ticketID uint, actor Actor, content string) (*model.Comment, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("备注内容不能为空")
	}
	if len([]rune(content)) > 5000 {
		return nil, errors.New("备注长度不能超过 5000 个字符")
	}
	t, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		return nil, errors.New("工单不存在")
	}
	// 备注权限：处理人、组长、提交人、管理员均可
	canComment := actor.IsAdmin() ||
		t.SubmitterID == actor.ID ||
		(t.AssigneeID != nil && *t.AssigneeID == actor.ID) ||
		(t.Group == actor.Group && actor.IsHandler())
	if !canComment {
		return nil, errors.New("无权在此工单添加备注")
	}
	c := &model.Comment{
		TicketID: ticketID, UserID: actor.ID,
		Content: content, Type: model.CommentTypeRemark, CreatedAt: time.Now(),
	}
	if err := s.cmtRepo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

// SaveUpload 把 multipart 文件落盘，返回相对路径。仅负责存储 I/O。
func (s *TicketService) SaveUpload(filename string, data []byte) (string, error) {
	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		return "", err
	}
	safe := sanitizeFilename(filename)
	rel := filepath.Join(time.Now().Format("20060102"), fmt.Sprintf("%d_%s", time.Now().UnixNano(), safe))
	abs := filepath.Join(s.uploadDir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return "", err
	}
	return rel, nil
}

// AddAttachment 为已有工单补传附件（兼容内部调用）。
func (s *TicketService) AddAttachment(ticketID uint, att SavedAttachment) error {
	return s.attRepo.Create(&model.Attachment{
		TicketID: ticketID, Filename: att.Filename, FilePath: att.Path, FileSize: att.Size,
	})
}

// AddAttachmentFor 为已有工单补传附件并校验访问权限。
func (s *TicketService) AddAttachmentFor(ticketID uint, actor Actor, att SavedAttachment) error {
	t, err := s.Get(ticketID)
	if err != nil {
		return err
	}
	if !canViewTicket(t, actor) {
		return errors.New("无权在此工单添加附件")
	}
	if t.Status == model.StatusClosed {
		return errors.New("工单已关闭，不可添加附件")
	}
	if att.Filename == "" || att.Path == "" {
		return errors.New("附件信息不能为空")
	}
	return s.AddAttachment(ticketID, att)
}

// AttachmentByID 取附件（兼容内部调用）。
func (s *TicketService) AttachmentByID(id uint) (*model.Attachment, error) {
	return s.attRepo.FindByID(id)
}

// AttachmentFor 取附件并校验当前用户是否能看到所属工单。
func (s *TicketService) AttachmentFor(id uint, actor Actor) (*model.Attachment, error) {
	att, err := s.AttachmentByID(id)
	if err != nil {
		return nil, errors.New("附件不存在")
	}
	if _, err := s.GetForActor(att.TicketID, actor); err != nil {
		return nil, err
	}
	return att, nil
}

// sanitizeFilename 仅保留安全字符，防止路径穿越。
func sanitizeFilename(name string) string {
	var b []rune
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' {
			b = append(b, r)
		} else if r == ' ' {
			b = append(b, '_')
		}
	}
	out := string(b)
	if out == "" {
		out = "file"
	}
	return out
}

// UploadDir 返回上传目录绝对路径，供下载 handler 拼接。
func (s *TicketService) UploadDir() string { return s.uploadDir }
