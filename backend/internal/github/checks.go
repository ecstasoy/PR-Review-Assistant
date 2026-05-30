package github

import (
	"context"

	gh "github.com/google/go-github/v66/github"
)

// fetchChecks 拉指定 ref 上所有 check-run（GitHub Actions / 第三方 CI 都通过 Checks API 暴露），
// 汇总成单一 CI 状态 + 每条 check 明细。
// ref 通常传 PR HeadSHA；调用方负责降级（抓失败仍要继续主流程）。
func fetchChecks(ctx context.Context, client *gh.Client, owner, repo, ref string) (string, []Check, error) {
	res, _, err := client.Checks.ListCheckRunsForRef(ctx, owner, repo, ref, nil)
	if err != nil {
		return "", nil, err
	}
	runs := res.CheckRuns
	if len(runs) == 0 {
		// 仓库无 CI 配置 / 或 head 还没跑 → 视为 pending
		return CIStatusPending, []Check{}, nil
	}

	checks := make([]Check, 0, len(runs))
	overall := CIStatusPassing
	for _, r := range runs {
		c := Check{
			Name:   r.GetName(),
			Status: mapCheckStatus(r),
			Note:   r.GetOutput().GetSummary(), // 自定义文本，例如覆盖率 "82.4% (-0.3%)"；多数 check 为空
		}
		// duration = completed_at - started_at（毫秒）；未完成 / 缺时间戳时为 0
		started := r.GetStartedAt()
		completed := r.GetCompletedAt()
		if !started.IsZero() && !completed.IsZero() {
			c.DurationMS = int(completed.Sub(started.Time).Milliseconds())
		}
		checks = append(checks, c)

		// 优先级：任一 failing 即整体 failing；任一 pending 把整体降为 pending（除非已 failing）
		switch c.Status {
		case CIStatusFailing:
			overall = CIStatusFailing
		case CIStatusPending:
			if overall != CIStatusFailing {
				overall = CIStatusPending
			}
		}
	}
	return overall, checks, nil
}

// mapCheckStatus 把 GitHub check-run 的 status + conclusion 映射到本包三态。
//
//	completed + success                                          → passing
//	completed + (failure|cancelled|timed_out|action_required)    → failing
//	其它（含 not-completed、neutral、skipped、stale、unknown）       → pending
func mapCheckStatus(r *gh.CheckRun) string {
	if r.GetStatus() != "completed" {
		return CIStatusPending
	}
	switch r.GetConclusion() {
	case "success":
		return CIStatusPassing
	case "failure", "cancelled", "timed_out", "action_required":
		return CIStatusFailing
	default:
		// neutral / skipped / stale / 空字符串 等"既非明确成功也非明确失败"
		return CIStatusPending
	}
}
