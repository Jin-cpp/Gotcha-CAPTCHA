package gotchago

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
)

// SessionState 用于在服务端维护单个受试者会话的作答状态与失败历史记录。
type SessionState struct {
	SessionID         string          // 会话唯一标识符
	ActiveChallengeID string          // 当前正在作答的题目编号
	CorrectTileIDs    map[string]bool // 当前题目正确答案的方块 ID 集合（哈希表实现 O(1) 查找）
	FailCount         int             // 当前会话已连续失败的累计次数
}

// Engine 整蛊验证码引擎核心，负责管理题库、处理会话状态以及图片随机洗牌。
type Engine struct {
	mu               sync.RWMutex             // 读写锁，保障多并发请求下的线程安全
	challenges       map[string]Challenge     // 已加载的题目字典，Key 为题目 ID
	challengeIDs     []string                 // 题目 ID 列表，便于通过索引高效随机抽选
	challengeQueue   []string                 // 无重复题目的出题滑动队列，长度为 min{MinNoRepeatCount, len(challengeIDs)}
	sessions         map[string]*SessionState // 活跃会话状态表，Key 为 SessionID
	MaxFailsToScare  int                      // 连续失败触发整蛊（Jumpscare）的阈值（默认 3 次）
	MinNoRepeatCount int                      // 至少不会抽到重复题的次数（防重复抽取窗口长度）
}

// NewEngine 创建并初始化一个新的整蛊验证码引擎实例。
// 参数 maxFailsToScare 指定连续失败多少次后触发整蛊（默认最少为 3 次）。
// 可选参数 minNoRepeatCount 指定出题时至少连续多少次不会抽到重复题目。
func NewEngine(maxFailsToScare int, minNoRepeatCount ...int) *Engine {
	if maxFailsToScare <= 0 {
		maxFailsToScare = 3
	}
	minNoRepeat := 0
	if len(minNoRepeatCount) > 0 {
		minNoRepeat = minNoRepeatCount[0]
	}
	return &Engine{
		challenges:       make(map[string]Challenge),
		challengeIDs:     make([]string, 0),
		challengeQueue:   make([]string, 0),
		sessions:         make(map[string]*SessionState),
		MaxFailsToScare:  maxFailsToScare,
		MinNoRepeatCount: minNoRepeat,
	}
}

// LoadChallengesFromDir 扫描并加载指定目录下的所有 *.json 题目配置文件。
// 该方法会自动读取文件内容、反序列化为 Challenge 结构体并注入引擎。
func (e *Engine) LoadChallengesFromDir(dir string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("读取题库目录失败: %w", err)
	}

	for _, entry := range entries {
		// 忽略子目录及非 JSON 格式文件
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("读取题库文件 %s 失败: %w", filePath, err)
		}

		var challenge Challenge
		if err := json.Unmarshal(data, &challenge); err != nil {
			return fmt.Errorf("解析题库文件 %s 的 JSON 格式失败: %w", filePath, err)
		}

		// 若未配置 ID，则默认使用文件名作为题目编号
		if challenge.ID == "" {
			challenge.ID = entry.Name()
		}

		e.challenges[challenge.ID] = challenge
		e.challengeIDs = append(e.challengeIDs, challenge.ID)
	}

	if len(e.challenges) == 0 {
		return errors.New("指定目录下未发现任何有效的题目 JSON 配置文件")
	}

	// 题库更新后，重置出题队列以便重新洗牌初始化
	e.challengeQueue = nil

	return nil
}

// AddChallenge 向引擎中动态注册一道纯内存挑战题目。
func (e *Engine) AddChallenge(c Challenge) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.challenges[c.ID] = c
	e.challengeIDs = append(e.challengeIDs, c.ID)
	// 动态添加新题目后重置队列，使新题目纳入候选抽选题池
	e.challengeQueue = nil
}

// SetMinNoRepeatCount 动态设置“至少不会抽到重复题的次数”参数并重置出题队列。
func (e *Engine) SetMinNoRepeatCount(count int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.MinNoRepeatCount = count
	e.challengeQueue = nil
}

// GetChallengeQueue 返回当前滑动出题队列的只读快照副本（便于测试断言与状态排查）。
func (e *Engine) GetChallengeQueue() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	res := make([]string, len(e.challengeQueue))
	copy(res, e.challengeQueue)
	return res
}

// initChallengeQueue 依据题库全量题目列表，随机抽取 targetLen 个互不重复的题目初始化出题队列。
// 注意：该方法必须在持有 e.mu 互斥锁的上下文中被调用。
func (e *Engine) initChallengeQueue(targetLen int) {
	if len(e.challengeIDs) == 0 || targetLen <= 0 {
		e.challengeQueue = nil
		return
	}
	if targetLen > len(e.challengeIDs) {
		targetLen = len(e.challengeIDs)
	}

	// 复制题目 ID 列表并使用 crypto/rand 执行 Fisher-Yates 洗牌
	shuffled := make([]string, len(e.challengeIDs))
	copy(shuffled, e.challengeIDs)
	for i := len(shuffled) - 1; i > 0; i-- {
		nBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			break
		}
		j := nBig.Int64()
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	e.challengeQueue = shuffled[:targetLen]
}

// drawNextChallengeID 从无重复题目队列中抽取下一道题（出队首题并随机入队新候选）。
// 算法保证：维护一个无重复题目的长度为 min{至少不会抽到重复题的次数, 题目总数} 的队列，
// 每次取队头的题目，将其出队，并随机取一个不在当前队列中的题目入队。
// 注意：该方法必须在持有 e.mu 互斥锁的上下文中被调用。
func (e *Engine) drawNextChallengeID() (string, error) {
	if len(e.challengeIDs) == 0 {
		return "", errors.New("引擎题库为空，无法生成题目")
	}

	// 若未配置至少不重复次数，或总题库仅有 1 题，平滑回退至纯随机抽取
	if e.MinNoRepeatCount <= 0 || len(e.challengeIDs) == 1 {
		randIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(e.challengeIDs))))
		if err != nil {
			return "", fmt.Errorf("生成随机索引失败: %w", err)
		}
		return e.challengeIDs[randIndex.Int64()], nil
	}

	// 计算无重复队列的目标长度 L = min{至少不会抽到重复题的次数, 题目总数}
	targetLen := e.MinNoRepeatCount
	if targetLen > len(e.challengeIDs) {
		targetLen = len(e.challengeIDs)
	}

	// 校验当前队列有效性（长度匹配、包含的题目均有效且互不重复）
	queueValid := (len(e.challengeQueue) == targetLen)
	if queueValid {
		seen := make(map[string]bool, targetLen)
		for _, id := range e.challengeQueue {
			if _, exists := e.challenges[id]; !exists || seen[id] {
				queueValid = false
				break
			}
			seen[id] = true
		}
	}
	if !queueValid {
		e.initChallengeQueue(targetLen)
	}

	// 1. 取队头题目并将其出队
	head := e.challengeQueue[0]
	e.challengeQueue = e.challengeQueue[1:]

	// 2. 统计当前仍保留在队列中的题目 ID
	inQueue := make(map[string]bool, len(e.challengeQueue))
	for _, id := range e.challengeQueue {
		inQueue[id] = true
	}

	// 3. 找出所有不在当前队列中的候选题目
	var availableCandidates []string
	for _, id := range e.challengeIDs {
		if !inQueue[id] {
			availableCandidates = append(availableCandidates, id)
		}
	}

	// 4. 随机选取一个不在队列中的题目入队
	if len(availableCandidates) > 0 {
		nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(availableCandidates))))
		if err != nil {
			return "", fmt.Errorf("选取下一道入队题目失败: %w", err)
		}
		chosen := availableCandidates[nBig.Int64()]
		e.challengeQueue = append(e.challengeQueue, chosen)
	}

	// 5. 返回本次抽取的队头题目 ID
	return head, nil
}

// GenerateChallenge 为指定会话生成一道题目。
// 支持根据题目配置生成任意行数与列数（Rows * Columns）的图片矩阵，并支持两种出题模式：
// 1. ModeRandomPool（随机抽选模式）：从候选池中抽取图片，算法保证至少包含 1 张正确图片，并全局洗牌乱序；
// 2. ModeFixedLayout（固定布局模式）：严格按照预设位置坐标配置每个单元格，不打乱位置。
// 若传入的 sessionID 为空，则自动生成全新的加密随机令牌；返回脱敏后的 ClientChallenge 供前端渲染。
func (e *Engine) GenerateChallenge(sessionID string) (*ClientChallenge, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.challengeIDs) == 0 {
		return nil, errors.New("引擎题库为空，无法生成题目")
	}

	// 1. 依据无重复出题滑动队列机制选取一道题目模板
	challengeID, err := e.drawNextChallengeID()
	if err != nil {
		return nil, err
	}
	original := e.challenges[challengeID]

	// 2. 解析矩阵长宽尺寸（若未配置或非法则默认采用经典 3x3）
	rows := original.Rows
	if rows <= 0 {
		rows = 3
	}
	cols := original.Columns
	if cols <= 0 {
		cols = 3
	}
	totalSlots := rows * cols

	// 3. 判定出题模式（若未显式指定，则根据字段配置自动识别）
	mode := original.Mode
	if mode == "" {
		if len(original.FixedTiles) > 0 {
			mode = ModeFixedLayout
		} else if len(original.TargetPool) > 0 || len(original.DistractorPool) > 0 || len(original.Pool) > 0 {
			mode = ModeRandomPool
		} else if len(original.Tiles) == totalSlots {
			mode = ModeFixedLayout
		} else {
			mode = ModeRandomPool
		}
	}

	// 选出的服务端方块切片（长度为 totalSlots）
	var chosenTiles []Tile
	correctIDs := make(map[string]bool)

	switch mode {
	case ModeFixedLayout:
		// ==================== 模式二：全位置指定固定模式 ====================
		fixedSource := original.FixedTiles
		if len(fixedSource) == 0 {
			fixedSource = original.Tiles
		}
		if len(fixedSource) != totalSlots {
			return nil, fmt.Errorf("fixed_layout 模式要求方块总数与矩阵规格一致: 期望 %d (行 %d * 列 %d)，实际配置了 %d", totalSlots, rows, cols, len(fixedSource))
		}

		chosenTiles = make([]Tile, totalSlots)
		copy(chosenTiles, fixedSource)

		// 固定模式保持预设位置，不执行乱序洗牌
		for idx, t := range chosenTiles {
			cellID := t.ID
			if cellID == "" {
				cellID = fmt.Sprintf("cell_%d", idx)
				chosenTiles[idx].ID = cellID
			}
			if t.IsTarget {
				correctIDs[cellID] = true
			}
		}

	case ModeRandomPool:
		fallthrough
	default:
		// ==================== 模式一：候选池随机抽取模式 ====================
		mode = ModeRandomPool

		// 整理目标图片候选池与干扰图片候选池
		var targetCandidates []Tile
		var distractorCandidates []Tile

		// 汇总显式配置的 TargetPool 与 DistractorPool
		targetCandidates = append(targetCandidates, original.TargetPool...)
		distractorCandidates = append(distractorCandidates, original.DistractorPool...)

		// 汇总统一 Pool 集合
		for _, t := range original.Pool {
			if t.IsTarget {
				targetCandidates = append(targetCandidates, t)
			} else {
				distractorCandidates = append(distractorCandidates, t)
			}
		}

		// 汇总兼容字段 Tiles
		for _, t := range original.Tiles {
			if t.IsTarget {
				targetCandidates = append(targetCandidates, t)
			} else {
				distractorCandidates = append(distractorCandidates, t)
			}
		}

		// 校验候选池中是否至少存在 1 张正确图片
		if len(targetCandidates) == 0 {
			return nil, fmt.Errorf("题目 %s 的 random_pool 模式缺少正确目标图片候选", challengeID)
		}

		// 确定抽取的正确目标数量 K（保证至少 1 张正确）
		minK := 1
		if original.MinTargets > 0 {
			minK = original.MinTargets
			if minK > totalSlots {
				minK = totalSlots
			}
		}

		maxK := totalSlots
		if original.MaxTargets > 0 && original.MaxTargets >= minK {
			maxK = original.MaxTargets
			if maxK > totalSlots {
				maxK = totalSlots
			}
		} else if len(distractorCandidates) > 0 && totalSlots > 1 {
			// 若存在干扰项且槽位大于1，则最大正确数上限为 totalSlots - 1，确保至少保留 1 个干扰位以具有挑战性
			maxK = totalSlots - 1
			if maxK < minK {
				maxK = minK
			}
		}

		// 随机确定本次题目生成的正确目标数量 targetCount (minK <= targetCount <= maxK)
		targetCount := minK
		if maxK > minK {
			deltaBig, _ := rand.Int(rand.Reader, big.NewInt(int64(maxK-minK+1)))
			targetCount = minK + int(deltaBig.Int64())
		}
		distractorCount := totalSlots - targetCount

		// 从 targetCandidates 抽选 targetCount 张
		sampledTargets, err := sampleTiles(targetCandidates, targetCount)
		if err != nil {
			return nil, fmt.Errorf("抽选目标图片失败: %w", err)
		}

		// 从 distractorCandidates 抽选 distractorCount 张
		var sampledDistractors []Tile
		if distractorCount > 0 {
			if len(distractorCandidates) > 0 {
				sampledDistractors, err = sampleTiles(distractorCandidates, distractorCount)
				if err != nil {
					return nil, fmt.Errorf("抽选干扰图片失败: %w", err)
				}
			} else {
				// 若无干扰项，则全部使用目标项填补
				additionalTargets, _ := sampleTiles(targetCandidates, distractorCount)
				sampledDistractors = additionalTargets
			}
		}

		// 合并抽取的方块
		rawTiles := append(sampledTargets, sampledDistractors...)

		// 对抽选的方块切片执行全局 Fisher-Yates 洗牌乱序
		for i := len(rawTiles) - 1; i > 0; i-- {
			jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
			if err != nil {
				return nil, fmt.Errorf("洗牌随机数生成失败: %w", err)
			}
			j := int(jBig.Int64())
			rawTiles[i], rawTiles[j] = rawTiles[j], rawTiles[i]
		}

		// 为乱序后的每个槽位分配唯一的会话级 ID，防止同一候选图片多次出现时 ID 冲突
		chosenTiles = make([]Tile, totalSlots)
		for idx, t := range rawTiles {
			slotID := fmt.Sprintf("%s_pos%d", t.ID, idx)
			chosenTiles[idx] = Tile{
				ID:       slotID,
				ImageURL: t.ImageURL,
				IsTarget: t.IsTarget,
			}
			if t.IsTarget {
				correctIDs[slotID] = true
			}
		}
	}

	// 4. 构建脱敏后供前端渲染的 ClientTile 列表
	clientTiles := make([]ClientTile, len(chosenTiles))
	for idx, t := range chosenTiles {
		clientTiles[idx] = ClientTile{
			ID:       t.ID,
			ImageURL: t.ImageURL,
		}
	}

	// 5. 若未提供会话 ID 则生成全新的安全随机令牌
	if sessionID == "" {
		sessionID = generateRandomID(16)
	}

	session, exists := e.sessions[sessionID]
	if !exists {
		session = &SessionState{
			SessionID: sessionID,
			FailCount: 0,
		}
		e.sessions[sessionID] = session
	}

	// 更新当前会话的活跃题目与正确答案集合
	session.ActiveChallengeID = challengeID
	session.CorrectTileIDs = correctIDs

	// 生成每次出题唯一的实例 ID
	instanceID := generateRandomID(8)

	return &ClientChallenge{
		ID:          challengeID,
		InstanceID:  instanceID,
		SessionID:   sessionID,
		Mode:        mode,
		Rows:        rows,
		Columns:     cols,
		Title:       original.Title,
		Target:      original.Target,
		Instruction: original.Instruction,
		Seamless:    original.Seamless,
		Tiles:       clientTiles,
	}, nil
}

// sampleTiles 从候选池 source 中随机抽取 count 个元素。
// 若候选池元素充足，采用无放回抽样；若候选池元素不足，则进行有放回安全抽样。
func sampleTiles(source []Tile, count int) ([]Tile, error) {
	if len(source) == 0 || count <= 0 {
		return []Tile{}, nil
	}

	result := make([]Tile, 0, count)

	if len(source) >= count {
		// 候选数量充足：复制并洗牌前 count 个（无放回）
		poolCopy := make([]Tile, len(source))
		copy(poolCopy, source)

		for i := len(poolCopy) - 1; i > 0; i-- {
			jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
			if err != nil {
				return nil, err
			}
			j := int(jBig.Int64())
			poolCopy[i], poolCopy[j] = poolCopy[j], poolCopy[i]
		}
		result = append(result, poolCopy[:count]...)
	} else {
		// 候选数量不足：随机有放回抽样填满 count 个
		for i := 0; i < count; i++ {
			idxBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(source))))
			if err != nil {
				return nil, err
			}
			result = append(result, source[idxBig.Int64()])
		}
	}

	return result, nil
}

// generateRandomID 利用 crypto/rand 生成指定字节长度的加密安全十六进制随机字符串。
func generateRandomID(bytesLen int) string {
	b := make([]byte, bytesLen)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
