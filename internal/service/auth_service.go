package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"ticket-system/internal/model"
	"ticket-system/internal/repository"
)

// AuthService 负责认证业务：注册、登录、JWT 签发与校验。
// 单一职责：仅处理认证与密码哈希，不触碰 HTTP、不直接持有 DB 连接。
type AuthService struct {
	userRepo    *repository.UserRepository
	jwtSecret   []byte
	expireHours int
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, expireHours int) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		jwtSecret:   []byte(jwtSecret),
		expireHours: expireHours,
	}
}

// parseLeeway 是解析侧对过期/签发时刻的容忍量。
// golang-jwt 默认 leeway=0，过期判定为 now < exp；当 now 恰好等于 exp（边界时刻）
// 时会被判为已过期，导致“刚到点的会话仍被判失效、稍早一瞬的会话仍有效”的不一致。
// 统一注入一个固定容差，使每个请求——无论新旧会话、无论配置是否刚变更——都按
// “exp + leeway”一致地判定，避免边界抖动。容忍量刻意远小于最小寿命。
const parseLeeway = 30 * time.Second

// Claims 是 JWT 中携带的用户信息。
type Claims struct {
	UserID uint   `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// RegisterInput 注册入参。
type RegisterInput struct {
	Username string
	Name     string
	Password string
	Role     string
	Group    string
	IsLeader bool
}

// Register 创建新用户。默认角色为普通员工；管理员不能自助注册。
func (s *AuthService) Register(in RegisterInput) (*model.User, error) {
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" || in.Password == "" || in.Name == "" {
		return nil, errors.New("用户名、姓名、密码不能为空")
	}
	if len(in.Password) < 6 {
		return nil, errors.New("密码长度至少 6 位")
	}
	if in.Role == "" {
		in.Role = model.RoleEmployee
	}
	if in.Role == model.RoleAdmin {
		return nil, errors.New("不允许自助注册管理员")
	}
	if in.Role == model.RoleHandler && in.Group == "" {
		return nil, errors.New("处理人必须指定所属组")
	}

	exists, err := s.userRepo.ExistsByUsername(in.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("用户名已存在")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &model.User{
		Username: in.Username,
		Password: string(hash),
		Name:     in.Name,
		Role:     in.Role,
		Group:    in.Group,
		IsLeader: in.IsLeader,
	}
	if err := s.userRepo.Create(u); err != nil {
		return nil, err
	}
	return u, nil
}

// Login 校验凭据并返回 JWT。
func (s *AuthService) Login(username, password string) (string, *model.User, error) {
	u, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return "", nil, errors.New("用户名或密码错误")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return "", nil, errors.New("用户名或密码错误")
	}
	token, err := s.issueToken(u)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}

func (s *AuthService) issueToken(u *model.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: u.ID,
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(s.tokenExpiry(now)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(s.jwtSecret)
}

// tokenExpiry 计算令牌的绝对过期时刻：签发时刻 + 配置寿命。
// 寿命直接取配置小时数，不得再减一——历史上这里的 (expireHours-1) 会让
// 1 小时寿命收敛为 0、令牌签发即过期，且令所有寿命都偷短 1 小时。
func (s *AuthService) tokenExpiry(now time.Time) time.Time {
	return now.Add(time.Duration(s.expireHours) * time.Hour)
}

// ParseToken 解析并校验 JWT，返回声明。
// 校验顺序：签名方法 + 密钥（防伪）→ 声明时效（含 parseLeeway 一致容差）。
// 不在此处放宽对“异常令牌”的处理：解析失败一律返回错误，由中间件拒绝。
func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	}, jwt.WithLeeway(parseLeeway))
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// FindUserByID 供中间件加载当前用户。
func (s *AuthService) FindUserByID(id uint) (*model.User, error) {
	return s.userRepo.FindByID(id)
}
