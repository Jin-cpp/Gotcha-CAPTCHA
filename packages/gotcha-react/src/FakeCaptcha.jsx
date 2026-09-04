import React, { useState, useEffect } from 'react';
import gotchaLogo from './assets/gotcha_logo.svg';
import './style.css';

/**
 * FakeCaptcha: 高拟真 Google reCAPTCHA v2 伪装验证码核心组件。
 * 纯通用组件库层实现，不包含特定业务或整蛊主题代码。
 *
 * @param {Object} props
 * @param {Object} props.challenge 当前活跃的验证题目数据 { id, title, target, instruction, tiles: [{ id, imageUrl }] }
 * @param {Function} props.onVerify 异步校验回调 (selectedIds) => Promise<{ success: boolean, triggerScare?: boolean, message?: string, nextChallenge?: object }>
 * @param {Function} props.onRefresh 获取新题目的刷新回调
 * @param {Function} props.onScare 达到整蛊阈值时的触发回调
 * @param {string} [props.checkboxLabel="我不是机器人"] 复选框旁展示的文本
 * @param {string} [props.brandLogoSrc] 品牌标识外置图片素材地址，默认为 Gotcha-CAPTCHA 专属徽标
 * @param {number} [props.checkDuration=1200] 点击复选框后思考旋转动画持续时间 (ms)
 */
export function FakeCaptcha({
  challenge,
  onVerify,
  onRefresh,
  onScare,
  failCount = 0,
  maxFails = 3,
  checkboxLabel = '我不是机器人',
  brandLogoSrc = gotchaLogo,
  checkDuration = 1200,
}) {
  // 1. 组件内部交互状态
  const [isChecked, setIsChecked] = useState(false);        // 验证码是否已成功通过打勾
  const [isChecking, setIsChecking] = useState(false);      // 点击复选框后的“计算/思考中”转圈状态
  const [showPopup, setShowPopup] = useState(false);        // 是否弹出九宫格验证弹窗
  const [selectedIds, setSelectedIds] = useState(new Set());// 当前用户已勾选的方块 ID 集合
  const [isVerifying, setIsVerifying] = useState(false);    // 点击“验证”后的网络请求等待状态
  const [errorMessage, setErrorMessage] = useState('');     // 校验失败时的错误提示横幅内容

  // 构造题目实例唯一标识（优先使用 instanceId，若无则结合 id 与方块 ID 序列）
  const challengeKey = challenge?.instanceId || (challenge?.id ? `${challenge.id}_${challenge?.tiles?.map((t) => t.id).join(',')}` : '');

  // 当题目发生变化（换题、重新洗牌或刷新）时，必须清空之前勾选的方块
  useEffect(() => {
    setSelectedIds(new Set());
  }, [challengeKey]);

  /**
   * 处理首屏复选框点击事件
   * 模拟真实 Google reCAPTCHA 的变圆缩小消失并放大展示环形旋转动画，随后展开验证弹窗
   */
  const handleCheckboxClick = () => {
    if (isChecked || isChecking) return;
    setIsChecking(true);
    setErrorMessage('');

    // 延迟 checkDuration（默认 1200ms），让变圆变小消失与旋转进度条完整展示
    setTimeout(() => {
      setIsChecking(false);
      setShowPopup(true);
    }, checkDuration);
  };

  /**
   * 切换方块的选中状态
   * @param {string} tileId 方块唯一标识符
   */
  const handleTileClick = (tileId) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(tileId)) {
        next.delete(tileId);
      } else {
        next.add(tileId);
      }
      return next;
    });
  };

  /**
   * 提交当前所选方块进行答案校验
   */
  const handleVerify = async () => {
    if (isVerifying) return;
    setIsVerifying(true);
    setErrorMessage('');

    try {
      const result = await onVerify?.(Array.from(selectedIds));
      
      // 1. 若触发了整蛊条件（如连续失败达上限）
      if (result?.triggerScare) {
        setShowPopup(false);
        onScare?.();
        return;
      }

      // 2. 验证通过
      if (result?.success) {
        setIsChecked(true);
        setShowPopup(false);
      } else {
        // 3. 验证未通过，展示提示文本并等待换题
        setErrorMessage(
          result?.message ||
            (failCount > 0
              ? `未能通过验证，已失败 ${failCount + 1}/${maxFails} 次。请再试一次！`
              : '未能通过验证，请再试一次！')
        );
      }
    } catch (err) {
      setErrorMessage(err.message || '网络验证异常，请重试');
    } finally {
      setIsVerifying(false);
    }
  };

  /**
   * 点击刷新按钮，清空选项并请求新题目
   */
  const handleReload = () => {
    setSelectedIds(new Set());
    setErrorMessage('');
    onRefresh?.();
  };

  // 动态矩阵规格：提取列数与行数（默认 3x3）
  const columns = challenge?.columns || 3;
  const rows = challenge?.rows || 3;
  // 根据列数自适应弹窗宽度（3列约390px，4列约480px，2列约320px）
  const popupWidth = Math.min(Math.max(columns * 115 + 40, 320), 560);

  return (
    <div className="gotcha-wrapper">
      {/* 1. Google reCAPTCHA 经典复选框锚点组件 */}
      <div className="gotcha-anchor" onClick={handleCheckboxClick}>
        <div className="gotcha-checkbox-container">
          <div className="gotcha-checkbox-wrapper">
            <div
              className={`gotcha-checkbox ${isChecked ? 'checked' : ''} ${
                isChecking ? 'checking' : ''
              }`}
            >
              {isChecked && (
                <svg className="gotcha-checkmark" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z" />
                </svg>
              )}
            </div>
            {isChecking && (
              <div className="gotcha-spinner-container">
                <div className="gotcha-spinner" />
              </div>
            )}
          </div>
          <span className="gotcha-label">{checkboxLabel}</span>
        </div>

        {/* 伪装的 Google reCAPTCHA 品牌标识 */}
        <div className="gotcha-branding">
          <img
            src={brandLogoSrc}
            alt="GOPTCHA logo"
            className={`gotcha-brand-logo gotcha-logo ${isChecking ? 'spinning' : ''}`}
            draggable="false"
          />
          <span className="gotcha-brand-text">GOPTCHA</span>
          <div className="gotcha-links">
            <a href="#privacy" onClick={(e) => e.stopPropagation()}>隐私</a>
            <span>-</span>
            <a href="#terms" onClick={(e) => e.stopPropagation()}>条款</a>
          </div>
        </div>
      </div>

      {/* 2. 模态验证弹窗 */}
      {showPopup && challenge && (
        <div className="gotcha-popup-overlay">
          <div className="gotcha-popup" style={{ maxWidth: `${popupWidth}px`, width: '92vw' }}>
            {/* 顶部说明横幅（经典 Google 蓝） */}
            <div className="gotcha-header">
              <p className="gotcha-header-prompt">{challenge.title || '选择所有包含以下内容的方块'}</p>
              <h2 className="gotcha-header-target">{challenge.target || challenge.prompt || '目标对象'}</h2>
              <p className="gotcha-header-instruction">
                {challenge.instruction || '如果没有任何方块符合要求，请直接点击“验证”。'}
              </p>
            </div>

            {/* 错误警告条 */}
            {errorMessage && (
              <div className="gotcha-error-bar">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z" />
                </svg>
                <span>{errorMessage}</span>
              </div>
            )}

            {/* 图片矩阵网格 (动态 repeat 列数) */}
            <div className="gotcha-grid-container">
              <div
                className="gotcha-grid"
                style={{
                  gridTemplateColumns: `repeat(${columns}, 1fr)`,
                }}
              >
                {challenge.tiles?.map((tile) => {
                  const isSelected = selectedIds.has(tile.id);
                  return (
                    <div
                      key={tile.id}
                      className={`gotcha-tile ${isSelected ? 'selected' : ''}`}
                      onClick={() => handleTileClick(tile.id)}
                    >
                      <img src={tile.imageUrl} alt="challenge item" />
                      {/* 选中时的蓝色对勾徽章 */}
                      {isSelected && (
                        <div className="gotcha-tile-badge">
                          <svg viewBox="0 0 24 24" fill="currentColor">
                            <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z" />
                          </svg>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>

            {/* 底部操作与验证工具栏 */}
            <div className="gotcha-footer">
              <div className="gotcha-tools">
                {/* 换题刷新按钮 */}
                <button
                  type="button"
                  className="gotcha-tool-btn"
                  title="获取新的验证码"
                  onClick={handleReload}
                >
                  <svg viewBox="0 0 24 24" fill="currentColor">
                    <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z" />
                  </svg>
                </button>
                {/* 伪装的无障碍语音验证按钮 */}
                <button
                  type="button"
                  className="gotcha-tool-btn"
                  title="语音验证"
                  onClick={() => alert('语音功能维护中，请使用视觉验证。')}
                >
                  <svg viewBox="0 0 24 24" fill="currentColor">
                    <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z" />
                  </svg>
                </button>
                {/* 帮助提示按钮 */}
                <button
                  type="button"
                  className="gotcha-tool-btn"
                  title="帮助"
                  onClick={() => alert('仔细阅读题目，点击符合条件的图片，最后点击验证。')}
                >
                  <svg viewBox="0 0 24 24" fill="currentColor">
                    <path d="M11 18h2v-2h-2v2zm1-16C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm0-14c-2.21 0-4 1.79-4 4h2c0-1.1.9-2 2-2s2 .9 2 2c0 2-3 1.75-3 5h2c0-2.25 3-2.5 3-5 0-2.21-1.79-4-4-4z" />
                  </svg>
                </button>
              </div>

              {/* 核心验证按钮 */}
              <button
                type="button"
                className="gotcha-verify-btn"
                onClick={handleVerify}
                disabled={isVerifying}
              >
                {isVerifying ? '正在验证...' : '验证'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default FakeCaptcha;

