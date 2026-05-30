package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// 用极小 burst 验证 token bucket 真的拦截
func TestRateLimit_AllowsBurstThenBlocks(t *testing.T) {
	r := gin.New()
	r.Use(RateLimit(RateLimitConfig{RPS: 0.01, Burst: 2}))
	r.GET("/x", func(c *gin.Context) { c.Status(200) })

	srv := httptest.NewServer(r)
	defer srv.Close()

	// 同一 IP 先打两个，应放行；第三个应 429
	for i := range 2 {
		res, err := http.Get(srv.URL + "/x")
		if err != nil {
			t.Fatalf("http.Get failed: %v", err)
		}
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("burst #%d should pass, got %d", i, res.StatusCode)
		}
	}
	res, err := http.Get(srv.URL + "/x")
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 429 {
		t.Errorf("after burst should be 429, got %d", res.StatusCode)
	}
	if ra := res.Header.Get("Retry-After"); ra == "" {
		t.Errorf("missing Retry-After header")
	} else if n, _ := strconv.Atoi(ra); n < 1 {
		t.Errorf("Retry-After should be >=1, got %s", ra)
	}
}

// 不同 IP 应各自独立计数
func TestRateLimit_PerIPIsolation(t *testing.T) {
	r := gin.New()
	r.Use(RateLimit(RateLimitConfig{RPS: 0.01, Burst: 1}))
	r.GET("/x", func(c *gin.Context) { c.Status(200) })

	doReq := func(ip string) int {
		req := httptest.NewRequest("GET", "/x", nil)
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}

	if got := doReq("10.0.0.1"); got != 200 {
		t.Fatalf("ip A first call should be 200, got %d", got)
	}
	if got := doReq("10.0.0.1"); got != 429 {
		t.Fatalf("ip A second call should be 429, got %d", got)
	}
	// 不同 IP 的第一次仍应放行
	if got := doReq("10.0.0.2"); got != 200 {
		t.Fatalf("ip B first call should be 200, got %d (isolation broken)", got)
	}
}
