package service

import (
	"errors"
	"strings"
	"time"

	"ticket-system/internal/model"
	"ticket-system/internal/repository"

	"gorm.io/gorm"
)

// ReviewService 负责评价业务。
// 单一职责：仅承载评价规则（关闭后、一单一评、仅提交人），不处理 HTTP。
type ReviewService struct {
	db         *gorm.DB
	reviewRepo *repository.ReviewRepository
	ticketRepo *repository.TicketRepository
}

func NewReviewService(db *gorm.DB, reviewRepo *repository.ReviewRepository, ticketRepo *repository.TicketRepository) *ReviewService {
	return &ReviewService{db: db, reviewRepo: reviewRepo, ticketRepo: ticketRepo}
}

// ReviewInput 评价入参。
type ReviewInput struct {
	Speed       int
	Quality     int
	Communicate int
	Comment     string
}

// Submit 提交评价。规则：工单已关闭、仅提交人可评、一单一评。
func (s *ReviewService) Submit(ticketID uint, actor Actor, in ReviewInput) (*model.Review, error) {
	t, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		return nil, errors.New("工单不存在")
	}
	if t.Status != model.StatusClosed {
		return nil, errors.New("仅已关闭的工单可评价")
	}
	if t.SubmitterID != actor.ID {
		return nil, errors.New("仅提交人可评价")
	}
	if in.Speed < 1 || in.Speed > 5 || in.Quality < 1 || in.Quality > 5 || in.Communicate < 1 || in.Communicate > 5 {
		return nil, errors.New("评分需在 1-5 之间")
	}
	exists, err := s.reviewRepo.ExistsByTicket(ticketID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("该工单已评价")
	}

	in.Comment = strings.TrimSpace(in.Comment)
	if len([]rune(in.Comment)) > 5000 {
		return nil, errors.New("评语长度不能超过 5000 个字符")
	}
	rv := &model.Review{
		TicketID: ticketID, SubmitterID: actor.ID,
		Speed: in.Speed, Quality: in.Quality, Communicate: in.Communicate,
		Comment: in.Comment, CreatedAt: time.Now(),
	}
	if err := s.reviewRepo.CreateForTicket(ticketID, rv); err != nil {
		// 并发提交时，唯一约束（uniqueIndex:ticket_id）兜底：
		// 落败的请求拿到重复键错误，转成清晰的“已评价”反馈，
		// 而不依赖 ExistsByTicket 与写入之间的先后顺序。
		// 优先用 gorm.ErrDuplicatedKey（需 TranslateError）；并匹配原始约束文本以防翻译未启用。
		if isDuplicatedKey(err) {
			return nil, errors.New("该工单已评价")
		}
		return nil, err
	}
	return rv, nil
}

// isDuplicatedKey 判断是否为唯一约束冲突。
// gorm.ErrDuplicatedKey 依赖 Dialector.Translate（由 TranslateError 开启）；
// 文本匹配作为兜底，覆盖 SQLite 未翻译时的 "UNIQUE constraint failed" 等。
func isDuplicatedKey(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicated key") ||
		strings.Contains(msg, "duplicate entry")
}
