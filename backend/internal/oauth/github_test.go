package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAuthorizeURL_FormatsRequiredParams(t *testing.T) {
	c := &Client{ClientID: "Iv1.abc", RedirectURI: "https://x/cb"}
	got, err := url.Parse(c.AuthorizeURL("state-123"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Host != "github.com" || got.Path != "/login/oauth/authorize" {
		t.Errorf("wrong host/path: %s", got)
	}
	q := got.Query()
	if q.Get("client_id") != "Iv1.abc" {
		t.Errorf("client_id=%q", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != "https://x/cb" {
		t.Errorf("redirect_uri=%q", q.Get("redirect_uri"))
	}
	if q.Get("state") != "state-123" {
		t.Errorf("state=%q", q.Get("state"))
	}
	// GitHub App 模式不传 scope
	if q.Has("scope") {
		t.Errorf("scope must NOT be set (GitHub App; permissions configured in App settings)")
	}
}

func TestExchangeCode_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login/oauth/access_token" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = r.ParseForm()
		if r.Form.Get("code") != "abc" {
			t.Errorf("code = %q", r.Form.Get("code"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "ghu_xxx", TokenType: "bearer"})
	}))
	defer srv.Close()

	// 替换包内常量 URL：临时把 tokenURL 重写到 test server
	// 直接 monkey patch package var；通过为 Client 暴露可注入 URL 更优，但当前实现是 const，所以测试用 transport mock
	c := &Client{
		ClientID: "cid", ClientSecret: "csec", RedirectURI: "https://x/cb",
		HTTPClient: srv.Client(),
	}
	c.HTTPClient.Transport = rewriteHost{base: srv.URL, real: c.HTTPClient.Transport}

	tok, err := c.ExchangeCode(context.Background(), "abc")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if tok.AccessToken != "ghu_xxx" {
		t.Errorf("token = %q", tok.AccessToken)
	}
}

func TestExchangeCode_GitHubReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GitHub 200 内含 error 而非 4xx 时也要被识别为失败
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"bad_verification_code","error_description":"The code is incorrect."}`))
	}))
	defer srv.Close()
	c := &Client{ClientID: "cid", ClientSecret: "csec", RedirectURI: "https://x/cb", HTTPClient: srv.Client()}
	c.HTTPClient.Transport = rewriteHost{base: srv.URL, real: c.HTTPClient.Transport}

	_, err := c.ExchangeCode(context.Background(), "bad")
	if err == nil {
		t.Fatal("expected error on empty access_token")
	}
	if !strings.Contains(err.Error(), "bad_verification_code") {
		t.Errorf("err missing github error: %v", err)
	}
}

func TestFetchUser_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("auth header = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42,"login":"octocat","avatar_url":"https://a","name":"Mona"}`))
	}))
	defer srv.Close()
	c := &Client{ClientID: "cid", HTTPClient: srv.Client()}
	c.HTTPClient.Transport = rewriteHost{base: srv.URL, real: c.HTTPClient.Transport}

	u, err := c.FetchUser(context.Background(), "tok")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if u.ID != 42 || u.Login != "octocat" {
		t.Errorf("user = %+v", u)
	}
}

// rewriteHost 把所有外发请求的 host 改写到 test server；保留 path
type rewriteHost struct {
	base string
	real http.RoundTripper
}

func (r rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	u, _ := url.Parse(r.base)
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host
	rt := r.real
	if rt == nil {
		rt = http.DefaultTransport
	}
	return rt.RoundTrip(req)
}
