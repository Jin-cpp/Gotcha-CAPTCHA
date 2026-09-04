import React, { useEffect, useRef } from 'react';

/**
 * KogasaScare: 多多良小伞专属的惊吓与碎屏动画组件。
 * 当受试者在验证码中连续选错并触发整蛊阈值时全屏弹出。
 *
 * @param {Object} props
 * @param {Function} props.onClose 关闭整蛊弹层并重置验证状态的回调函数
 * @param {string} [props.audioSrc="/urameshiya.mp3"] 惊吓音效文件地址
 * @param {string} [props.imageSrc="/kogasa_scare.svg"] 小伞高清惊吓大图矢量地址
 */
export function KogasaScare({
  onClose,
  audioSrc = '/urameshiya.mp3',
  imageSrc = '/kogasa_scare.svg',
}) {
  const audioRef = useRef(null);

  useEffect(() => {
    // 1. 初始化并播放小伞经典“恨めしや〜！”惊吓音频
    try {
      const audio = new Audio(audioSrc);
      audio.volume = 0.85;
      // 处理浏览器自动播放策略限制（若被拦截仅输出警告，不中断界面渲染）
      audio.play().catch((err) => {
        console.warn('音频自动播放被浏览器策略拦截或失败:', err);
      });
      audioRef.current = audio;
    } catch (e) {
      console.warn('初始化音频对象异常:', e);
    }

    // 2. 向页面 body 节点追加震动 class，触发全屏剧烈晃动动效
    document.body.classList.add('screen-shaking');

    // 3. 组件卸载时清理：移除震屏 class 并停止音频
    return () => {
      document.body.classList.remove('screen-shaking');
      if (audioRef.current) {
        audioRef.current.pause();
      }
    };
  }, [audioSrc]);

  return (
    <div className="kogasa-scare-overlay">
      {/* 视觉特效层 1: 红色高频警示闪光 (Strobe Flash) */}
      <div className="kogasa-flash" />

      {/* 视觉特效层 2: 碎屏玻璃放射状裂纹 (SVG Shatter Cracks) */}
      <svg className="kogasa-glass-crack" viewBox="0 0 1000 1000" preserveAspectRatio="none">
        <path
          d="M 500 500 L 0 200 M 500 500 L 250 0 M 500 500 L 750 0 M 500 500 L 1000 150 M 500 500 L 1000 650 M 500 500 L 800 1000 M 500 500 L 200 1000 M 500 500 L 0 750"
          stroke="rgba(255, 255, 255, 0.7)"
          strokeWidth="3"
          fill="none"
        />
        <path
          d="M 400 420 L 450 350 L 550 380 L 600 480 L 520 580 L 420 540 Z"
          stroke="rgba(255, 255, 255, 0.85)"
          strokeWidth="2.5"
          fill="rgba(255, 255, 255, 0.08)"
        />
        <path
          d="M 450 350 L 320 280 M 550 380 L 680 300 M 600 480 L 780 520 M 520 580 L 580 720 M 420 540 L 300 640"
          stroke="rgba(255, 255, 255, 0.6)"
          strokeWidth="2"
          fill="none"
        />
      </svg>

      {/* 核心内容层: 小伞大图突进与对话框 */}
      <div className="kogasa-scare-content">
        {/* 小伞突进大图 */}
        <img
          src={imageSrc}
          alt="Tatara Kogasa Jumpscare"
          className="kogasa-scare-image"
        />

        {/* 经典日系 AVG/漫画风格对话框 */}
        <div className="kogasa-dialogue-box">
          <div className="kogasa-dialogue-header">
            <span className="kogasa-tag">附丧神 · 多多良小伞</span>
            <span className="kogasa-status">👻 惊吓成功！</span>
          </div>
          <p className="kogasa-quote">
            「恨めしや〜！ (Urameshia~!) 吓到了吧？！哈哈哈！」
          </p>
          <p className="kogasa-desc">
            人类啊，被本唐伞大妖怪伪装的验证码折磨得抓狂了吧！别再点啦，你根本猜不对的~
          </p>
          {/* 重置与重试按钮 */}
          <button type="button" className="kogasa-retry-btn" onClick={onClose}>
            不服！再试一次 (再被吓一次)
          </button>
        </div>
      </div>
    </div>
  );
}

export default KogasaScare;

