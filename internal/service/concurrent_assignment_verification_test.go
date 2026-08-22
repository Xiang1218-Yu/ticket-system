package service

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ticket-system/internal/model"
	"ticket-system/internal/repository"
)

var assignmentTestID uint64

func TestConcurrentAssignmentsKeepGroupsIsolated(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:assignment_race_%d?mode=memory&cache=shared", atomic.AddUint64(&assignmentTestID, 1))), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	if err := db.AutoMigrate(&model.User{}); err != nil { t.Fatal(err) }
	users := []model.User{{Username:"it-user",Name:"it user",Role:model.RoleHandler,Group:model.GroupIT},{Username:"admin-user",Name:"admin user",Role:model.RoleHandler,Group:model.GroupAdmin}}
	if err := db.Create(&users).Error; err != nil { t.Fatal(err) }
	repo := repository.NewUserRepository(db)
	start := make(chan struct{}); var wg sync.WaitGroup
	for _, group := range []string{model.GroupIT, model.GroupAdmin} { wg.Add(1); go func(group string) { defer wg.Done(); <-start; _, _ = repo.FindByGroup(group) }(group) }
	close(start); wg.Wait()
	if _, err := repo.FindByGroup(model.GroupAdmin); err != nil { t.Fatal(err) }
	user, err := repo.FindByID(users[0].ID); if err != nil { t.Fatal(err) }
	if user.Group != model.GroupIT { t.Fatalf("IT handler was read as group %q after another group query", user.Group) }
}

func TestGroupLookupReturnsOnlyRequestedHandlers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:assignment_group_%d?mode=memory&cache=shared", atomic.AddUint64(&assignmentTestID, 1))), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	if err := db.AutoMigrate(&model.User{}); err != nil { t.Fatal(err) }
	users := []model.User{{Username:"it-handler", Name:"it", Role:model.RoleHandler, Group:model.GroupIT}, {Username:"admin-handler", Name:"admin", Role:model.RoleHandler, Group:model.GroupAdmin}}
	if err := db.Create(&users).Error; err != nil { t.Fatal(err) }
	got, err := repository.NewUserRepository(db).FindByGroup(model.GroupIT)
	if err != nil { t.Fatal(err) }
	if len(got) != 1 || got[0].Group != model.GroupIT { t.Fatalf("group lookup returned %#v", got) }
}
