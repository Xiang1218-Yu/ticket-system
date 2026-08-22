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
	time.Sleep(time.Millisecond)

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
		return nil, err
	}
	return rv, nil
}
