package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/appconfig"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type AuthHandler struct {
	registerUC     *application.RegisterUseCase
	loginUC        *application.LoginUseCase
	assignGateOpUC *application.AssignGateOperatorUseCase
	listGateOpUC   *application.ListAssignedGateOperatorsUseCase
	removeGateOpUC *application.RemoveGateOperatorUseCase
	searchGateOpUC *application.SearchGateOperatorsUseCase
	userRepo       application.UserRepository
	redis          *redis.Client
}

func NewAuthHandler(
	registerUC *application.RegisterUseCase,
	loginUC *application.LoginUseCase,
	assignGateOpUC *application.AssignGateOperatorUseCase,
	listGateOpUC *application.ListAssignedGateOperatorsUseCase,
	removeGateOpUC *application.RemoveGateOperatorUseCase,
	searchGateOpUC *application.SearchGateOperatorsUseCase,
	userRepo application.UserRepository,
	redisClient *redis.Client,
) *AuthHandler {
	return &AuthHandler{
		registerUC:     registerUC,
		loginUC:        loginUC,
		assignGateOpUC: assignGateOpUC,
		listGateOpUC:   listGateOpUC,
		removeGateOpUC: removeGateOpUC,
		searchGateOpUC: searchGateOpUC,
		userRepo:       userRepo,
		redis:          redisClient,
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

type assignGateOpRequest struct {
	Username string `json:"username"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	user, err := h.registerUC.Execute(r.Context(), application.RegisterInput{
		Email:    req.Email,
		Username: req.Username,
		Name:     req.Name,
		Password: req.Password,
		Role:     domain.Role(req.Role),
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmailAlreadyTaken):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrUsernameAlreadyTaken):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrForbiddenRole):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, domain.ErrPasswordTooShort),
			errors.Is(err, domain.ErrInvalidEmail),
			errors.Is(err, domain.ErrNameRequired),
			errors.Is(err, domain.ErrInvalidRole),
			errors.Is(err, domain.ErrUsernameRequired),
			errors.Is(err, domain.ErrInvalidUsername):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, userResponse{
		ID: user.ID, Email: user.Email, Username: user.Username, Name: user.Name, Role: string(user.Role),
	})
}

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

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    out.Token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"token": out.Token,
		"user": userResponse{
			ID: out.User.ID, Email: out.User.Email, Username: out.User.Username, Name: out.User.Name, Role: string(out.User.Role),
		},
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromCtx(r.Context())

	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user tidak ditemukan")
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		ID: user.ID, Email: user.Email, Username: user.Username, Name: user.Name, Role: string(user.Role),
	})
}

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

func (h *AuthHandler) AssignGateOperator(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	organizerID := UserIDFromCtx(r.Context())

	var req assignGateOpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username wajib diisi")
		return
	}

	result, err := h.assignGateOpUC.Execute(r.Context(), application.AssignGateOperatorInput{
		Username:    req.Username,
		EventID:     eventID,
		OrganizerID: organizerID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "user tidak ditemukan")
		case errors.Is(err, domain.ErrNotGateOperator):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrNotEventOrganizer):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) ListGateOperators(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	organizerID := UserIDFromCtx(r.Context())

	operators, err := h.listGateOpUC.Execute(r.Context(), eventID, organizerID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotEventOrganizer):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, operators)
}

func (h *AuthHandler) RemoveGateOperator(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	userID := chi.URLParam(r, "userID")
	organizerID := UserIDFromCtx(r.Context())

	err := h.removeGateOpUC.Execute(r.Context(), userID, eventID, organizerID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotEventOrganizer):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, domain.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "assignment tidak ditemukan")
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *AuthHandler) SearchGateOperators(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	users, err := h.searchGateOpUC.Execute(r.Context(), query)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	res := make([]userResponse, len(users))
	for i, u := range users {
		res[i] = userResponse{
			ID: u.ID, Email: u.Email, Username: u.Username, Name: u.Name, Role: string(u.Role),
		}
	}

	writeJSON(w, http.StatusOK, res)
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
