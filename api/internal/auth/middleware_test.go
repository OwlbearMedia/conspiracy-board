package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireCSRF(t *testing.T) {
	t.Parallel()

	handler := RequireCSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		method     string
		cookie     string
		header     string
		wantStatus int
	}{
		{"GET passes without token", http.MethodGet, "", "", http.StatusOK},
		{"HEAD passes without token", http.MethodHead, "", "", http.StatusOK},
		{"OPTIONS passes without token", http.MethodOptions, "", "", http.StatusOK},
		{"POST with matching tokens", http.MethodPost, "tok123", "tok123", http.StatusOK},
		{"POST with no cookie or header", http.MethodPost, "", "", http.StatusForbidden},
		{"POST with cookie only", http.MethodPost, "tok123", "", http.StatusForbidden},
		{"POST with header only", http.MethodPost, "", "tok123", http.StatusForbidden},
		{"POST with mismatched tokens", http.MethodPost, "tok123", "tok456", http.StatusForbidden},
		{"DELETE with mismatched tokens", http.MethodDelete, "tok123", "tok456", http.StatusForbidden},
		{"PATCH with matching tokens", http.MethodPatch, "tok123", "tok123", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: CSRFCookie, Value: tt.cookie})
			}
			if tt.header != "" {
				req.Header.Set(CSRFHeader, tt.header)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
