package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ebk-tech/be-booking-events/internal/appconfig"
	"github.com/ebk-tech/be-booking-events/internal/auth/application"
	"github.com/ebk-tech/be-booking-events/internal/auth/domain"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type AuthHandler struct {
	registerUC     *application.RegisterUseCase
	loginUC        *application.LoginUseCase
	assignGateOpUC *application.AssignGateOperatorUseCase
	userRepo       application.UserRepository
	redis          *redis.Client
}

func NewAuthHandler(
	registerUC *application.RegisterUseCase,
	loginUC *application.LoginUseCase,
	assignGateOpUC *application.AssignGateOperatorUseCase,
	userRepo application.UserRepository,
	redisClient *redis.Client,
) *AuthHandler {
	return &AuthHandler{
		registerUC:     registerUC,
		loginUC:        loginUC,
		assignGateOpUC: assignGateOpUC,
		userRepo:       userRepo,
		redis:          redisClient,
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type assignGateOpRequest struct {
	UserID string `json:"user_id"`
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	user, err := h.registerUC.Execute(r.Context(), application.RegisterInput{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
		Role:     domain.Role(req.Role),
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmailAlreadyTaken):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrPasswordTooShort),
			errors.Is(err, domain.ErrInvalidEmail),
			errors.Is(err, domain.ErrNameRequired),
			errors.Is(err, domain.ErrInvalidRole):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, userResponse{
		ID: user.ID, Email: user.Email, Name: user.Name, Role: string(user.Role),
	})
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	out, err := h.loginUC.Execute(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeInternalError(w, err)
		return
	}

	// Set HttpOnly Cookie untuk keamanan tingkat tinggi
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    out.Token,
		Path:     "/",
		MaxAge:   86400, // 24 jam
		HttpOnly: true,
		Secure:   false, // dev mode (HTTP), true di HTTPS prod
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"token": out.Token,
		"user": userResponse{
			ID: out.User.ID, Email: out.User.Email, Name: out.User.Name, Role: string(out.User.Role),
		},
	})
}

// GET /api/v1/auth/me  (requires AuthMiddleware)
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromCtx(r.Context())

	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user tidak ditemukan")
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		ID: user.ID, Email: user.Email, Name: user.Name, Role: string(user.Role),
	})
}

// POST /api/v1/auth/logout — invalidasi token via Redis blocklist dan hapus cookie
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var token string
	raw := r.Header.Get("Authorization")
	if strings.HasPrefix(raw, "Bearer ") {
		token = strings.TrimPrefix(raw, "Bearer ")
	} else if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
		token = cookie.Value
	}

	if token != "" && h.redis != nil {
		key := "jwt_blocklist:" + token
		h.redis.Set(context.Background(), key, "1", 25*time.Hour)
	}

	// Hapus HttpOnly Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

// POST /api/v1/events/{eventID}/gate-operators  (requires ORGANIZER)
func (h *AuthHandler) AssignGateOperator(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")

	var req assignGateOpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id wajib diisi")
		return
	}

	err := h.assignGateOpUC.Execute(r.Context(), application.AssignGateOperatorInput{
		GateOperatorUserID: req.UserID,
		EventID:            eventID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "user tidak ditemukan")
		case errors.Is(err, domain.ErrNotGateOperator):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "assigned"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeInternalError(w http.ResponseWriter, err error) {
	msg := "internal server error"
	if appconfig.IsDebug() {
		msg = err.Error()
	}
	writeError(w, http.StatusInternalServerError, msg)
}
