package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ajaka/redir/internal/cache"
	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/database"
	"github.com/ajaka/redir/internal/middlewares"
	"github.com/ajaka/redir/internal/repository"
	"github.com/ajaka/redir/internal/routes"
	"github.com/ajaka/redir/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type authResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func setupAuthRouter(t *testing.T) (*gin.Engine, *cache.Sredis, *pgxpool.Pool, *configs.EnvData) {
	t.Helper()
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	envPath := filepath.Join(basepath, "..", ".env")
	err := godotenv.Load(envPath)
	if err != nil {
		log.Panic(err)
	}

	cfg := &configs.EnvData{
		DATABASE_URL:   os.Getenv("DATABASE_URL"),
		REDIS_ADDR:     os.Getenv("REDIS_ADDR"),
		REDIS_PASSWORD: os.Getenv("REDIS_PASSWORD"),
		PRODUCTION:     false,
	}

	if cfg.DATABASE_URL == "" {
		t.Fatal("DATABASE_URL environment variable must be set for integration tests")
	}
	if cfg.REDIS_ADDR == "" {
		t.Fatal("REDIS_ADDR environment variable must be set for integration tests")
	}

	logger := slog.Default()
	ctx := context.Background()

	rdb := cache.InitializeRedis(ctx, cfg, logger)
	t.Cleanup(func() {
		rdb.Clean()
	})

	pool := database.ConnectDB(ctx, logger, cfg.DATABASE_URL)
	t.Cleanup(func() {
		pool.Close()
	})

	repo := repository.InitializeRepository(pool)
	store := store.InitializeStore(rdb, repo, cfg)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middlewares.GenAndAttachRequestIdMiddleware())
	router.Use(middlewares.AttachLoggerToContext(cfg))
	router.Use(middlewares.PerformBasicRequestCycleCalculations())
	router.Use(gin.Recovery())

	v1 := router.Group("/api/v1")
	routes.AuthRoutes(v1, cfg, store)

	return router, rdb, pool, cfg
}

func TestAuthFlowIntegration(t *testing.T) {
	router, rdb, _, _ := setupAuthRouter(t)

	email := fmt.Sprintf("test-flow-%s@example.com", uuid.NewString())
	password := "password123"
	fullName := "Integration Test User"

	// Register
	regBody, err := json.Marshal(map[string]string{
		"full_name": fullName,
		"email":     email,
		"password":  password,
	})
	if err != nil {
		t.Fatalf("failed to marshal register body: %v", err)
	}
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	router.ServeHTTP(regRec, regReq)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("register failed: expected %d got %d body=%s", http.StatusCreated, regRec.Code, regRec.Body.String())
	}
	var regResp authResponse
	if err := json.NewDecoder(regRec.Body).Decode(&regResp); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}
	if !regResp.Success {
		t.Fatalf("register returned success=false, body=%s", regRec.Body.String())
	}

	// Login should be forbidden until verification
	loginBody, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		t.Fatalf("failed to marshal login body: %v", err)
	}
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusForbidden {
		t.Fatalf("expected login to be forbidden for unverified user: got %d body=%s", loginRec.Code, loginRec.Body.String())
	}

	// read token from redis (set by register handler)
	ctx := context.Background()
	token, err := rdb.GetVerificationTokenByEmail(ctx, email)
	if err != nil {
		t.Fatalf("failed to read verification token from redis: %v", err)
	}
	// call verify endpoint
	vReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token="+token, nil)
	vRec := httptest.NewRecorder()
	router.ServeHTTP(vRec, vReq)
	if vRec.Code != http.StatusFound {
		t.Fatalf("expected redirect after successful verification, got %d body=%s", vRec.Code, vRec.Body.String())
	}

	// now login should succeed
	loginReq2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq2.Header.Set("Content-Type", "application/json")
	loginRec2 := httptest.NewRecorder()
	router.ServeHTTP(loginRec2, loginReq2)
	if loginRec2.Code != http.StatusOK {
		t.Fatalf("login after verification failed: expected %d got %d body=%s", http.StatusOK, loginRec2.Code, loginRec2.Body.String())
	}
	var sessionCookie *http.Cookie
	for _, c := range loginRec2.Result().Cookies() {
		if c.Name == "sessionId" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected session cookie from login after verification")
	}
}
