import React, { useState, useEffect, useCallback } from 'react';
import { createRoot } from 'react-dom/client';
import { FakeCaptcha } from 'gotcha-react';
import KogasaScare from './components/KogasaScare.jsx';
import './App.css';

/**
 * App: 多多良小伞整蛊网站（Touhou Boo）主页面。
 * 业务演示层核心组件：构造“博丽神社赛钱安全验证”伪装场景，
 * 接入纯净框架层的 FakeCaptcha，并与 KogasaScare 惊吓特效进行联动。
 */
export function App() {
  // 1. 业务与题目状态
  const [challenge, setChallenge] = useState(null);       // 当前活跃的挑战题目数据
  const [sessionId, setSessionId] = useState('');         // 服务端下发的会话唯一标识符
  const [failCount, setFailCount] = useState(0);          // 当前连续失败尝试次数
  const [maxFails, setMaxFails] = useState(3);            // 触发整蛊的最大失败次数阈值（从后端配置获取）
  const [isScared, setIsScared] = useState(false);        // 是否正处于小伞全屏惊吓状态
  const [isVerified, setIsVerified] = useState(false);    // 是否已成功通过验证码

  /**
   * 向 Go 后端请求一道新的打乱题目
   * @param {string} [currentSessionId=""] 现有的会话 ID（若有则保持会话连续性）
   */
  const loadChallenge = useCallback(async (currentSessionId = '') => {
    try {
      const url = currentSessionId
        ? `/api/challenge?sessionId=${encodeURIComponent(currentSessionId)}`
        : '/api/challenge';
      const res = await fetch(url);
      if (res.ok) {
        const data = await res.json();
        setChallenge(data);
        if (data.sessionId) {
          setSessionId(data.sessionId);
        }
      }
    } catch (err) {
      console.error('拉取验证码题目失败:', err);
    }
  }, []);

  // 页面首屏加载：获取业务配置及初始题目
  useEffect(() => {
    fetch('/api/config')
      .then((r) => r.json())
      .then((cfg) => {
        if (cfg.max_fails_to_scare) {
          setMaxFails(cfg.max_fails_to_scare);
        }
      })
      .catch(() => {});

    loadChallenge();
  }, [loadChallenge]);

  /**
   * 提交受试者勾选的方块答案至后端校验
   * @param {string[]} selectedIds 用户选中的方块 ID 数组
   * @returns {Promise<Object>} 校验结果对象
   */
  const handleVerify = async (selectedIds) => {
    if (!challenge) return { success: false };

    const res = await fetch('/api/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        challengeId: challenge.id,
        sessionId: sessionId,
        selectedIds: selectedIds,
      }),
    });

    const result = await res.json();

    // 更新连续失败次数
    if (result.failCount !== undefined) {
      setFailCount(result.failCount);
    }

    // 1. 触发整蛊分支：打开 KogasaScare 惊吓浮层
    if (result.triggerScare) {
      setIsScared(true);
      return { triggerScare: true };
    }

    // 2. 验证成功分支
    if (result.success) {
      setIsVerified(true);
      setFailCount(0);
      return { success: true };
    }

    // 3. 验证未通过：更新为后端下发的新乱序题目
    if (result.nextChallenge) {
      setChallenge(result.nextChallenge);
    }

    return {
      success: false,
      message: result.message,
    };
  };

  /**
   * 换一题（手动刷新）
   */
  const handleRefresh = () => {
    loadChallenge(sessionId);
  };

  /**
   * 受试者在惊吓弹窗中点击“再试一次”后，重置状态重新挑战
   */
  const handleScareReset = () => {
    setIsScared(false);
    setFailCount(0);
    loadChallenge();
  };

  return (
    <div className="portal-container">
      {/* 门禁卡片容器 */}
      <div className="portal-card">
        {/* 伪装的官方神社认证标头 */}
        <div className="shrine-badge">⛩️ 博丽神社官方通道</div>
        <h1 className="portal-title">赛钱支付安全验证</h1>
        <p className="portal-subtitle">
          检测到来自外界的异变访问。为防止河童自动化脚本盗刷塞钱箱，请完成人机身份验证。
        </p>

        {/* 伪装验证码挂载区域 */}
        <div className="captcha-mount-area">
          <FakeCaptcha
            challenge={challenge}
            onVerify={handleVerify}
            onRefresh={handleRefresh}
            onScare={() => setIsScared(true)}
            failCount={failCount}
            maxFails={maxFails}
            checkboxLabel="证明你不是河童制造的机器人"
            onBrandClick={(clickCount) => {
              console.log(`[Gotcha-CAPTCHA] 伪装品牌标识已累计被点击 ${clickCount} 次`);
            }}
          />
        </div>

        {/* 验证通过成功提示 */}
        {isVerified && (
          <div style={{ color: '#2e7d32', fontWeight: 'bold', margin: '12px 0' }}>
            🎉 验证通过！赛钱箱已解锁（灵梦露出了欣慰的笑容）。
          </div>
        )}

        {/* 页脚安全申明 */}
        <div className="portal-footer">
          Hakurei Shrine Cyber Security Division · Powered by Gotcha-CAPTCHA
        </div>
      </div>

      {/* 连续失败达上限时全屏展示的小伞惊吓模态组件 */}
      {isScared && (
        <KogasaScare
          onClose={handleScareReset}
          audioSrc="/urameshiya.mp3"
          imageSrc="/kogasa_scare.svg"
        />
      )}
    </div>
  );
}

export default App;

// 浏览器环境下自动挂载至 index.html 中的 #root 节点
const rootEl = document.getElementById('root');
if (rootEl) {
  createRoot(rootEl).render(<App />);
}

