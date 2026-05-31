package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestVerifyHMAC_Matches(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	secret := []byte("supersecret")
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !verifyHMAC(sig, body, secret) {
		t.Fatal("expected match")
	}
}

func TestVerifyHMAC_Mismatch(t *testing.T) {
	body := []byte("payload")
	if verifyHMAC("sha256=deadbeef", body, []byte("secret")) {
		t.Fatal("expected mismatch")
	}
}

func TestVerifyHMAC_MissingPrefix(t *testing.T) {
	if verifyHMAC("md5=xyz", []byte("p"), []byte("s")) {
		t.Fatal("non-sha256 prefix should fail")
	}
	if verifyHMAC("", []byte("p"), []byte("s")) {
		t.Fatal("empty sig should fail")
	}
}

func TestWebhook_RejectsBadSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/webhook/github", WebhookGitHub(Deps{}, "secret"))
	req := httptest.NewRequest(http.MethodPost, "/api/webhook/github", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", "sha256=00")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestWebhook_503WhenSecretMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/webhook/github", WebhookGitHub(Deps{}, ""))
	req := httptest.NewRequest(http.MethodPost, "/api/webhook/github", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-GitHub-Event", "pull_request")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status=%d want 503", w.Code)
	}
}

func TestWebhook_IgnoresNonPullRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	secret := "s"
	r.POST("/api/webhook/github", WebhookGitHub(Deps{}, secret))
	body := []byte(`{"zen":"hello"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	req := httptest.NewRequest(http.MethodPost, "/api/webhook/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "ping")
	req.Header.Set("X-Hub-Signature-256", sig)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status=%d want 200", w.Code)
	}
}

func TestWebhook_IgnoresNonOpenedAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	secret := "s"
	r.POST("/api/webhook/github", WebhookGitHub(Deps{}, secret))
	body := []byte(`{"action":"closed","number":1,"pull_request":{"html_url":"x"},"repository":{"owner":{"login":"o"},"name":"r"}}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	req := httptest.NewRequest(http.MethodPost, "/api/webhook/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", sig)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status=%d want 200; body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("ignored_action")) {
		t.Errorf("body should explain ignored_action; got %s", w.Body.String())
	}
}
