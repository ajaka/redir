package handlers_test

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"log/slog"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"github.com/ajaka/redir/internal/configs"
// 	"github.com/ajaka/redir/internal/domain"
// 	"github.com/ajaka/redir/internal/handlers"
// 	"github.com/ajaka/redir/internal/middlewares"
// 	"github.com/ajaka/redir/internal/models"
// 	"github.com/gin-gonic/gin"
// 	"golang.org/x/crypto/bcrypt"
// )

// func init() {
// 	gin.SetMode(gin.TestMode)
// }

// // mockAuthStore implements store.AuthStore for testing
// type mockAuthStore struct {
// 	createErr error
// 	created   *domain.CreateUserDetails
// 	getUser   *models.User
// 	getErr    error
// 	setExp    time.Time
// 	setErr    error
// 	revokeErr error
// }

// func (m *mockAuthStore) CreateUser(ctx context.Context, logger *slog.Logger, user *domain.CreateUserDetails, cfg *configs.EnvData) error {
// 	m.created = user
// 	return m.createErr
// }
// func (m *mockAuthStore) GetUserByEmail(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, email string) (*models.User, error) {
// 	return m.getUser, m.getErr
// }
// func (m *mockAuthStore) SetUserOnline(ctx context.Context, logger *slog.Logger, sessionId string, u *domain.LightUser) (time.Time, error) {
// 	return m.setExp, m.setErr
// }
// func (m *mockAuthStore) RevokeUser(ctx context.Context, logger *slog.Logger, sessionId string) error {
// 	return m.revokeErr
// }
// func (m *mockAuthStore) GetUserByProvider(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, provider string, sub string) (*domain.LightUser, error) {
// 	return nil, errors.New("not implemented")
// }
// func (m *mockAuthStore) CreateOrLinkOauth(ctx context.Context, logger *slog.Logger, cfg *configs.EnvData, id_or_sub string, email string, name string, provider string) (*domain.LightUser, error) {
// 	return nil, errors.New("not implemented")
// }
// func (m *mockAuthStore) GetUser(ctx context.Context, logger *slog.Logger, sessionId string) (*domain.LightUser, bool) {
// 	return nil, false
// }

// func TestRegister_Success(t *testing.T) {
// 	m := &mockAuthStore{}
// 	cfg := &configs.EnvData{PRODUCTION: false}
// 	r := gin.New()
// 	v1 := r.Group("/api/v1")
// 	auth := v1.Group("/auth")
// 	auth.POST("/register", middlewares.RegisterValidationMiddleware, handlers.HandleRegister(cfg, m))

// 	body := map[string]string{"full_name": "John Doe", "email": "john@example.com", "password": "supersecret"}
// 	b, _ := json.Marshal(body)
// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(b))
// 	req.Header.Set("Content-Type", "application/json")
// 	rw := httptest.NewRecorder()
// 	r.ServeHTTP(rw, req)

// 	if rw.Code != http.StatusCreated {
// 		t.Fatalf("expected status 201 got %d body: %s", rw.Code, rw.Body.String())
// 	}
// 	var res map[string]interface{}
// 	_ = json.Unmarshal(rw.Body.Bytes(), &res)
// 	if res["success"] != true {
// 		t.Fatalf("expected success true")
// 	}
// 	if m.created == nil {
// 		t.Fatalf("expected CreateUser to be called")
// 	}
// 	if m.created.Password == "supersecret" {
// 		t.Fatalf("expected password to be hashed")
// 	}
// }

// func TestRegister_Duplicate(t *testing.T) {
// 	m := &mockAuthStore{createErr: errors.New("duplicate key value violates unique constraint")}
// 	cfg := &configs.EnvData{PRODUCTION: false}
// 	r := gin.New()
// 	v1 := r.Group("/api/v1")
// 	auth := v1.Group("/auth")
// 	auth.POST("/register", middlewares.RegisterValidationMiddleware, handlers.HandleRegister(cfg, m))

// 	body := map[string]string{"full_name": "John Doe", "email": "john@example.com", "password": "supersecret"}
// 	b, _ := json.Marshal(body)
// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(b))
// 	req.Header.Set("Content-Type", "application/json")
// 	rw := httptest.NewRecorder()
// 	r.ServeHTTP(rw, req)

// 	if rw.Code != http.StatusConflict {
// 		t.Fatalf("expected status 409 got %d body: %s", rw.Code, rw.Body.String())
// 	}
// }

// func TestRegister_BadPayload(t *testing.T) {
// 	m := &mockAuthStore{}
// 	cfg := &configs.EnvData{PRODUCTION: false}
// 	r := gin.New()
// 	v1 := r.Group("/api/v1")
// 	auth := v1.Group("/auth")
// 	auth.POST("/register", middlewares.RegisterValidationMiddleware, handlers.HandleRegister(cfg, m))

// 	body := map[string]string{"full_name": "John", "email": "john@example.com", "password": "short"}
// 	b, _ := json.Marshal(body)
// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(b))
// 	req.Header.Set("Content-Type", "application/json")
// 	rw := httptest.NewRecorder()
// 	r.ServeHTTP(rw, req)

// 	if rw.Code != http.StatusBadRequest {
// 		t.Fatalf("expected status 400 got %d body: %s", rw.Code, rw.Body.String())
// 	}
// }

// func TestLogin_NonExistent(t *testing.T) {
// 	m := &mockAuthStore{getErr: errors.New("not found")}
// 	cfg := &configs.EnvData{PRODUCTION: false}
// 	r := gin.New()
// 	v1 := r.Group("/api/v1")
// 	auth := v1.Group("/auth")
// 	auth.POST("/login", middlewares.LoginValidationMiddleware, handlers.HandleLogin(cfg, m))

// 	body := map[string]string{"email": "noone@example.com", "password": "whatever"}
// 	b, _ := json.Marshal(body)
// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(b))
// 	req.Header.Set("Content-Type", "application/json")
// 	rw := httptest.NewRecorder()
// 	r.ServeHTTP(rw, req)

// 	if rw.Code != http.StatusUnauthorized {
// 		t.Fatalf("expected 401 got %d body: %s", rw.Code, rw.Body.String())
// 	}
// }

// func TestLogin_Unverified(t *testing.T) {
// 	user := &models.User{Email: "u@example.com", Verified: false}
// 	m := &mockAuthStore{getUser: user}
// 	cfg := &configs.EnvData{PRODUCTION: false}
// 	r := gin.New()
// 	v1 := r.Group("/api/v1")
// 	auth := v1.Group("/auth")
// 	auth.POST("/login", middlewares.LoginValidationMiddleware, handlers.HandleLogin(cfg, m))

// 	body := map[string]string{"email": "u@example.com", "password": "whatever"}
// 	b, _ := json.Marshal(body)
// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(b))
// 	req.Header.Set("Content-Type", "application/json")
// 	rw := httptest.NewRecorder()
// 	r.ServeHTTP(rw, req)

// 	if rw.Code != http.StatusForbidden {
// 		t.Fatalf("expected 403 got %d body: %s", rw.Code, rw.Body.String())
// 	}
// }

// func TestLogin_WrongPassword(t *testing.T) {
// 	hash, _ := bcrypt.GenerateFromPassword([]byte("rightpass"), bcrypt.DefaultCost)
// 	user := &models.User{Email: "u@example.com", Verified: true, Password: string(hash)}
// 	m := &mockAuthStore{getUser: user}
// 	cfg := &configs.EnvData{PRODUCTION: false}
// 	r := gin.New()
// 	v1 := r.Group("/api/v1")
// 	auth := v1.Group("/auth")
// 	auth.POST("/login", middlewares.LoginValidationMiddleware, handlers.HandleLogin(cfg, m))

// 	body := map[string]string{"email": "u@example.com", "password": "wrongpass"}
// 	b, _ := json.Marshal(body)
// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(b))
// 	req.Header.Set("Content-Type", "application/json")
// 	rw := httptest.NewRecorder()
// 	r.ServeHTTP(rw, req)

// 	if rw.Code != http.StatusUnauthorized {
// 		t.Fatalf("expected 401 got %d body: %s", rw.Code, rw.Body.String())
// 	}
// }

// func TestLogin_Success(t *testing.T) {
// 	hash, _ := bcrypt.GenerateFromPassword([]byte("rightpass"), bcrypt.DefaultCost)
// 	user := &models.User{Email: "u@example.com", Verified: true, Password: string(hash)}
// 	m := &mockAuthStore{getUser: user, setExp: time.Now().Add(1 * time.Hour)}
// 	cfg := &configs.EnvData{PRODUCTION: false}
// 	r := gin.New()
// 	v1 := r.Group("/api/v1")
// 	auth := v1.Group("/auth")
// 	auth.POST("/login", middlewares.LoginValidationMiddleware, handlers.HandleLogin(cfg, m))

// 	body := map[string]string{"email": "u@example.com", "password": "rightpass"}
// 	b, _ := json.Marshal(body)
// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(b))
// 	req.Header.Set("Content-Type", "application/json")
// 	rw := httptest.NewRecorder()
// 	r.ServeHTTP(rw, req)

// 	if rw.Code != http.StatusOK {
// 		t.Fatalf("expected 200 got %d body: %s", rw.Code, rw.Body.String())
// 	}
// 	if rw.Header().Get("Set-Cookie") == "" {
// 		t.Fatalf("expected Set-Cookie header to be present")
// 	}
// }

// func TestLogout_Success(t *testing.T) {
// 	m := &mockAuthStore{revokeErr: nil}
// 	cfg := &configs.EnvData{PRODUCTION: false}
// 	r := gin.New()
// 	v1 := r.Group("/api/v1")
// 	auth := v1.Group("/auth")
// 	// middleware to set sessionId in context as AuthMiddleware would
// 	auth.Use(func(c *gin.Context) {
// 		c.Set("sessionId", "sess-1")
// 		c.Next()
// 	})
// 	auth.POST("/logout", handlers.HandleLogout(m, cfg))

// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
// 	rw := httptest.NewRecorder()
// 	r.ServeHTTP(rw, req)

// 	if rw.Code != http.StatusNoContent {
// 		t.Fatalf("expected 204 got %d body: %s", rw.Code, rw.Body.String())
// 	}
// 	if rw.Header().Get("Set-Cookie") == "" {
// 		t.Fatalf("expected Set-Cookie header to be present")
// 	}
// }
