package mw

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testToken = "test-secret-token"

func TestBearerAuth(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func(r *http.Request)
		expectedStatus int
		shouldPassNext bool
	}{
		{
			name: "valid_bearer_token",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+testToken)
			},
			expectedStatus: http.StatusOK,
			shouldPassNext: true,
		},
		{
			name: "valid_bearer_lowercase",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "bearer "+testToken)
			},
			expectedStatus: http.StatusOK,
			shouldPassNext: true,
		},
		{
			name: "valid_cookie",
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: AuthCookieName, Value: testToken})
			},
			expectedStatus: http.StatusOK,
			shouldPassNext: true,
		},
		{
			name: "cookie_takes_precedence_over_invalid_bearer",
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: AuthCookieName, Value: testToken})
				r.Header.Set("Authorization", "Bearer wrong-token")
			},
			expectedStatus: http.StatusOK,
			shouldPassNext: true,
		},
		{
			name: "invalid_bearer_token",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer wrong-token")
			},
			expectedStatus: http.StatusUnauthorized,
			shouldPassNext: false,
		},
		{
			name: "invalid_cookie_falls_back_to_bearer",
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: AuthCookieName, Value: "wrong"})
				r.Header.Set("Authorization", "Bearer "+testToken)
			},
			expectedStatus: http.StatusOK,
			shouldPassNext: true,
		},
		{
			name: "invalid_cookie_no_bearer",
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: AuthCookieName, Value: "wrong"})
			},
			expectedStatus: http.StatusUnauthorized,
			shouldPassNext: false,
		},
		{
			name:           "no_auth_provided",
			setupRequest:   func(r *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
			shouldPassNext: false,
		},
		{
			name: "basic_auth_not_accepted",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Basic abc123")
			},
			expectedStatus: http.StatusUnauthorized,
			shouldPassNext: false,
		},
		{
			name: "bearer_no_space",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer"+testToken)
			},
			expectedStatus: http.StatusUnauthorized,
			shouldPassNext: false,
		},
		{
			name: "bearer_only_no_token",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer")
			},
			expectedStatus: http.StatusUnauthorized,
			shouldPassNext: false,
		},
		{
			name: "empty_bearer_value",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer ")
			},
			expectedStatus: http.StatusUnauthorized,
			shouldPassNext: false,
		},
		{
			name: "bearer_with_extra_spaces",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer  "+testToken)
			},
			expectedStatus: http.StatusUnauthorized,
			shouldPassNext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			middleware := BearerAuth(testToken)
			handler := middleware(next)

			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			tt.setupRequest(req)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.Equal(t, tt.shouldPassNext, nextCalled)
		})
	}
}

func TestLoginHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           interface{}
		secureCookie   bool
		expectedStatus int
		checkCookie    bool
	}{
		{
			name:           "successful_login",
			method:         http.MethodPost,
			body:           map[string]string{"token": testToken},
			secureCookie:   false,
			expectedStatus: http.StatusOK,
			checkCookie:    true,
		},
		{
			name:           "successful_login_secure_cookie",
			method:         http.MethodPost,
			body:           map[string]string{"token": testToken},
			secureCookie:   true,
			expectedStatus: http.StatusOK,
			checkCookie:    true,
		},
		{
			name:           "invalid_token",
			method:         http.MethodPost,
			body:           map[string]string{"token": "wrong-token"},
			secureCookie:   false,
			expectedStatus: http.StatusUnauthorized,
			checkCookie:    false,
		},
		{
			name:           "empty_token",
			method:         http.MethodPost,
			body:           map[string]string{"token": ""},
			secureCookie:   false,
			expectedStatus: http.StatusUnauthorized,
			checkCookie:    false,
		},
		{
			name:           "method_not_allowed_get",
			method:         http.MethodGet,
			body:           nil,
			secureCookie:   false,
			expectedStatus: http.StatusMethodNotAllowed,
			checkCookie:    false,
		},
		{
			name:           "method_not_allowed_put",
			method:         http.MethodPut,
			body:           map[string]string{"token": testToken},
			secureCookie:   false,
			expectedStatus: http.StatusMethodNotAllowed,
			checkCookie:    false,
		},
		{
			name:           "invalid_json",
			method:         http.MethodPost,
			body:           "not json",
			secureCookie:   false,
			expectedStatus: http.StatusBadRequest,
			checkCookie:    false,
		},
		{
			name:           "missing_token_field",
			method:         http.MethodPost,
			body:           map[string]string{"wrong_field": "value"},
			secureCookie:   false,
			expectedStatus: http.StatusUnauthorized,
			checkCookie:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := LoginHandler(testToken, tt.secureCookie)

			var body *bytes.Buffer
			if tt.body != nil {
				switch v := tt.body.(type) {
				case string:
					body = bytes.NewBufferString(v)
				default:
					b, _ := json.Marshal(v)
					body = bytes.NewBuffer(b)
				}
			} else {
				body = bytes.NewBuffer(nil)
			}

			req := httptest.NewRequest(tt.method, "/api/auth/login", body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.checkCookie {
				cookies := rec.Result().Cookies()
				require.Len(t, cookies, 1)
				cookie := cookies[0]

				assert.Equal(t, AuthCookieName, cookie.Name)
				assert.Equal(t, testToken, cookie.Value)
				assert.Equal(t, "/", cookie.Path)
				assert.True(t, cookie.HttpOnly)
				assert.Equal(t, tt.secureCookie, cookie.Secure)
				assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
				assert.Equal(t, 86400*30, cookie.MaxAge)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Equal(t, "ok", resp["status"])
			}
		})
	}
}

func TestLogoutHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkCookie    bool
	}{
		{
			name:           "successful_logout",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			checkCookie:    true,
		},
		{
			name:           "method_not_allowed_get",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
			checkCookie:    false,
		},
		{
			name:           "method_not_allowed_delete",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
			checkCookie:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := LogoutHandler()

			req := httptest.NewRequest(tt.method, "/api/auth/logout", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.checkCookie {
				cookies := rec.Result().Cookies()
				require.Len(t, cookies, 1)
				cookie := cookies[0]

				assert.Equal(t, AuthCookieName, cookie.Name)
				assert.Equal(t, "", cookie.Value)
				assert.Equal(t, "/", cookie.Path)
				assert.True(t, cookie.HttpOnly)
				assert.Equal(t, -1, cookie.MaxAge)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Equal(t, "ok", resp["status"])
			}
		})
	}
}

func TestLogoutHandler_MultipleLogouts(t *testing.T) {
	handler := LogoutHandler()

	// First logout
	req1 := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Second logout should also succeed
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
}
