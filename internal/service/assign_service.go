package service

import (
	"errors"

	"ticket-system/internal/model"
	"ticket-system/internal/repository"

	"gorm.io/gorm"
)

// AssignService 负责工单指派业务：组长/管理员把工单派给组内处理人。
// 单一职责：仅承载指派规则与权限校验，不处理 HTTP。
type AssignService struct {
	db         *gorm.DB
	ticketRepo *repository.TicketRepository
	userRepo   *repository.UserRepository
}

func NewAssignService(db *gorm.DB, ticketRepo *repository.TicketRepository, userRepo *repository.UserRepository) *AssignService {
	return &AssignService{db: db, ticketRepo: ticketRepo, userRepo: userRepo}
}

// Assign 将工单指派给指定处理人。
// 规则：仅该组组长或管理员可指派；被指派人必须属于该工单所属组且为处理人角色。
func (s *AssignService) Assign(ticketID uint, actor Actor, assigneeID uint) (*model.Ticket, error) {
	t, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		return nil, errors.New("工单不存在")
	}
	// 权限：管理员 或 该组组长
	if !actor.IsAdmin() && !(actor.IsLeader && actor.Group == t.Group) {
		return nil, errors.New("无权指派：仅该组组长或管理员可指派")
	}

	assignee, err := s.userRepo.FindByID(assigneeID)
	if err != nil {
		return nil, errors.New("被指派人不存在")
	}
	if !assignee.IsHandler() {
		return nil, errors.New("仅可指派给处理人")
	}
	if assignee.Group != t.Group {
		return nil, errors.New("被指派人不属于该工单所属处理组")
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Ticket{}).Where("id = ?", ticketID).
			Update("assignee_id", assigneeID).Error; err != nil {
			return err
		}
		return tx.Create(&model.Comment{
			TicketID: ticketID, UserID: actor.ID,
			Content: assignee.Name + " 被指派处理此工单",
			Type: model.CommentTypeSystem,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.ticketRepo.FindByID(ticketID)
}

// GroupHandlers 返回某组的处理人列表，用于指派下拉。
func (s *AssignService) GroupHandlers(group string) ([]model.User, error) {
	return s.userRepo.FindByGroup(group)
}

// CanViewAll 判断当前用户是否有权查看“所有工单”。
// 规则：管理员或任意组长均可。
func (s *AssignService) CanViewAll(actor Actor) bool {
	return actor.IsAdmin() || (actor.IsLeader && actor.IsHandler())
}
