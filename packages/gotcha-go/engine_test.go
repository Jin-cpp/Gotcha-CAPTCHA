package gotchago

import (
	"fmt"
	"testing"
)

// TestEngineVerificationAndScare 测试验证码引擎的答案判定、失败计数累加与触发整蛊阈值逻辑。
func TestEngineVerificationAndScare(t *testing.T) {
	// 初始化引擎：设定连续失败 2 次即触发整蛊
	engine := NewEngine(2)

	// 注册测试题目：t1 为目标正确答案，t2 为诱饵干扰项
	c1 := Challenge{
		ID:          "test-1",
		Title:       "选择所有目标物品",
		Target:      "目标物品",
		Instruction: "选择完成后点击验证",
		Tiles: []Tile{
			{ID: "t1", ImageURL: "/img1.png", IsTarget: true},
			{ID: "t2", ImageURL: "/img2.png", IsTarget: false},
		},
	}
	engine.AddChallenge(c1)

	// 生成初始题目会话
	challenge, err := engine.GenerateChallenge("")
	if err != nil {
		t.Fatalf("生成题目失败: %v", err)
	}

	sessionID := challenge.SessionID
	if sessionID == "" {
		t.Fatal("期望获得非空的 sessionID，但实际为空")
	}

	// 模拟第 1 次错误作答（误选了诱饵项 t2）
	res1, err := engine.Verify(VerifyRequest{
		ChallengeID: "test-1",
		SessionID:   sessionID,
		SelectedIDs: []string{"t2"},
	})
	if err != nil {
		t.Fatalf("第 1 次验证执行失败: %v", err)
	}
	// 断言：验证不通过、未触发惊吓、失败计数为 1
	if res1.Success || res1.TriggerScare || res1.FailCount != 1 {
		t.Fatalf("第 1 次验证结果不符合预期: %+v", res1)
	}

	// 模拟第 2 次错误作答（达到设定的最大阈值 2 次）
	res2, err := engine.Verify(VerifyRequest{
		ChallengeID: "test-1",
		SessionID:   sessionID,
		SelectedIDs: []string{"t2"},
	})
	if err != nil {
		t.Fatalf("第 2 次验证执行失败: %v", err)
	}
	// 断言：应当触发整蛊标记 TriggerScare
	if !res2.TriggerScare {
		t.Fatalf("达到 2 次失败但未触发 TriggerScare，实际结果: %+v", res2)
	}
}

// TestRandomPoolMode 验证随机候选池模式：
// 1. 验证矩阵尺寸正确（3x3 = 9 格）；
// 2. 循环生成 50 次，断言每次抽选均保证至少包含 1 张正确目标图片。
func TestRandomPoolMode(t *testing.T) {
	engine := NewEngine(3)

	cPool := Challenge{
		ID:          "pool-test",
		Mode:        ModeRandomPool,
		Rows:        3,
		Columns:     3,
		Title:       "选择所有正确目标",
		Target:      "正确目标",
		Instruction: "至少出现一张正确图片",
		TargetPool: []Tile{
			{ID: "target-1", ImageURL: "/target1.svg", IsTarget: true},
			{ID: "target-2", ImageURL: "/target2.svg", IsTarget: true},
			{ID: "target-3", ImageURL: "/target3.svg", IsTarget: true},
		},
		DistractorPool: []Tile{
			{ID: "decoy-1", ImageURL: "/decoy1.svg", IsTarget: false},
			{ID: "decoy-2", ImageURL: "/decoy2.svg", IsTarget: false},
			{ID: "decoy-3", ImageURL: "/decoy3.svg", IsTarget: false},
			{ID: "decoy-4", ImageURL: "/decoy4.svg", IsTarget: false},
		},
	}
	engine.AddChallenge(cPool)

	// 压力验证 50 次生成
	for i := 0; i < 50; i++ {
		clientChal, err := engine.GenerateChallenge("")
		if err != nil {
			t.Fatalf("第 %d 次生成题目失败: %v", i, err)
		}

		if clientChal.Rows != 3 || clientChal.Columns != 3 {
			t.Fatalf("矩阵行数列数错误: 期望 3x3, 得到 %dx%d", clientChal.Rows, clientChal.Columns)
		}

		if len(clientChal.Tiles) != 9 {
			t.Fatalf("方块总数错误: 期望 9, 得到 %d", len(clientChal.Tiles))
		}

		// 检查服务端 session 中记录的正确答案数量
		session := engine.sessions[clientChal.SessionID]
		targetCount := len(session.CorrectTileIDs)

		// 核心断言：必须保证至少包含 1 张正确图片
		if targetCount < 1 {
			t.Fatalf("第 %d 次生成未包含任何正确目标图片！targetCount=%d", i, targetCount)
		}

		// 核心断言：不能全部都是目标（必须保留干扰项）
		if targetCount > 8 {
			t.Fatalf("第 %d 次生成全部为正确目标图片，缺乏干扰性！targetCount=%d", i, targetCount)
		}
	}
}

// TestFixedLayoutMode 验证全位置指定固定模式与不同长宽矩阵（4x4 与 2x3）：
// 1. 验证方块位置严格保留预设顺序不被打乱；
// 2. 验证非对称长宽矩阵正常工作。
func TestFixedLayoutMode(t *testing.T) {
	engine := NewEngine(3)

	// 1. 测试 4x4 矩阵（16 格）
	fixedTiles4x4 := make([]Tile, 16)
	for i := 0; i < 16; i++ {
		// 偶数索引设为正确目标，奇数索引设为干扰项
		fixedTiles4x4[i] = Tile{
			ID:       string(rune('A' + i)),
			ImageURL: "/img.png",
			IsTarget: (i%2 == 0),
		}
	}

	c4x4 := Challenge{
		ID:         "fixed-4x4",
		Mode:       ModeFixedLayout,
		Rows:       4,
		Columns:    4,
		FixedTiles: fixedTiles4x4,
	}
	engine.AddChallenge(c4x4)

	chal4x4, err := engine.GenerateChallenge("session-fixed-4x4")
	if err != nil {
		t.Fatalf("生成 4x4 题目失败: %v", err)
	}

	if chal4x4.Rows != 4 || chal4x4.Columns != 4 || len(chal4x4.Tiles) != 16 {
		t.Fatalf("4x4 规格不匹配: rows=%d, cols=%d, tiles=%d", chal4x4.Rows, chal4x4.Columns, len(chal4x4.Tiles))
	}

	// 验证位置完全一致且未乱序
	for i := 0; i < 16; i++ {
		expectedID := string(rune('A' + i))
		if chal4x4.Tiles[i].ID != expectedID {
			t.Fatalf("位置 %d 被打乱: 期望 %s, 得到 %s", i, expectedID, chal4x4.Tiles[i].ID)
		}
	}

	// 2. 测试 2x3 矩阵（6 格）
	fixedTiles2x3 := []Tile{
		{ID: "r0c0", ImageURL: "/00.png", IsTarget: true},
		{ID: "r0c1", ImageURL: "/01.png", IsTarget: false},
		{ID: "r0c2", ImageURL: "/02.png", IsTarget: true},
		{ID: "r1c0", ImageURL: "/10.png", IsTarget: false},
		{ID: "r1c1", ImageURL: "/11.png", IsTarget: false},
		{ID: "r1c2", ImageURL: "/12.png", IsTarget: true},
	}

	c2x3 := Challenge{
		ID:         "fixed-2x3",
		Mode:       ModeFixedLayout,
		Rows:       2,
		Columns:    3,
		FixedTiles: fixedTiles2x3,
	}
	engine2x3 := NewEngine(3)
	engine2x3.AddChallenge(c2x3)

	chal2x3, err := engine2x3.GenerateChallenge("session-fixed-2x3")
	if err != nil {
		t.Fatalf("生成 2x3 题目失败: %v", err)
	}

	if chal2x3.Rows != 2 || chal2x3.Columns != 3 || len(chal2x3.Tiles) != 6 {
		t.Fatalf("2x3 规格不匹配: rows=%d, cols=%d, tiles=%d", chal2x3.Rows, chal2x3.Columns, len(chal2x3.Tiles))
	}

	// 提交正确答案进行校验
	correctSelections := []string{"r0c0", "r0c2", "r1c2"}
	vRes, err := engine2x3.Verify(VerifyRequest{
		ChallengeID: "fixed-2x3",
		SessionID:   "session-fixed-2x3",
		SelectedIDs: correctSelections,
	})
	if err != nil {
		t.Fatalf("2x3 校验执行异常: %v", err)
	}
	if !vRes.Success {
		t.Fatalf("2x3 提交全正确答案却未通过: %+v", vRes)
	}
}

// TestNonRepeatingQueueMechanism 验证无重复题目滑动队列机制：
// 1. 题库总数 N=5，设置至少不重复次数 M=3，验证队列长度始终为 min{3, 5}=3；
// 2. 验证任意一道被抽中的题目，在接下来的 3 次抽取中绝对不会重复出现。
func TestNonRepeatingQueueMechanism(t *testing.T) {
	// 创建 5 道不同的题目
	engine := NewEngine(3, 3)
	for i := 1; i <= 5; i++ {
		c := Challenge{
			ID:      fmt.Sprintf("chal-%d", i),
			Mode:    ModeRandomPool,
			Rows:    3,
			Columns: 3,
			TargetPool: []Tile{
				{ID: fmt.Sprintf("t%d_1", i), ImageURL: "/target.png", IsTarget: true},
			},
			DistractorPool: []Tile{
				{ID: fmt.Sprintf("d%d_1", i), ImageURL: "/dist1.png", IsTarget: false},
				{ID: fmt.Sprintf("d%d_2", i), ImageURL: "/dist2.png", IsTarget: false},
			},
		}
		engine.AddChallenge(c)
	}

	drawHistory := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		chal, err := engine.GenerateChallenge("")
		if err != nil {
			t.Fatalf("第 %d 次抽取题目失败: %v", i+1, err)
		}

		// 检查当前内部队列长度是否恒为 min{3, 5} = 3
		queueSnapshot := engine.GetChallengeQueue()
		if len(queueSnapshot) != 3 {
			t.Fatalf("第 %d 次抽取后队列长度异常: 期望 3, 实际为 %d (%v)", i+1, len(queueSnapshot), queueSnapshot)
		}

		// 检查队列中是否存在重复元素
		seenInQueue := make(map[string]bool)
		for _, qID := range queueSnapshot {
			if seenInQueue[qID] {
				t.Fatalf("第 %d 次抽取后队列中发现重复题目: %s in %v", i+1, qID, queueSnapshot)
			}
			seenInQueue[qID] = true
		}

		drawHistory = append(drawHistory, chal.ID)
	}

	// 核心断言：验证在任何连续 3 次的抽取范围内，不存在相同的题目
	// 即：对于第 idx 次抽取的题目，在 idx+1 与 idx+2 次抽取中绝不可重复出现
	for idx := 0; idx < len(drawHistory)-2; idx++ {
		cur := drawHistory[idx]
		for offset := 1; offset < 3; offset++ {
			next := drawHistory[idx+offset]
			if cur == next {
				t.Fatalf("违反防重复规则: 题目 %s 在距离为 %d 的抽取中再次出现 (位置 %d 与 %d), 历史: %v",
					cur, offset, idx, idx+offset, drawHistory[idx:idx+4])
			}
		}
	}
}

// TestNonRepeatingQueueExceedsTotal 验证当“至少不重复次数”大于题库总数时：
// 长度自动截断为 min{M, N} = N，题目呈全量轮转，每连续 N 次必包含全量题目无重复。
func TestNonRepeatingQueueExceedsTotal(t *testing.T) {
	// N = 3 道题，M = 10 (M > N)
	engine := NewEngine(3, 10)
	for i := 1; i <= 3; i++ {
		c := Challenge{
			ID:      fmt.Sprintf("item-%d", i),
			Mode:    ModeFixedLayout,
			Rows:    2,
			Columns: 2,
			FixedTiles: []Tile{
				{ID: "t1", ImageURL: "/1.png", IsTarget: true},
				{ID: "t2", ImageURL: "/2.png", IsTarget: false},
				{ID: "t3", ImageURL: "/3.png", IsTarget: false},
				{ID: "t4", ImageURL: "/4.png", IsTarget: false},
			},
		}
		engine.AddChallenge(c)
	}

	draws := make([]string, 30)
	for i := 0; i < 30; i++ {
		chal, err := engine.GenerateChallenge("")
		if err != nil {
			t.Fatalf("抽取失败: %v", err)
		}
		draws[i] = chal.ID
	}

	// 每 3 次连续抽取必须包含全量 3 道不同题目
	for i := 0; i < len(draws)-2; i += 3 {
		window := draws[i : i+3]
		set := make(map[string]bool)
		for _, id := range window {
			set[id] = true
		}
		if len(set) != 3 {
			t.Fatalf("窗口 %d~%d 未包含全部 3 道不同题目: %v", i, i+2, window)
		}
	}
}

// TestNonRepeatingQueueSingleChallenge 验证题库只有 1 道题目时的边界平稳降级
func TestNonRepeatingQueueSingleChallenge(t *testing.T) {
	engine := NewEngine(3, 5)
	engine.AddChallenge(Challenge{
		ID:      "only-one",
		Mode:    ModeFixedLayout,
		Rows:    2,
		Columns: 2,
		FixedTiles: []Tile{
			{ID: "t1", ImageURL: "/1.png", IsTarget: true},
			{ID: "t2", ImageURL: "/2.png", IsTarget: false},
			{ID: "t3", ImageURL: "/3.png", IsTarget: false},
			{ID: "t4", ImageURL: "/4.png", IsTarget: false},
		},
	})

	for i := 0; i < 5; i++ {
		chal, err := engine.GenerateChallenge("")
		if err != nil {
			t.Fatalf("单题目抽取失败: %v", err)
		}
		if chal.ID != "only-one" {
			t.Fatalf("期望 only-one, 得到 %s", chal.ID)
		}
	}
}
