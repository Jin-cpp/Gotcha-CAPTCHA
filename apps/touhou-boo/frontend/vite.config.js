/**
 * @file apps/touhou-boo/frontend/vite.config.js
 * @description 前端 Vite 工程构建配置文件。
 * 集成 React 官方插件，并配置反向代理将 API 与静态资源请求转发至 Go 后端。
 */
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  // 启用 React JSX 编译与 Fast Refresh 热更新支持
  plugins: [react()],
  server: {
    port: 5173, // 前端开发服务器端口
    build: {
        assetsDir: 'static', // 将前端打包产物目录由 assets 改为 static
    },
    proxy: {
      // 代理后端业务 API（如获取配置、获取挑战题目、验证答案）
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      // 代理后端托管的题库图片与素材静态资源
      '/assets': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});


