package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/domain"
	"github.com/ajaka/redir/internal/errs"
	"github.com/ajaka/redir/internal/store"
	"github.com/ajaka/redir/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

func HandleRegister(cfg *configs.EnvData, store store.AuthStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		logger.Info("beginning registration for user")
		val, _ := c.Get("reg")
		RegisterRequestBody, ok := val.(*domain.
			CreateUserDetails)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			logger.Error("couldn't get user registration details from context")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(RegisterRequestBody.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			logger.Error("couldn't hash user password", "error", err.Error())
			return
		}
		RegisterRequestBody.Password = string(hash)
		err = store.CreateUser(c.Request.Context(), logger, RegisterRequestBody, cfg)
		if err != nil {
			if errors.Is(err, errs.ErrDuplicateEmail) {
				c.JSON(http.StatusConflict, gin.H{"success": false, "message": "Email is already registered"})
				logger.Warn("email unique constraint conflict during registration", "email", RegisterRequestBody.Email)
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			logger.Error("failed to create user", "error", err.Error())
			return
		}
		token := utils.GenCleanedUpUUid()
		err = store.SetVerificationUser(c.Request.Context(), logger, RegisterRequestBody.Email, token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		// Implement email later?
		logger.Info("user registered successfully", "email", RegisterRequestBody.Email)
		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "User registered successfully, please proceed to login"})
	}
}

func HandleLogin(cfg *configs.EnvData, store store.AuthStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		logger.Info("starting user login attempt")

		val, _ := c.Get("login")
		LoginRequestBody, ok := val.(*domain.
			LoginUserDetails)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			logger.Error("couldn't get user login details from context")
			return
		}
		user, err := store.GetUserByEmail(c.Request.Context(), logger, cfg, LoginRequestBody.Email)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid credentials"})
			logger.Warn("login attempt with non-existent email", "email", LoginRequestBody.Email)
			return
		}
		if !user.Verified {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Please verify email before logging in"})
			logger.Warn("login attempt with unverified email", "email", user.Email)
			return
		}
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(LoginRequestBody.Password))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid credentials"})
			logger.Warn("login attempt with wrong password", "email", LoginRequestBody.Email)
			return
		}
		lightUser := domain.
			LightUser{
			Id:    user.Id,
			Email: user.Email,
			Admin: user.Admin,
			Paid:  user.Paid,
		}

		sessionCookie, err := utils.PerformLoginActivity(c.Request.Context(), store, cfg, logger, &lightUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong, couldn't sign you in"})
			logger.Error("failed to create session", "email", user.Email, "error", err.Error())
			return
		}
		setCookieFromHttpCookie(c, sessionCookie)
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "User logged in successfully"})
		logger.Info("user logged in successfully", "email", user.Email)
	}
}

func HandleVerify(store store.AuthStore, cfg *configs.EnvData) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		email := c.GetString("email")
		err := store.SetUserVerified(c.Request.Context(), logger, email)
		if err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid data provided"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		c.Redirect(http.StatusFound, cfg.CLIENT_LOGIN_URL)
	}
}

func HandleLogout(store store.AuthStore, cfg *configs.EnvData) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		val, _ := c.Get("sessionId")
		sessionId, _ := val.(string)
		err := store.RevokeUser(c.Request.Context(), logger, sessionId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			logger.Error("failed to revoke session", "session_id", sessionId, "error", err.Error())
			return
		}
		exp := time.Now().Add(-1 * time.Second)
		sessioncookie := utils.SetAndGetCookieDetails("sessionId", "", cfg.PRODUCTION, exp)
		setCookieFromHttpCookie(c, sessioncookie)
		logger.Info("user logged out successfully")
		c.Status(http.StatusNoContent)
	}
}

type GoogleOauth struct {
	o *oauth2.Config
}

func InitGoogleOauth(cfg *configs.EnvData) *GoogleOauth {
	c := oauth2.Config{
		ClientID:     cfg.GOOGLE_CLIENT_ID,
		ClientSecret: cfg.GOOGLE_CLIENT_SECRET,
		RedirectURL:  cfg.GOOGLE_REDIRECT_URL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	return &GoogleOauth{
		o: &c,
	}
}

func (g *GoogleOauth) HandleRedirectToGoogle(cfg *configs.EnvData) gin.HandlerFunc {
	return func(c *gin.Context) {
		// logger := utils.GetLogger(c)
		state := utils.GenUUID()
		exp := time.Now().Add(15 * time.Minute)
		cookie := utils.SetAndGetCookieDetails("state", state, cfg.PRODUCTION, exp)
		setCookieFromHttpCookie(c, cookie)
		c.Redirect(http.StatusFound, g.o.AuthCodeURL(state))
	}
}

func (g *GoogleOauth) HandleGoogleCallback(cfg *configs.EnvData, store store.AuthStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := "google"
		logger := utils.GetLogger(c)
		state, err := c.Cookie("state")
		returnedState := c.Query("state")
		if err != nil || state != returnedState {
			logger.Warn("invalid or expired state token during oauth callback", "provider", provider)
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid or expired state token"})
			return
		}
		code := c.Query("code")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		token, err := g.o.Exchange(ctx, code)
		if err != nil {
			logger.Error("failed to exchange oauth code", "provider", provider, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		client := g.o.Client(ctx, token)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v3/userinfo", nil)
		if err != nil {
			logger.Error("failed to create userinfo request", "provider", provider, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			logger.Error("failed to get user info from provider", "provider", provider, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to get user info"})
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			logger.Error("non-ok status from provider userinfo endpoint", "provider", provider, "status", resp.StatusCode)
			c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "failed to get user info"})
			return
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("failed to read userinfo response body", "provider", provider, "error", err.Error())
			c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "failed to get user info"})
			return
		}
		var user domain.GoogleUser
		if err := json.Unmarshal(data, &user); err != nil {
			logger.Error("failed to unmarshal userinfo response", "provider", provider, "error", err.Error())
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if !user.VerifiedEmail {
			logger.Warn("oauth callback with unverified email", "provider", provider, "email", user.Email)
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Email provided is unverified"})
			return
		}
		u, err := store.GetUserByProvider(c.Request.Context(), logger, cfg, provider, user.ID)
		if err != nil {
			if err == pgx.ErrNoRows {
				u, err = store.CreateOrLinkOauth(c.Request.Context(), logger, cfg, user.ID, user.Email, user.Name, provider)
				if err != nil {
					logger.Error("failed to create or link oauth user", "provider", provider, "email", user.Email, "error", err.Error())
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
					return
				}
			} else {
				logger.Error("failed to get user by provider", "provider", provider, "error", err.Error())
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
				return
			}
		}
		sessionIdCookie, err := utils.PerformLoginActivity(c.Request.Context(), store, cfg, logger, u)
		if err != nil {
			logger.Error("failed to create session after oauth login", "provider", provider, "email", user.Email, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong, couldn't sign you in"})
			return
		}
		setCookieFromHttpCookie(c, sessionIdCookie)
		logger.Info("user logged in via oauth", "provider", provider, "email", user.Email)
		c.Redirect(http.StatusFound, cfg.CLIENT_DASHBOARD)
	}
}

type GithubOauth struct {
	o *oauth2.Config
}

func InitGithubOauth(cfg *configs.EnvData) *GithubOauth {
	c := oauth2.Config{
		ClientID:     cfg.GITHUB_CLIENT_ID,
		ClientSecret: cfg.GITHUB_CLIENT_SECRET,
		RedirectURL:  cfg.GITHUB_REDIRECT_URL,
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     github.Endpoint,
	}
	return &GithubOauth{
		o: &c,
	}
}
func (g *GithubOauth) HandleRedirectToGithub(cfg *configs.EnvData) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		state := utils.GenUUID()
		exp := time.Now().Add(15 * time.Minute)
		cookie := utils.SetAndGetCookieDetails("state", state, cfg.PRODUCTION, exp)
		setCookieFromHttpCookie(c, cookie)
		logger.Info("redirecting to github oauth", "state", state)
		c.Redirect(http.StatusFound, g.o.AuthCodeURL(state))
	}
}

// func (g *GithubOauth) HandleGithubCallback(cfg *configs.EnvData, store *store.Store) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		provider := "github"
// 		logger := utils.GetLogger(c)
// 		state, err := c.Cookie("state")
// 		returnedState := c.Query("state")
// 		if err != nil || state != returnedState {
// 			logger.Warn("invalid or expired state token during oauth callback", "provider", provider)
// 			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid or expired state token"})
// 			return
// 		}
// 		code := c.Query("code")
// 		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
// 		defer cancel()

// 		token, err := g.o.Exchange(ctx, code)
// 		if err != nil {
// 			logger.Error("failed to exchange oauth code", "provider", provider, "error", err.Error())
// 			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
// 			return
// 		}
// 		client := g.o.Client(ctx, token)
// 		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
// 		if err != nil {
// 			logger.Error("failed to create user request", "provider", provider, "error", err.Error())
// 			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
// 			return
// 		}
// 		resp, err := client.Do(req)
// 		if err != nil {
// 			logger.Error("failed to get user info from provider", "provider", provider, "error", err.Error())
// 			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to get user info"})
// 			return
// 		}
// 		defer resp.Body.Close()
// 		if resp.StatusCode != http.StatusOK {
// 			logger.Error("non-ok status from provider userinfo endpoint", "provider", provider, "status", resp.StatusCode)
// 			c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "failed to get user info"})
// 			return
// 		}
// 		data, err := io.ReadAll(resp.Body)
// 		if err != nil {
// 			logger.Error("failed to read userinfo response body", "provider", provider, "error", err.Error())
// 			c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "failed to get user info"})
// 			return
// 		}
// 		var user domain.GithubUser
// 		if err := json.Unmarshal(data, &user); err != nil {
// 			logger.Error("failed to unmarshal userinfo response", "provider", provider, "error", err.Error())
// 			c.AbortWithStatus(http.StatusInternalServerError)
// 			return
// 		}
// 		u, err := store.GetUserByProvider(c.Request.Context(), logger, cfg, provider, strconv.Itoa(user.ID))
// 		if err != nil {
// 			if err == pgx.ErrNoRows {
// 				u, err = store.CreateOrLinkOauth(c.Request.Context(), logger, cfg, strconv.Itoa(user.ID), user.Email, user.Login, provider)
// 				if err != nil {
// 					logger.Error("failed to create or link oauth user", "provider", provider, "email", user.Email, "error", err.Error())
// 					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
// 					return
// 				}
// 			} else {
// 				logger.Error("failed to get user by provider", "provider", provider, "error", err.Error())
// 				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
// 				return
// 			}
// 		}
// 		sessionIdCookie, err := utils.PerformLoginActivity(c.Request.Context(), store, cfg, logger, u)
// 		if err != nil {
// 			logger.Error("failed to create session after oauth login", "provider", provider, "email", user.Email, "error", err.Error())
// 			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong, couldn't sign you in"})
// 			return
// 		}
// 		c.SetCookieData(sessionIdCookie)
// 		logger.Info("user logged in via oauth", "provider", provider, "email", user.Email)
// 		c.Redirect(http.StatusFound, cfg.CLIENT_DASHBOARD)
// 	}
// }
