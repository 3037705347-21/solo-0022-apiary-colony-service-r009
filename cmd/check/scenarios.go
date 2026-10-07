package main

import (
	"fmt"
	"strings"
	"sync"
)

const (
	demoApiaryID = "apiary-north"
	demoHiveID   = "hive-01"
)

func seedApiary(c *client, id, name, district, keeper string) error {
	_, err := c.expect("POST", "/apiaries", map[string]any{
		"id":       id,
		"name":     name,
		"district": district,
		"keeper":   keeper,
	}, 201, `"id":"`+id+`"`)
	return err
}

func seedHive(c *client, apiaryID, hiveID, model string) error {
	_, err := c.expect("POST", "/apiaries/"+apiaryID+"/hives", map[string]any{
		"id":    hiveID,
		"model": model,
	}, 201, `"state":"colonized"`)
	return err
}

func seedDefault(c *client) error {
	if err := seedApiary(c, demoApiaryID, "北岭蜂场", "北岭片区", "周岚"); err != nil {
		return err
	}
	return seedHive(c, demoApiaryID, demoHiveID, "langstroth")
}

func checkRegisterApiary(c *client) error {
	if _, err := c.expect("POST", "/apiaries", map[string]any{
		"id": "apiary-north", "name": "北岭蜂场", "district": "北岭片区", "keeper": "周岚",
	}, 201, `"id":"apiary-north"`, `"name":"北岭蜂场"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-north", nil, 200,
		`"district":"北岭片区"`, `"keeper":"周岚"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries", map[string]any{
		"id": "apiary-north", "name": "重复蜂场", "district": "北岭片区", "keeper": "周岚",
	}, 409, `"code":"conflict"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries", map[string]any{
		"id": "apiary-south", "name": "", "district": "南坡片区", "keeper": "李澈",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	return nil
}

func checkHiveInstall(c *client) error {
	if err := seedApiary(c, demoApiaryID, "北岭蜂场", "北岭片区", "周岚"); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries/apiary-north/hives", map[string]any{
		"id": "hive-01", "model": "langstroth",
	}, 201, `"state":"colonized"`, `"apiaryId":"apiary-north"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-01", nil, 200, `"model":"langstroth"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries/apiary-north/hives", map[string]any{
		"id": "hive-01", "model": "warre",
	}, 409, `"code":"conflict"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries/apiary-missing/hives", map[string]any{
		"id": "hive-09", "model": "warre",
	}, 404, `"code":"not_found"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries/apiary-north/hives", map[string]any{
		"id": "hive-02", "model": "dadant",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	return nil
}

func checkRecordInspection(c *client) error {
	if err := seedDefault(c); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 9, "broodPattern": "patchy",
		"temper": "calm", "miteCount": 3, "note": "",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 3, "broodPattern": "patchy",
		"temper": "aggressive", "miteCount": 12, "note": "西侧巢门观察",
	}, 201, `"inspection-000001"`, `"mite_load"`, `"temper_risk"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-01/inspections", nil, 200, `"note":"西侧巢门观察"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "retired"}, 200,
		`"to":"retired"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-11T09:00:00Z", "colonyStrength": 3, "broodPattern": "solid",
		"temper": "calm", "miteCount": 1, "note": "",
	}, 409, `"code":"invalid_state"`); err != nil {
		return err
	}
	return nil
}

func checkReplaceQueen(c *client) error {
	if err := seedDefault(c); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "bred", "introducedAt": "2026-04-02T08:00:00Z", "reason": "aging",
	}, 201, `"queen-000001"`, `"markColor":"white"`, `"state":"laying"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "cells", "introducedAt": "2026-04-20T08:00:00Z", "reason": "aggression",
	}, 201, `"queen-000002"`, `"markColor":"white"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-01/queens", nil, 200,
		`"queen-000001"`, `"superseded"`, `"queen-000002"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "imported", "introducedAt": "2026-05-01T08:00:00Z", "reason": "aging",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "retired"}, 200,
		`"to":"retired"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "bred", "introducedAt": "2026-05-02T08:00:00Z", "reason": "loss",
	}, 409, `"code":"invalid_state"`); err != nil {
		return err
	}
	return nil
}

func checkAdvanceHiveState(c *client) error {
	if err := seedDefault(c); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "production"}, 200,
		`"from":"colonized"`, `"to":"production"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "colonized"}, 409,
		`"code":"invalid_state"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "production"}, 409,
		`"code":"invalid_state"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "retired"}, 200,
		`"to":"retired"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "production"}, 409,
		`"code":"invalid_state"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-missing/state", map[string]any{"target": "production"}, 404,
		`"code":"not_found"`); err != nil {
		return err
	}
	return nil
}

func checkDisposeConcern(c *client) error {
	if err := seedDefault(c); err != nil {
		return err
	}
	// 尚无巡检的蜂箱没有可处置的关注项。
	if _, err := c.expect("POST", "/hives/hive-01/concern-dispositions", map[string]any{
		"concern": "mite_load", "disposedBy": "陈启",
	}, 409, `"code":"invalid_state"`); err != nil {
		return err
	}
	// 巡检派生 mite_load 与 temper_risk 两个关注项。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 3, "broodPattern": "patchy",
		"temper": "aggressive", "miteCount": 12, "note": "",
	}, 201, `"inspection-000001"`, `"mite_load"`, `"temper_risk"`); err != nil {
		return err
	}
	// 处置前：全部关注项未处置。
	if _, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-12T00:00:00Z", nil, 200,
		`"concernHives":["hive-01"]`, `"resolvedHives":[]`,
		`"openConcerns":["mite_load","temper_risk"]`, `"resolvedConcerns":[]`); err != nil {
		return err
	}
	// 登记对 mite_load 的处置，绑定到最近一次巡检。
	if _, err := c.expect("POST", "/hives/hive-01/concern-dispositions", map[string]any{
		"concern": "mite_load", "disposedBy": "陈启", "note": "已投放杀螨条",
	}, 201, `"disposition-000001"`, `"inspectionId":"inspection-000001"`, `"concern":"mite_load"`); err != nil {
		return err
	}
	// 部分处置：蜂箱仍有未处置关注项。
	if _, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-12T00:00:00Z", nil, 200,
		`"concernHives":["hive-01"]`, `"resolvedHives":[]`,
		`"openConcerns":["temper_risk"]`, `"resolvedConcerns":["mite_load"]`); err != nil {
		return err
	}
	// 重复处置同一关注项返回冲突。
	if _, err := c.expect("POST", "/hives/hive-01/concern-dispositions", map[string]any{
		"concern": "mite_load", "disposedBy": "陈启",
	}, 409, `"code":"conflict"`); err != nil {
		return err
	}
	// 最近一次巡检未派生的关注项不能处置。
	if _, err := c.expect("POST", "/hives/hive-01/concern-dispositions", map[string]any{
		"concern": "brood_gap", "disposedBy": "陈启",
	}, 409, `"code":"invalid_state"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/concern-dispositions", map[string]any{
		"concern": "temper_risk",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-missing/concern-dispositions", map[string]any{
		"concern": "mite_load", "disposedBy": "陈启",
	}, 404, `"code":"not_found"`); err != nil {
		return err
	}
	// 处置 temper_risk 后，该巡检的关注项全部处置完毕。
	if _, err := c.expect("POST", "/hives/hive-01/concern-dispositions", map[string]any{
		"concern": "temper_risk", "disposedBy": "陈启", "note": "已转移隔离",
	}, 201, `"disposition-000002"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-12T00:00:00Z", nil, 200,
		`"concernHives":[]`, `"resolvedHives":["hive-01"]`, `"openConcerns":[]`); err != nil {
		return err
	}
	// 新巡检派生新的 mite_load，旧处置不能掩盖它。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-11T09:00:00Z", "colonyStrength": 4, "broodPattern": "solid",
		"temper": "calm", "miteCount": 10, "note": "",
	}, 201, `"inspection-000002"`, `"mite_load"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-12T00:00:00Z", nil, 200,
		`"concernHives":["hive-01"]`, `"resolvedHives":[]`, `"openConcerns":["mite_load"]`); err != nil {
		return err
	}
	// 原始巡检与派生关注项保持原样，处置不改写它们。
	inspections, err := c.expect("GET", "/hives/hive-01/inspections", nil, 200,
		`"concerns":["mite_load","temper_risk"]`, `"concerns":["mite_load"]`)
	if err != nil {
		return err
	}
	if strings.Contains(inspections.text(), "disposition") {
		return fmt.Errorf("巡检记录不应包含处置内容：%s", inspections.text())
	}
	// 新巡检派生的同一关注项可以再次处置，绑定到新巡检。
	if _, err := c.expect("POST", "/hives/hive-01/concern-dispositions", map[string]any{
		"concern": "mite_load", "disposedBy": "陈启",
	}, 201, `"disposition-000003"`, `"inspectionId":"inspection-000002"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-01/concern-dispositions", nil, 200,
		`"disposition-000001"`, `"disposition-000002"`, `"disposition-000003"`); err != nil {
		return err
	}
	// 与巡检提交交错：新巡检后并发重复处置，最终状态必须唯一。
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-12T09:00:00Z", "colonyStrength": 4, "broodPattern": "solid",
		"temper": "calm", "miteCount": 15, "note": "",
	}, 201, `"inspection-000003"`); err != nil {
		return err
	}
	const attempts = 8
	results := make(chan int, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reply, err := c.call("POST", "/hives/hive-01/concern-dispositions", map[string]any{
				"concern": "mite_load", "disposedBy": "陈启",
			})
			if err != nil {
				results <- 0
				return
			}
			results <- reply.status
		}()
	}
	wg.Wait()
	close(results)
	created, conflicts := 0, 0
	for status := range results {
		switch status {
		case 201:
			created++
		case 409:
			conflicts++
		}
	}
	if created != 1 || conflicts != attempts-1 {
		return fmt.Errorf("并发重复处置期望 1 次成功、%d 次冲突，实际成功 %d 次、冲突 %d 次", attempts-1, created, conflicts)
	}
	list, err := c.expect("GET", "/hives/hive-01/concern-dispositions", nil, 200)
	if err != nil {
		return err
	}
	if count := strings.Count(list.text(), `"inspectionId":"inspection-000003"`); count != 1 {
		return fmt.Errorf("巡检 inspection-000003 的处置记录期望恰好 1 条，实际 %d 条：%s", count, list.text())
	}
	// 并发处置落定的关注项在汇总中显示为已处置。
	if _, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-13T00:00:00Z", nil, 200,
		`"concernHives":[]`, `"resolvedHives":["hive-01"]`, `"resolvedConcerns":["mite_load"]`); err != nil {
		return err
	}
	return nil
}

func checkReadApiaryStatus(c *client) error {
	if err := seedApiary(c, demoApiaryID, "北岭蜂场", "北岭片区", "周岚"); err != nil {
		return err
	}
	for _, hiveID := range []string{"hive-01", "hive-02", "hive-03"} {
		if err := seedHive(c, demoApiaryID, hiveID, "langstroth"); err != nil {
			return err
		}
	}
	if _, err := c.expect("POST", "/hives/hive-02/state", map[string]any{"target": "production"}, 200,
		`"to":"production"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 4, "broodPattern": "solid",
		"temper": "calm", "miteCount": 12, "note": "",
	}, 201, `"mite_load"`); err != nil {
		return err
	}
	report, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-12T00:00:00Z", nil, 200,
		`"hiveCount":3`, `"colonized":2`, `"production":1`, `"layingQueens":0`)
	if err != nil {
		return err
	}
	for _, fragment := range []string{
		`"needsInspection":["hive-02","hive-03"]`,
		`"concernHives":["hive-01"]`,
		`"resolvedHives":[]`,
		`"openConcerns":["mite_load"]`,
		`"resolvedConcerns":[]`,
	} {
		if !strings.Contains(report.text(), fragment) {
			return fmt.Errorf("状态汇总缺少 %q：%s", fragment, report.text())
		}
	}
	if err := report.mustOrder(`"hiveId":"hive-01"`, `"hiveId":"hive-02"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-missing/status", nil, 404, `"code":"not_found"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=not-a-time", nil, 400,
		`"code":"invalid_request"`); err != nil {
		return err
	}
	return nil
}
