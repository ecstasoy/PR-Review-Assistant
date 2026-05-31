package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AppJWT 用 App private key (PEM) RS256 签 GitHub 要求的 JWT
// iss = App ID；iat = now（5s 余量防时钟漂移）；exp = now+9min（GitHub 上限 10min）
// 拿 JWT 后调 /app/installations/{id}/access_tokens 换 installation token
func AppJWT(appID int64, privateKeyPEM []byte) (string, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("oauth: parse private key: %w", err)
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Add(-5 * time.Second).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": appID,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return tok.SignedString(key)
}

// InstallationToken /app/installations/{id}/access_tokens 返字段
type InstallationToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GetInstallationToken 用 App JWT 换某个 installation 的 token；token 1h 过期
// 这个 token 让我们以 bot 身份调 GitHub API（不是用户身份）
func (c *Client) GetInstallationToken(ctx context.Context, appJWT string, installationID int64) (*InstallationToken, error) {
	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("oauth: build installation token req: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth: get installation token: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("oauth: installation token %d: %s", res.StatusCode, string(body))
	}
	var tok InstallationToken
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("oauth: parse installation token: %w", err)
	}
	return &tok, nil
}

// GetRepoInstallationID 查某个 repo 上 App 的 installation_id
// /repos/{o}/{r}/installation 用 App JWT 调；404 = 没装；其它非 200 = 调用错
func (c *Client) GetRepoInstallationID(ctx context.Context, appJWT, owner, repo string) (int64, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/installation", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("oauth: build installation lookup: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := c.httpClient().Do(req)
	if err != nil {
		return 0, fmt.Errorf("oauth: installation lookup: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode == http.StatusNotFound {
		return 0, fmt.Errorf("oauth: App not installed on %s/%s", owner, repo)
	}
	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("oauth: installation lookup %d: %s", res.StatusCode, string(body))
	}
	var resp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("oauth: parse installation: %w", err)
	}
	return resp.ID, nil
}
