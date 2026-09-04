# 多多良小伞的整蛊验证码 (Touhou Boo) 👻☂️

> "恨めしや〜！ (Urameshia~!) 吓到了吧？！" —— 多多良小伞

---

## 📖 作案动机与背景故事

**多多良小伞**（Tatara Kogasa）是幻想乡中由被遗忘的长柄唐伞化成的附丧神妖怪。
她的终身追求只有一个——**吓唬人类并收集惊吓点数**。

然而，在和平的现代社会，人类已经对传统的幽灵作怪见怪不怪了。
小伞潜心观察人类社会后发现：**现代人类最容易破防、最烦躁、最容易尖叫的瞬间，就是面对永远点不完的验证码！**

于是，小伞用妖力伪装成了 Google reCAPTCHA 验证码：
1. 题目是辨识东方角色（例如“请在九宫格里找出所有的博丽灵梦”）。
2. 但混入了一堆长得很像、或者容易混淆的角色（魔理沙、小伞自己、早苗等）。
3. 一旦人类连续选错达到上限（默认 3 次），小伞的本尊就会伴随着经典的「恨めしや〜！」怪叫、碎屏特效和满屏乱晃直接跳出来！

---

## ⚙️ 业务配置 (`config.json`)

```json
{
  "app_name": "touhou-boo",
  "title": "博丽神社香油钱安全验证",
  "port": 8080,
  "max_fails_to_scare": 3,
  "min_no_repeat_count": 2,
  "audio_file": "/urameshiya.mp3",
  "scare_image": "/kogasa_scare.svg"
}
```

- `max_fails_to_scare`：连续失败多少次后触发小伞惊吓（Jumpscare）。
- `min_no_repeat_count`：至少不会抽到重复题目的次数（通过滑动队列算法维护无重复出题池，长度为 $\min\{\text{min\_no\_repeat\_count}, \text{总题数}\}$）。

---

## 🚀 启动方式

### 1. 启动后端

```bash
cd apps/touhou-boo/backend
go run main.go
```

### 2. 启动前端

```bash
cd apps/touhou-boo/frontend
npm install
npm run dev
```

打开浏览器访问 `http://localhost:5173` 体验。

