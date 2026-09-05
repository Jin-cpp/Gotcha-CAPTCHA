// Package gotchago 提供纯净通用的整蛊伪装验证码核心引擎与校验模型。
// 本模块不包含任何特定业务或主题元素，专注于题库装载、图片乱序洗牌与答案判定逻辑。
package gotchago

// ChallengeMode 定义验证码出题模式。
type ChallengeMode string

const (
	// ModeRandomPool 候选池随机抽取模式：
	// 从目标池与干扰池中随机抽取图片填充指定矩阵，严格保证至少包含一张正确目标图片，并执行洗牌乱序。
	ModeRandomPool ChallengeMode = "random_pool"

	// ModeFixedLayout 全位置指定固定模式：
	// 矩阵的每一个位置坐标（0 到 Rows*Columns - 1）指定固定的图片与正确性标记，适用于拼图切割或固定编排。
	ModeFixedLayout ChallengeMode = "fixed_layout"
)

// Tile 表示矩阵挑战中的单个候选方块元素（服务端完整结构）。
type Tile struct {
	ID       string `json:"id"`                 // 方块的唯一标识符（如 t1, t2）
	ImageURL string `json:"imageUrl"`           // 方块图片的访问地址或静态资源路径
	IsTarget bool   `json:"isTarget,omitempty"` // 该方块是否为正确答案（仅限服务端判定，脱敏后禁止下发给客户端）
}

// ClientTile 表示安全脱敏后下发给前端浏览器的矩阵方块数据。
// 移除了敏感的 IsTarget 标记，防止受试者通过浏览器审查元素作弊。
type ClientTile struct {
	ID       string `json:"id"`       // 方块的唯一标识符
	ImageURL string `json:"imageUrl"` // 方块图片的静态资源路径
}

// Challenge 表示题库中加载的挑战题目完整定义（服务端视图）。
type Challenge struct {
	ID             string        `json:"id"`                       // 题目的唯一编号（如 reimu_detection）
	Mode           ChallengeMode `json:"mode"`                     // 出题模式：random_pool（候选池随机抽选）或 fixed_layout（全位置指定固定）
	Rows           int           `json:"rows"`                     // 图片矩阵行数（默认 3）
	Columns        int           `json:"columns"`                  // 图片矩阵列数（默认 3）
	Title          string        `json:"title"`                    // 挑战大标题（如：“选择所有包含以下内容的方块”）
	Target         string        `json:"target"`                   // 目标对象名称（如：“博丽灵梦”）
	Instruction    string        `json:"instruction"`              // 辅助说明引导文字
	MinTargets     int           `json:"minTargets,omitempty"`     // 随机池模式下：抽取的最小目标数（默认至少 1 张）
	MaxTargets     int           `json:"maxTargets,omitempty"`     // 随机池模式下：抽取的最大目标数（默认无限制）
	TargetPool     []Tile        `json:"targetPool,omitempty"`     // 随机池模式：正确目标图片候选池
	DistractorPool []Tile        `json:"distractorPool,omitempty"` // 随机池模式：干扰项图片候选池
	Pool           []Tile        `json:"pool,omitempty"`           // 随机池模式：统一候选池（由各方块的 isTarget 标记正误）
	FixedTiles     []Tile        `json:"fixedTiles,omitempty"`     // 固定布局模式：按坐标位置指定的固定方块列表
	Tiles          []Tile        `json:"tiles,omitempty"`          // 向后兼容字段：若未指定 pools/fixedTiles 则作为默认方块列表
	Seamless       bool          `json:"seamless,omitempty"`       // 是否启用无间隙贴合模式（适用于单张大图切割切片题型）
}

// ClientChallenge 表示下发给前端界面的挑战题目（客户端脱敏视图）。
type ClientChallenge struct {
	ID          string        `json:"id"`          // 题目的唯一编号
	InstanceID  string        `json:"instanceId"`  // 每次出题生成的唯一实例编号（用于前端精准追踪换题与清空选择项）
	SessionID   string        `json:"sessionId"`   // 会话标识符，用于跟踪连续验证尝试次数
	Mode        ChallengeMode `json:"mode"`        // 出题模式
	Rows        int           `json:"rows"`        // 矩阵行数
	Columns     int           `json:"columns"`     // 矩阵列数
	Title       string        `json:"title"`       // 挑战大标题
	Target      string        `json:"target"`      // 目标对象名称
	Instruction string        `json:"instruction"` // 辅助说明文字
	Seamless    bool          `json:"seamless"`    // 是否启用无间隙贴合模式
	Tiles       []ClientTile  `json:"tiles"`       // 组装并脱敏后的方块列表（长度为 Rows * Columns）
}

// VerifyRequest 表示受试者在前端点击“验证”按钮后提交的答案数据结构。
type VerifyRequest struct {
	ChallengeID string   `json:"challengeId"` // 当前作答的题目编号
	SessionID   string   `json:"sessionId"`   // 会话标识符
	SelectedIDs []string `json:"selectedIds"` // 用户勾选的所有方块 ID 数组
}

// VerifyResult 表示服务端对答卷判定后的响应反馈。
type VerifyResult struct {
	Success       bool             `json:"success"`                 // 验证是否通过
	Message       string           `json:"message"`                 // 反馈提示文本（如：“验证失败，请重试”）
	FailCount     int              `json:"failCount"`               // 当前会话连续失败的累计次数
	TriggerScare  bool             `json:"triggerScare"`            // 是否达到阈值并触发整蛊（Jumpscare）
	NextChallenge *ClientChallenge `json:"nextChallenge,omitempty"` // 若失败则附带下一道重新打乱的题目
}
