# Gotcha-CAPTCHA 🎭

> 一款高拟真、组件化、可扩展的整蛊伪装验证码开发框架。
> "看起来是 Google reCAPTCHA，其实是精心准备的恶作剧！"

📖 **完整技术白皮书与架构文档**：请参阅 [TECHNICAL_DOCUMENTATION.md](./TECHNICAL_DOCUMENTATION.md)。

---

## 🌟 框架特性

1. **高拟真视觉伪装**：
   - 像素级还原 Google reCAPTCHA v2 交互形态（经典复选框旋转动画、蓝色说明横幅、九宫格微缩点击反馈）。
   - 让受访者毫无防备地进入“验证”流程。
2. **业务与通用内核解耦 (Monorepo)**：
   - **`packages/` (通用框架层)**：完全独立无污染，提供标准 React 验证码组件与 Go 题库/校验引擎，未来可直接发包至 npm / Go pkg。
   - **`apps/` (业务应用层)**：各具特色的整蛊应用落地（如东方 Project 主题的多多良小伞整蛊站 `touhou-boo`）。
3. **强大的题库与洗牌引擎**：
   - 基于 Fisher-Yates 算法的题库图片乱序重排机制。
   - 数据与校验分离：敏感目标标记在服务端校验，脱敏后下发客户端，杜绝审查元素作弊。
4. **灵活的整蛊触发机制**：
   - 支持连续失败次数阈值控制（`max_fails_to_scare`）。
   - 触发时支持外部回调（弹窗、碎屏、音效、全屏跳转等自定义整蛊特效）。

---

## 📁 代码架构

```
gotcha-captcha/                      # 项目根目录 (Monorepo)
├── README.md                        # 框架总说明
├── go.work                          # Go 工作区配置
├── package.json                     # 前端 npm workspaces 配置
│
├── packages/                        # 📦 通用框架层 (完全独立，不含特定业务元素)
│   ├── gotcha-react/                # 纯净 React 伪装验证码组件库
│   └── gotcha-go/                   # 纯净 Go 校验与题库引擎
│
└── apps/                            # 🚀 业务应用层 (框架的最佳实践示例)
    └── touhou-boo/                  # 多多良小伞整蛊网站 (Touhou Kogasa Jumpscare)
        ├── README.md
        ├── config.json
        ├── backend/                 # Go 后端服务
        └── frontend/                # React + Vite 前端应用
```

---

## 🚀 快速开始

### 依赖环境
- **Go**: 1.22 及以上
- **Node.js**: 18 及以上 (推荐 20+)

### 1. 启动后端服务 (`apps/touhou-boo/backend`)

```bash
# 进入后端目录并启动
cd apps/touhou-boo/backend
go run main.go
# 默认监听 http://localhost:8080
```

### 2. 启动前端页面 (`apps/touhou-boo/frontend`)

```bash
# 在项目根目录下安装所有前端依赖
npm install

# 启动前端开发服务器
npm run dev
# 默认访问 http://localhost:5173
```

---

## 🛠️ 二次开发指南

### 如何接入自定义整蛊主题？

1. **题库定制**：在 `apps/<your-app>/backend/data/challenges/` 下新增 JSON 题目，并在 `assets/` 放入素材。
2. **前端接入**：
   ```jsx
   import { FakeCaptcha } from 'gotcha-react';
   import 'gotcha-react/src/style.css';

   function App() {
     return (
       <FakeCaptcha
         challenge={currentChallenge}
         onVerify={handleVerify}
         onScare={triggerCustomJumpscare}
       />
     );
   }
   ```
3. **调整触发阈值**：通过 `config.json` 设置连续尝试几次后触发最终整蛊效果。

---

## 📄 开源许可

本项目遵循 [GPL-3.0](./LICENSE) 开源协议。仅供技术交流与娱乐整蛊，请勿用于非法或恶意骚扰用途。

