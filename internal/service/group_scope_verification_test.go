package service

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ticket-system/internal/model"
	"ticket-system/internal/repository"
)

var groupScopeTestID uint64

func TestLeaderCannotExpandListScopeWithRequestedGroup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:scope_%d?mode=memory&cache=shared", atomic.AddUint64(&groupScopeTestID, 1))), &gorm.Config{}); if err != nil { t.Fatal(err) }
	if err := db.AutoMigrate(&model.Ticket{}, &model.User{}, &model.Category{}, &model.Comment{}); err != nil { t.Fatal(err) }
	if err := db.Create(&model.Ticket{TicketNo:"T-other",Title:"other",Urgency:model.UrgencyUrgent,Status:model.StatusPending,SubmitterID:1,Group:model.GroupAdmin,SubmittedAt:time.Now().Add(-13*time.Hour)}).Error; err != nil { t.Fatal(err) }
	svc := NewTicketService(db, repository.NewTicketRepository(db), repository.NewCategoryRepository(db), repository.NewCommentRepository(db), repository.NewAttachmentRepository(db), t.TempDir())
	result, err := svc.ListFor(Actor{ID:1,Role:model.RoleHandler,Group:model.GroupIT,IsLeader:true}, ListFilter{Group:model.GroupAdmin,Page:1,PageSize:10})
	if err != nil { t.Fatal(err) }
	if len(result.Items) != 0 { t.Fatalf("leader observed %d tickets outside own group", len(result.Items)) }
}

func TestAdministratorCanFilterRequestedGroup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:scope_admin_%d?mode=memory&cache=shared", atomic.AddUint64(&groupScopeTestID, 1))), &gorm.Config{}); if err != nil { t.Fatal(err) }
	if err := db.AutoMigrate(&model.Ticket{}, &model.User{}, &model.Category{}, &model.Comment{}); err != nil { t.Fatal(err) }
	items := []model.Ticket{
		{TicketNo:"T-it", Title:"it", Urgency:model.UrgencyNormal, Status:model.StatusPending, SubmitterID:1, Group:model.GroupIT, SubmittedAt:time.Now()},
		{TicketNo:"T-admin", Title:"admin", Urgency:model.UrgencyNormal, Status:model.StatusPending, SubmitterID:2, Group:model.GroupAdmin, SubmittedAt:time.Now()},
	}
	if err := db.Create(&items).Error; err != nil { t.Fatal(err) }
	svc := NewTicketService(db, repository.NewTicketRepository(db), repository.NewCategoryRepository(db), repository.NewCommentRepository(db), repository.NewAttachmentRepository(db), t.TempDir())
	result, err := svc.ListFor(Actor{ID:99, Role:model.RoleAdmin}, ListFilter{Group:model.GroupAdmin, Page:1, PageSize:10})
	if err != nil { t.Fatal(err) }
	if len(result.Items) != 1 || result.Items[0].Group != model.GroupAdmin { t.Fatalf("administrator filter returned %#v", result.Items) }
}
