package gotchago

import (
	"errors"
	"fmt"
)

// Verify 对受试者在前端提交的方块勾选答案进行比对校验。
// 若答案全部正确，重置失败计数并返回通过；
// 若答案错误，累计失败次数，并在达到 MaxFailsToScare 阈值时触发 TriggerScare 整蛊标记。
func (e *Engine) Verify(req VerifyRequest) (*VerifyResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. 基础校验：SessionID 不能为空
	if req.SessionID == "" {
		return nil, errors.New("校验请求缺少 sessionId 会话标识符")
	}

	// 2. 获取当前会话状态，防止跨会话串号或会话过期
	session, exists := e.sessions[req.SessionID]
	if !exists {
		return nil, errors.New("会话不存在或已过期，请刷新重试")
	}

	// 3. 校验题目 ID 是否一致，防止受试者重复提交上一题作答
	if session.ActiveChallengeID != req.ChallengeID {
		return nil, fmt.Errorf("题目 ID 不匹配（当前活跃题目: %s, 接收题目: %s）", session.ActiveChallengeID, req.ChallengeID)
	}

	// 4. 将用户提交的选中 ID 转为哈希表以便高效 O(1) 判定
	selectedMap := make(map[string]bool)
	for _, id := range req.SelectedIDs {
		selectedMap[id] = true
	}

	isCorrect := true

	// 5. 检查是否漏选了任何一个目标正确方块
	for targetID := range session.CorrectTileIDs {
		if !selectedMap[targetID] {
			isCorrect = false
			break
		}
	}

	// 6. 检查是否误选了任何非目标方块
	if isCorrect {
		for _, id := range req.SelectedIDs {
			if !session.CorrectTileIDs[id] {
				isCorrect = false
				break
			}
		}
	}

	// 7. 答案完全正确的分支
	if isCorrect {
		session.FailCount = 0 // 验证成功，重置失败计数
		return &VerifyResult{
			Success:      true,
			Message:      "验证通过！",
			FailCount:    0,
			TriggerScare: false,
		}, nil
	}

	// 8. 答案错误分支：累加失败计数并判定是否达到整蛊阈值
	session.FailCount++
	triggerScare := session.FailCount >= e.MaxFailsToScare

	result := &VerifyResult{
		Success:      false,
		FailCount:    session.FailCount,
		TriggerScare: triggerScare,
	}

	if triggerScare {
		result.Message = fmt.Sprintf("已达到连续失败上限 (%d 次)，触发整蛊惩罚！", e.MaxFailsToScare)
		// 触发整蛊后，重置失败计数以便后续重新开始
		session.FailCount = 0
	} else {
		result.Message = fmt.Sprintf("未能通过验证，已失败 %d/%d 次。请再试一次！", session.FailCount, e.MaxFailsToScare)
	}

	// 9. 临时解锁以调用 GenerateChallenge 为该会话生成下一道题目
	e.mu.Unlock()
	nextChallenge, err := e.GenerateChallenge(req.SessionID)
	e.mu.Lock()

	if err == nil {
		result.NextChallenge = nextChallenge
	}

	return result, nil
}
