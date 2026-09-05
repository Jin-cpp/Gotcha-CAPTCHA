// Package main 为 touhou.boo（多多良小伞整蛊网站）的后端服务启动入口。
// 负责组装 gotcha-go 引擎、挂载东方专属题库、提供 RESTful 接口与静态资源服务。
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	gotchago "github.com/jin-cpp/gotcha-captcha/packages/gotcha-go"
)

// AppConfig 定义整蛊业务配置项（映射自 config.json）。
type AppConfig struct {
	AppName              string `json:"app_name"`                // 业务应用名称
	Title                string `json:"title"`                   // 界面展示的门禁标题
	Port                 int    `json:"port"`                    // HTTP 服务监听端口
	MaxFailsToScare      int    `json:"max_fails_to_scare"`      // 触发小伞惊吓的最大连续失败次数
	MinNoRepeatCount     int    `json:"min_no_repeat_count"`     // 至少不会抽到重复题的次数
	MinNonRepeatingCount int    `json:"min_non_repeating_count"` // 兼容别名（支持不同命名习惯）
}

// loadConfig 尝试从多个候选路径加载并解析 config.json 配置文件；
// 若未找到配置文件或解析失败，则平滑回退至预置的默认配置。
func loadConfig() AppConfig {
	cfg := AppConfig{
		AppName:          "touhou-boo",
		Title:            "博丽神社香油钱安全验证",
		Port:             8080,
		MaxFailsToScare:  3,
		MinNoRepeatCount: 2,
	}

	paths := []string{"../config.json", "./config.json", "apps/touhou-boo/config.json"}
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			if err := json.Unmarshal(data, &cfg); err == nil {
				if cfg.MinNoRepeatCount <= 0 && cfg.MinNonRepeatingCount > 0 {
					cfg.MinNoRepeatCount = cfg.MinNonRepeatingCount
				}
				log.Printf("成功从配置文件 %s 装载业务参数 (防重复抽题次数: %d)", p, cfg.MinNoRepeatCount)
				break
			}
		}
	}
	return cfg
}

// enableCORS 跨域请求处理中间件。
// 允许 Vite 前端开发服务器（如 http://localhost:5173）跨域调用后端接口与静态资源。
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// 处理预检请求 (Preflight)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func main() {
	// 1. 初始化业务配置
	cfg := loadConfig()

	// 2. 实例化通用的 gotcha-go 验证引擎，并注入小伞整蛊阈值与至少不重复抽题次数
	engine := gotchago.NewEngine(cfg.MaxFailsToScare, cfg.MinNoRepeatCount)

	// 3. 动态定位题库与静态资源目录（兼容从根目录或应用目录启动）
	dataDir := "./data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		dataDir = "apps/touhou-boo/backend/data"
	}
	challengesDir := filepath.Join(dataDir, "challenges")
	assetsDir := filepath.Join(dataDir, "assets")

	// 4. 加载东方专属题库（如灵梦辨识、雨伞辨识）
	if err := engine.LoadChallengesFromDir(challengesDir); err != nil {
		log.Fatalf("从 %s 加载题库失败: %v", challengesDir, err)
	}
	log.Printf("东方题库加载成功，题目来源: %s", challengesDir)

	mux := http.NewServeMux()

	// 路由 1: 获取前端展示所需的业务配置信息 (GET /api/config)
	mux.HandleFunc("/api/config", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)
	}))

	// 路由 2: 生成或刷新一道经过乱序洗牌的题目 (GET /api/challenge)
	mux.HandleFunc("/api/challenge", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("sessionId")
		challenge, err := engine.GenerateChallenge(sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(challenge)
	}))

	// 路由 3: 接收受试者提交的答案并执行校验 (POST /api/verify)
	mux.HandleFunc("/api/verify", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "仅支持 POST 请求", http.StatusMethodNotAllowed)
			return
		}

		var req gotchago.VerifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "无效的 JSON 请求负载", http.StatusBadRequest)
			return
		}

		result, err := engine.Verify(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}))

	// 路由 4: 静态资源托管（题目素材图片如 reimu.svg, kogasa.svg）
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir))))

	// 5. 启动 HTTP 监听服务
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("👻 多多良小伞整蛊后端服务启动于 http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("服务运行异常: %v", err)
	}
}
