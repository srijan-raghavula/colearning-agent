package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	jwtSecret = loadJWTSecret()
)

func loadJWTSecret() []byte {
	if secret := os.Getenv("COLEARN_JWT_SECRET"); secret != "" {
		if len(secret) < 32 {
			panic("COLEARN_JWT_SECRET must contain at least 32 bytes")
		}
		return []byte(secret)
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic("generate in-memory JWT secret: " + err.Error())
	}
	return secret
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

type Account struct {
	ID              string
	Email           string
	PasswordHash    string
	Role            string
	Name            string
	AcademicProgram string
	Branch          string
	Semester        int
	AcademicYear    string
	Department      string
	TeachingTitle   string
	AuthVersion     int
}

type Identity struct {
	ID          string
	Name        string
	Email       string
	Role        string
	AuthVersion int
}

// IdentityStore resolves signed identities against current account state and
// lets security-sensitive changes revoke all previously issued tokens.
type IdentityStore interface {
	FindAccountByID(context.Context, string) (Account, bool, error)
}

type identityContextKey struct{}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(Identity)
	return identity, ok
}

func IdentityFromRequest(r *http.Request) (Identity, error) {
	tokenString := ""
	if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
		tokenString = strings.TrimPrefix(header, "Bearer ")
	} else if cookie, err := r.Cookie("auth_token"); err == nil {
		tokenString = cookie.Value
	}
	if tokenString == "" {
		return Identity{}, errors.New("authentication required")
	}
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return Identity{}, err
	}
	identity := Identity{
		ID:    claimString(claims, "sub"),
		Name:  claimString(claims, "name"),
		Email: claimString(claims, "email"),
		Role:  claimString(claims, "role"),
	}
	if version, ok := claims["ver"].(float64); ok && version >= 1 {
		identity.AuthVersion = int(version)
	} else {
		// Tokens created before versioned revocation remain valid as version 1.
		identity.AuthVersion = 1
	}
	if identity.ID == "" || identity.Name == "" || identity.Email == "" || (identity.Role != "student" && identity.Role != "teacher" && identity.Role != "faculty") {
		return Identity{}, errors.New("token does not contain a valid identity")
	}
	return identity, nil
}

func IdentityFromRequestWithStore(r *http.Request, store IdentityStore) (Identity, error) {
	identity, err := IdentityFromRequest(r)
	if err != nil || store == nil {
		return identity, err
	}
	account, found, err := store.FindAccountByID(r.Context(), identity.ID)
	if err != nil {
		return Identity{}, err
	}
	if !found || account.AuthVersion != identity.AuthVersion || account.Role != identity.Role {
		return Identity{}, errors.New("session has expired; sign in again")
	}
	identity.Name, identity.Email = account.Name, account.Email
	return identity, nil
}

func claimString(claims map[string]any, key string) string {
	value, _ := claims[key].(string)
	return value
}

type UserStore interface {
	CreateAccount(ctx context.Context, account Account) error
	FindAccountByEmail(ctx context.Context, email string) (Account, bool, error)
}

func SignupWithStore(ctx context.Context, store UserStore, account Account, password string) error {
	if store == nil {
		return errors.New("account store is not configured")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	account.PasswordHash = hash
	return store.CreateAccount(ctx, account)
}

func LoginWithStore(ctx context.Context, store UserStore, email, password string) (map[string]any, string, error) {
	if store == nil {
		return nil, "", errors.New("account store is not configured")
	}
	account, found, err := store.FindAccountByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	if err != nil {
		return nil, "", err
	}
	if !found || !CheckPassword(account.PasswordHash, password) {
		return nil, "", errors.New("invalid credentials")
	}
	token, err := GenerateToken(account.ID, account.Role, account.Email, account.Name, account.AuthVersion)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"user_id": account.ID, "email": account.Email, "role": account.Role, "name": account.Name}, token, nil
}

func GenerateToken(userID, role, email, name string, versions ...int) (string, error) {
	version := 1
	if len(versions) > 0 && versions[0] > 0 {
		version = versions[0]
	}
	claims := jwt.MapClaims{
		"sub":   userID,
		"role":  role,
		"email": email,
		"name":  name,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"ver":   version,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenString string) (map[string]any, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		claimString := func(key string) (string, bool) {
			value, ok := claims[key].(string)
			return value, ok && strings.TrimSpace(value) != ""
		}
		sub, subOK := claimString("sub")
		role, roleOK := claimString("role")
		email, emailOK := claimString("email")
		name, nameOK := claimString("name")
		if !subOK || !roleOK || !emailOK || !nameOK || (role != "student" && role != "teacher" && role != "faculty") {
			return nil, errors.New("invalid identity claims")
		}
		return map[string]any{
			"sub":   sub,
			"role":  role,
			"email": email,
			"name":  name,
			"ver":   claims["ver"],
		}, nil
	}
	return nil, errors.New("invalid token")
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddlewareWithStore(nil, next)
}

func AuthMiddlewareWithStore(store IdentityStore, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := IdentityFromRequestWithStore(r, store)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Bearer realm="CoLearn"`)
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}
		next(w, r.WithContext(WithIdentity(r.Context(), identity)))
	}
}

func RoleRequired(role string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := IdentityFromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}
			userRole := identity.Role
			if userRole != role && !(role == "teacher" && userRole == "faculty") {
				http.Error(w, fmt.Sprintf(`{"error":"forbidden: role %s required"}`, role), http.StatusForbidden)
				return
			}
			next(w, r)
		})
	}
}
