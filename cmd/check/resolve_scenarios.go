package main

import (
	"fmt"
	"strings"
	"sync"
)

// resolvePath 是登记关注项处置的入口。
const resolvePath = "/hives/" + demoHiveID + "/concern-resolutions"

// checkResolveConcerns 覆盖处置登记、汇总区分、新旧关注项隔离、
// 重复处置拒绝，以及并发处置时最终状态唯一。
func checkResolveConcerns(c *client) error {
	if err := seedDefault(c); err != nil {
		return err
	}

	// 没有巡检时不能登记处置。
	if _, err := c.expect("POST", resolvePath, map[string]any{
		"concerns": []string{"mite_load"}, "resolvedBy": "协调人-陆岩",
		"resolution": "撒粉治螨",
	}, 409, `"code":"invalid_state"`); err != nil {
		return err
	}

	// 第一次巡检派生出 mite_load、temper_risk 两个关注项。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 3, "broodPattern": "patchy",
		"temper": "aggressive", "miteCount": 12, "note": "西侧巢门观察",
	}, 201, `"inspection-000001"`, `"mite_load"`, `"temper_risk"`); err != nil {
		return err
	}

	// 不能处置最近一次巡检没有派生出的关注项。
	if _, err := c.expect("POST", resolvePath, map[string]any{
		"concerns": []string{"brood_gap"}, "resolvedBy": "协调人-陆岩",
		"resolution": "补饲观察",
	}, 400, `"code":"invalid_request"`, `"field":"concernCode"`); err != nil {
		return err
	}

	// 处置其中一个关注项，响应需带回来源巡检编号。
	if _, err := c.expect("POST", resolvePath, map[string]any{
		"concerns": []string{"mite_load"}, "resolvedBy": "协调人-陆岩",
		"resolution": "撒粉治螨并复查", "resolvedAt": "2026-04-11T08:30:00Z",
	}, 201, `"resolution-000001"`, `"inspectionId":"inspection-000001"`,
		`"concernCode":"mite_load"`, `"resolvedBy":"协调人-陆岩"`); err != nil {
		return err
	}

	// 同一关注项重复处置被拒绝。
	if _, err := c.expect("POST", resolvePath, map[string]any{
		"concerns": []string{"mite_load"}, "resolvedBy": "协调人-陆岩",
		"resolution": "再次处理",
	}, 409, `"code":"conflict"`, `不能重复处置`); err != nil {
		return err
	}

	// 汇总要在同一份视图里区分已处置与未处置，且仍把该蜂箱列为有未处置关注项。
	report, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-12T00:00:00Z", nil, 200,
		`"concernHives":["hive-01"]`,
		`"resolvedConcernHives":[]`,
		`"openConcernCount":1`,
		`"resolvedConcernCount":1`,
		`"latestInspectionId":"inspection-000001"`,
		`"code":"mite_load","status":"resolved"`,
		`"code":"temper_risk","status":"open"`,
		`"resolution":"撒粉治螨并复查"`)
	if err != nil {
		return err
	}
	// 已处置项必须排在未处置项之前（巡检派生顺序为 mite_load、temper_risk）。
	if err := report.mustOrder(`"code":"mite_load","status":"resolved"`, `"code":"temper_risk","status":"open"`); err != nil {
		return err
	}

	// 处置 temper_risk 后，蜂箱从「有未处置关注项」移到「已全部处置」。
	if _, err := c.expect("POST", resolvePath, map[string]any{
		"concerns": []string{"temper_risk"}, "resolvedBy": "协调人-陆岩",
		"resolution": "调整巢门位置",
	}, 201, `"resolution-000002"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-13T00:00:00Z", nil, 200,
		`"concernHives":[]`,
		`"resolvedConcernHives":["hive-01"]`,
		`"openConcernCount":0`,
		`"resolvedConcernCount":2`); err != nil {
		return err
	}

	// 新巡检派生出新关注项 brood_gap：旧处置不能掩盖它，必须重新变成未处置。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-18T09:00:00Z", "colonyStrength": 2, "broodPattern": "none",
		"temper": "calm", "miteCount": 1, "note": "断子复查",
	}, 201, `"inspection-000002"`, `"brood_gap"`, `"low_strength"`); err != nil {
		return err
	}
	report, err = c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-18T10:00:00Z", nil, 200,
		`"latestInspectionId":"inspection-000002"`,
		`"concernHives":["hive-01"]`,
		`"resolvedConcernHives":[]`,
		`"openConcernCount":2`,
		`"resolvedConcernCount":0`,
		`"code":"brood_gap","status":"open"`,
		`"code":"low_strength","status":"open"`)
	if err != nil {
		return err
	}
	if strings.Contains(report.text(), `"status":"resolved"`) {
		return fmt.Errorf("新巡检的关注项不应带上旧处置：%s", report.text())
	}

	// 原始巡检与派生关注项不可被处置改写。
	if _, err := c.expect("GET", "/hives/hive-01/inspections", nil, 200,
		`"id":"inspection-000001"`,
		`"concerns":["mite_load","temper_risk"]`,
		`"id":"inspection-000002"`,
		`"concerns":["brood_gap","low_strength"]`); err != nil {
		return err
	}

	// 处置历史可列出，且只追加。
	if _, err := c.expect("GET", "/hives/hive-01/concern-resolutions", nil, 200,
		`"resolution-000001"`, `"inspectionId":"inspection-000001"`, `"concernCode":"mite_load"`,
		`"resolution-000002"`, `"concernCode":"temper_risk"`); err != nil {
		return err
	}

	// 并发处置同一个新关注项，必须恰好一次成功，其余全部冲突，最终状态唯一。
	if err := assertConcurrentResolveWinner(c, "brood_gap", 20); err != nil {
		return err
	}
	report, err = c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-18T11:00:00Z", nil, 200,
		`"openConcernCount":1`,
		`"resolvedConcernCount":1`,
		`"code":"brood_gap","status":"resolved"`,
		`"code":"low_strength","status":"open"`)
	if err != nil {
		return err
	}
	// 并发结束后再次重复登记仍必须被拒绝，终态不会被翻转。
	if _, err := c.expect("POST", resolvePath, map[string]any{
		"concerns": []string{"brood_gap"}, "resolvedBy": "协调人-陆岩",
		"resolution": "补发处置",
	}, 409, `"code":"conflict"`); err != nil {
		return err
	}

	// 未知蜂箱的处置与列表返回未找到。
	if _, err := c.expect("POST", "/hives/hive-missing/concern-resolutions", map[string]any{
		"concerns": []string{"mite_load"}, "resolvedBy": "协调人-陆岩",
		"resolution": "x",
	}, 404, `"code":"not_found"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-missing/concern-resolutions", nil, 404,
		`"code":"not_found"`); err != nil {
		return err
	}
	return nil
}

// assertConcurrentResolveWinner 让多个请求并发登记同一关注项，
// 断言恰好一个 201，其余全部 409。
func assertConcurrentResolveWinner(c *client, code string, concurrency int) error {
	var wg sync.WaitGroup
	statuses := make([]int, concurrency)
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		i := i
		go func() {
			defer wg.Done()
			reply, err := c.call("POST", resolvePath, map[string]any{
				"concerns":   []string{code},
				"resolvedBy": "协调人-陆岩",
				"resolution": fmt.Sprintf("并发处置-%02d", i),
			})
			if err != nil {
				statuses[i] = -1
				return
			}
			statuses[i] = reply.status
		}()
	}
	wg.Wait()
	created, conflicted, other := 0, 0, 0
	for _, status := range statuses {
		switch status {
		case 201:
			created++
		case 409:
			conflicted++
		default:
			other++
		}
	}
	if created != 1 || conflicted != concurrency-1 || other != 0 {
		return fmt.Errorf("并发处置 %q 的最终状态不唯一：201=%d 409=%d 其他=%d（共 %d）",
			code, created, conflicted, other, concurrency)
	}
	return nil
}
