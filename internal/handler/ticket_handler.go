package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ticket-system/internal/model"
	"ticket-system/internal/service"
)

// TicketHandler 处理工单相关 HTTP 接口（含附件/备注/评价/指派/超时/分类/处理人列表）。
// 单一职责：仅做 HTTP I/O 与参数解析，业务规则在 TicketService / AssignService。
type TicketHandler struct {
	ticket *service.TicketService
	assign *service.AssignService
	review *service.ReviewService
	cat    CategoryProvider // 提供类型列表
}

// CategoryProvider 抽象“取类型列表”的依赖，避免 handler 直接依赖 repository。
type CategoryProvider interface {
	AllCategories() ([]model.Category, error)
}

func NewTicketHandler(ticket *service.TicketService, assign *service.AssignService, review *service.ReviewService, cat CategoryProvider) *TicketHandler {
	return &TicketHandler{ticket: ticket, assign: assign, review: review, cat: cat}
}

// ---- 工单提交 ----

func (h *TicketHandler) Create(c *gin.Context) {
	actor := actorFromUser(currentUser(c))

	title := c.PostForm("title")
	desc := c.PostForm("description")
	categoryID := atoi(c.PostForm("category_id"), 0)
	urgency := c.PostForm("urgency")
	if urgency == "" {
		urgency = model.UrgencyNormal
	}
	if title == "" || categoryID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "标题和类型不能为空"})
		return
	}

	// 附件落盘
	form, _ := c.MultipartForm()
	var saved []service.SavedAttachment
	if form != nil && len(form.File["attachments"]) > 0 {
		for _, fh := range form.File["attachments"] {
			if fh.Size > int64(20)*1024*1024 {
				c.JSON(http.StatusBadRequest, gin.H{"message": "单个附件不能超过 20MB"})
				return
			}
			f, err := fh.Open()
			if err != nil {
				fail(c, err)
				return
			}
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				fail(c, err)
				return
			}
			rel, err := h.ticket.SaveUpload(fh.Filename, data)
			if err != nil {
				fail(c, err)
				return
			}
			saved = append(saved, service.SavedAttachment{Filename: fh.Filename, Path: rel, Size: fh.Size})
		}
	}

	t, err := h.ticket.Create(service.CreateInput{
		Title: title, Description: desc, CategoryID: uint(categoryID),
		Urgency: urgency, SubmitterID: actor.ID, Attachments: saved,
	})
	if err != nil {
		fail(c, err)
		return
	}
	created(c, ticketDetailDTO(t))
}

// ---- 列表 ----

func (h *TicketHandler) List(c *gin.Context) {
	f := parseListFilter(c)
	res, err := h.ticket.ListFor(actorFromUser(currentUser(c)), f)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, listDTO(res))
}

func (h *TicketHandler) MySubmitted(c *gin.Context) {
	f := parseListFilter(c)
	res, err := h.ticket.MySubmitted(actorFromUser(currentUser(c)), f)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, listDTO(res))
}

func (h *TicketHandler) Pending(c *gin.Context) {
	f := parseListFilter(c)
	res, err := h.ticket.Pending(actorFromUser(currentUser(c)), f)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, listDTO(res))
}

func (h *TicketHandler) Processed(c *gin.Context) {
	f := parseListFilter(c)
	res, err := h.ticket.Processed(actorFromUser(currentUser(c)), f)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, listDTO(res))
}

func (h *TicketHandler) Overdue(c *gin.Context) {
	f := parseListFilter(c)
	res, err := h.ticket.OverdueFor(actorFromUser(currentUser(c)), f)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, listDTO(res))
}

func (h *TicketHandler) Get(c *gin.Context) {
	id := pathID(c)
	t, err := h.ticket.GetForActor(id, actorFromUser(currentUser(c)))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, ticketDetailDTO(t))
}

// ---- 更新 ----

type updateReq struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

func (h *TicketHandler) Update(c *gin.Context) {
	var req updateReq
	if !bindJSON(c, &req) {
		return
	}
	t, err := h.ticket.Update(pathID(c), actorFromUser(currentUser(c)), service.UpdateInput{
		Title: req.Title, Description: req.Description,
	})
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, ticketDetailDTO(t))
}

type statusReq struct {
	Status string `json:"status" binding:"required"`
}

func (h *TicketHandler) UpdateStatus(c *gin.Context) {
	var req statusReq
	if !bindJSON(c, &req) {
		return
	}
	t, err := h.ticket.UpdateStatus(pathID(c), actorFromUser(currentUser(c)), strings.TrimSpace(strings.ToLower(req.Status)))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, ticketDetailDTO(t))
}

type assignReq struct {
	AssigneeID uint `json:"assignee_id" binding:"required"`
}

func (h *TicketHandler) Assign(c *gin.Context) {
	var req assignReq
	if !bindJSON(c, &req) {
		return
	}
	t, err := h.assign.Assign(pathID(c), actorFromUser(currentUser(c)), req.AssigneeID)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, ticketDetailDTO(t))
}

type commentReq struct {
	Content string `json:"content" binding:"required"`
}

func (h *TicketHandler) AddComment(c *gin.Context) {
	var req commentReq
	if !bindJSON(c, &req) {
		return
	}
	cm, err := h.ticket.AddComment(pathID(c), actorFromUser(currentUser(c)), req.Content)
	if err != nil {
		fail(c, err)
		return
	}
	created(c, commentDTO(cm))
}

func (h *TicketHandler) AddAttachment(c *gin.Context) {
	id := pathID(c)
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请上传文件"})
		return
	}
	if file.Size > int64(20)*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "单个附件不能超过 20MB"})
		return
	}
	f, err := file.Open()
	if err != nil {
		fail(c, err)
		return
	}
	data, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		fail(c, err)
		return
	}
	rel, err := h.ticket.SaveUpload(file.Filename, data)
	if err != nil {
		fail(c, err)
		return
	}
	if err := h.ticket.AddAttachmentFor(id, actorFromUser(currentUser(c)), service.SavedAttachment{
		Filename: file.Filename, Path: rel, Size: file.Size,
	}); err != nil {
		fail(c, err)
		return
	}
	created(c, gin.H{"filename": file.Filename, "path": rel, "size": file.Size})
}

func (h *TicketHandler) DownloadAttachment(c *gin.Context) {
	id := pathID(c)
	att, err := h.ticket.AttachmentFor(id, actorFromUser(currentUser(c)))
	if err != nil {
		fail(c, err)
		return
	}
	root, err := filepath.Abs(h.ticket.UploadDir())
	if err != nil {
		fail(c, err)
		return
	}
	abs, err := filepath.Abs(filepath.Join(root, att.FilePath))
	if err != nil {
		fail(c, err)
		return
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "附件路径非法"})
		return
	}
	if _, err := os.Stat(abs); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "附件文件不存在"})
		return
	}
	c.FileAttachment(abs, att.Filename)
}

// ---- 评价 ----

type reviewReq struct {
	Speed       int    `json:"speed" binding:"required"`
	Quality     int    `json:"quality" binding:"required"`
	Communicate int    `json:"communicate" binding:"required"`
	Comment     string `json:"comment"`
}

func (h *TicketHandler) SubmitReview(c *gin.Context) {
	var req reviewReq
	if !bindJSON(c, &req) {
		return
	}
	if req.Speed < 1 || req.Speed > 5 || req.Quality < 1 || req.Quality > 5 || req.Communicate < 1 || req.Communicate > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "评分需在 1-5 之间"})
		return
	}
	rv, err := h.review.Submit(pathID(c), actorFromUser(currentUser(c)), service.ReviewInput{
		Speed: req.Speed, Quality: req.Quality, Communicate: req.Communicate, Comment: req.Comment,
	})
	if err != nil {
		fail(c, err)
		return
	}
	created(c, reviewDTO(rv))
}

// ---- 参考数据 ----

func (h *TicketHandler) Categories(c *gin.Context) {
	cats, err := h.cat.AllCategories()
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, cats)
}

func (h *TicketHandler) Handlers(c *gin.Context) {
	group := c.Query("group")
	if group == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "缺少 group 参数"})
		return
	}
	users, err := h.assign.GroupHandlers(group)
	if err != nil {
		fail(c, err)
		return
	}
	dtos := make([]gin.H, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, gin.H{
			"id": u.ID, "name": u.Name, "group": u.Group, "is_leader": u.IsLeader,
		})
	}
	ok(c, dtos)
}

// ---- DTO 与辅助 ----

func parseListFilter(c *gin.Context) service.ListFilter {
	page, pageSize := pageQuery(c)
	return service.ListFilter{
		Page: page, PageSize: pageSize,
		Status: c.Query("status"), Category: c.Query("category"),
		Urgency: c.Query("urgency"), Search: c.Query("search"),
	}
}

func pathID(c *gin.Context) uint {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(id)
}

func listDTO(res *service.ListResult) gin.H {
	items := make([]gin.H, 0, len(res.Items))
	now := time.Now()
	for _, t := range res.Items {
		items = append(items, ticketListDTO(t, now))
	}
	return gin.H{
		"items": items, "total": res.Total, "page": res.Page, "page_size": res.PageSize,
	}
}

func ticketListDTO(t model.Ticket, now time.Time) gin.H {
	dto := gin.H{
		"id": t.ID, "ticket_no": t.TicketNo, "title": t.Title,
		"urgency": t.Urgency, "status": t.Status, "group": t.Group,
		"submitted_at": t.SubmittedAt, "overdue": t.IsOverdue(now),
		"submitter": userLite(t.Submitter), "assignee": userLite(t.Assignee),
		"category": catLite(t.Category),
	}
	return dto
}

func ticketDetailDTO(t *model.Ticket) gin.H {
	now := time.Now()
	comments := make([]gin.H, 0, len(t.Comments))
	for _, cm := range t.Comments {
		comments = append(comments, commentDTO(&cm))
	}
	attachments := make([]gin.H, 0, len(t.Attachments))
	for _, a := range t.Attachments {
		attachments = append(attachments, gin.H{
			"id": a.ID, "filename": a.Filename, "file_size": a.FileSize,
			"created_at": a.CreatedAt, "download_url": fmt.Sprintf("/api/v1/attachments/%d", a.ID),
		})
	}
	dto := gin.H{
		"id": t.ID, "ticket_no": t.TicketNo, "title": t.Title, "description": t.Description,
		"urgency": t.Urgency, "status": t.Status, "group": t.Group,
		"submitted_at": t.SubmittedAt, "completed_at": t.CompletedAt, "closed_at": t.ClosedAt,
		"overdue":   t.IsOverdue(now),
		"submitter": userLite(t.Submitter), "assignee": userLite(t.Assignee),
		"category": catLite(t.Category),
		"comments": comments, "attachments": attachments,
		"review": reviewDTOOrNil(t.Review),
	}
	return dto
}

func userLite(u *model.User) gin.H {
	if u == nil {
		return nil
	}
	return gin.H{"id": u.ID, "name": u.Name, "role": u.Role, "group": u.Group, "is_leader": u.IsLeader}
}

func catLite(c *model.Category) gin.H {
	if c == nil {
		return nil
	}
	return gin.H{"id": c.ID, "name": c.Name, "group": c.Group}
}

func commentDTO(cm *model.Comment) gin.H {
	if cm == nil {
		return nil
	}
	return gin.H{
		"id": cm.ID, "content": cm.Content, "type": cm.Type,
		"created_at": cm.CreatedAt, "user": userLite(cm.User),
	}
}

func reviewDTO(rv *model.Review) gin.H {
	if rv == nil {
		return nil
	}
	return gin.H{
		"id": rv.ID, "speed": rv.Speed, "quality": rv.Quality,
		"communicate": rv.Communicate, "comment": rv.Comment,
		"average": rv.Average(), "created_at": rv.CreatedAt,
		"submitter": userLite(rv.Submitter),
	}
}

func reviewDTOOrNil(rv *model.Review) gin.H {
	if rv == nil {
		return nil
	}
	return reviewDTO(rv)
}
