# AimiliVPN 前端 UI/UX 全面优化方案

## 📋 项目概况

**当前技术栈**：
- **架构**：Vanilla HTML + CSS + JavaScript (无框架依赖)
- **字体**：Geist (400/500/600/700 多字重)
- **设计语言**：深色主题 (Dark Mode)
- **代码规模**：~5,247 行 (HTML 963行 / CSS 1,884行 / JS 2,400行)

**核心功能模块**：
1. 运行概览仪表板 (Dashboard)
2. Sing-box 边缘抗封锁入站管理
3. 多端口代理与自适应分流矩阵
4. 优质 VPN 节点广场
5. 系统与安全参数设置

---

## 🎯 优化目标与原则

### 设计目标
1. **专业可信赖感**：网络安全与基础设施工具应传递稳定、可控、专业的视觉印象
2. **信息密度与可读性平衡**：Dashboard 需要高信息密度，但不能牺牲视觉呼吸感
3. **响应式适配**：桌面端优先，移动端底部导航栏已实现
4. **无障碍合规**：ARIA 标签完善，键盘导航友好

### 优化原则
- **渐进式增强**：保持现有技术栈，不引入新的框架依赖
- **性能优先**：避免大量 DOM 操作，优化渲染性能
- **向后兼容**：确保现有功能不受影响

---

## 🎨 视觉设计优化

### 1. 色彩系统重构

**现状问题**：
- 当前调色板：`#0b0d10` (背景) → `#11151a` (卡片) → `#171c22` (次级卡片)
- 主色调：`#4cc8d9` (青色)，缺乏温度感
- 色阶跳跃过大，缺少微妙的层次过渡

**优化方案**：

```css
:root {
    /* 深色基底色阶 (7 levels) - 更细腻的深度层次 */
    --surface-0: #05070c;  /* L: 3% - 深空底色 */
    --surface-1: #0a0e14;  /* L: 5% - 应用背景 */
    --surface-2: #0f141c;  /* L: 7% - 主卡片 */
    --surface-3: #161d27;  /* L: 10% - 次级卡片 */
    --surface-4: #1e2636;  /* L: 13% - 悬停态 */
    --surface-5: #273345;  /* L: 17% - 激活态 */
    --surface-6: #2f3f54;  /* L: 22% - 边框/分割线 */

    /* 主色调：青蓝偏向 - 技术感与可信赖 */
    --primary-50: #e6f7fa;
    --primary-100: #b3e8f0;
    --primary-200: #80d9e6;
    --primary-300: #4dcadc;  /* 主色 */
    --primary-400: #38bdd2;
    --primary-500: #2aa9bf;
    --primary-600: #1f8a9e;
    --primary-700: #166b7d;
    --primary-800: #0f4c5c;
    --primary-900: #092d3b;

    /* 语义色彩 - 带透明度变体 */
    --success: #4cc38a;
    --success-soft: rgba(76, 195, 138, 0.12);
    --warning: #e9a568;  /* 降低饱和度 */
    --warning-soft: rgba(233, 165, 104, 0.12);
    --danger: #e07575;   /* 降低饱和度 */
    --danger-soft: rgba(224, 117, 117, 0.12);
    --info: #6ea8d9;
    --info-soft: rgba(110, 168, 217, 0.12);

    /* 文字层级 */
    --text-primary: #f0f4f7;      /* 主标题 */
    --text-secondary: #b8c5d0;    /* 正文 */
    --text-tertiary: #7a8b9a;     /* 辅助文字 */
    --text-quaternary: #4e5d6c;   /* 禁用/占位符 */

    /* 边框与分割 */
    --border-subtle: rgba(255, 255, 255, 0.06);
    --border-default: rgba(255, 255, 255, 0.1);
    --border-strong: rgba(255, 255, 255, 0.16);

    /* 阴影系统 - 带色调倾向 */
    --shadow-xs: 0 1px 2px rgba(5, 7, 12, 0.6);
    --shadow-sm: 0 2px 8px rgba(5, 7, 12, 0.5);
    --shadow-md: 0 8px 24px rgba(5, 7, 12, 0.45);
    --shadow-lg: 0 16px 48px rgba(5, 7, 12, 0.4);
    --shadow-xl: 0 24px 64px rgba(5, 7, 12, 0.35);

    /* 叠加光效 (用于卡片、按钮) */
    --glow-primary: 0 0 20px rgba(77, 202, 220, 0.15);
    --glow-success: 0 0 20px rgba(76, 195, 138, 0.15);
    --glow-danger: 0 0 20px rgba(224, 117, 117, 0.15);
}
```

**应用策略**：
- **背景层次**：`body { background: var(--surface-1); }`
- **卡片提升**：主卡片用 `--surface-2`，嵌套卡片用 `--surface-3`
- **交互反馈**：悬停 `--surface-4`，激活 `--surface-5`
- **去除纯黑/纯白**：所有阴影带深蓝色调倾向

---

### 2. 字体排版优化

**现状问题**：
- 已使用 Geist 字体（优秀选择），但排版细节不足
- 缺少流式字号系统
- 中英文混排行高不合理

**优化方案**：

```css
:root {
    /* 流式字号 (Fluid Typography) */
    --text-xs: clamp(0.75rem, 0.7rem + 0.15vw, 0.813rem);      /* 12-13px */
    --text-sm: clamp(0.875rem, 0.82rem + 0.18vw, 0.938rem);   /* 14-15px */
    --text-base: clamp(0.938rem, 0.88rem + 0.2vw, 1rem);      /* 15-16px */
    --text-lg: clamp(1.063rem, 0.98rem + 0.25vw, 1.125rem);   /* 17-18px */
    --text-xl: clamp(1.25rem, 1.1rem + 0.4vw, 1.375rem);      /* 20-22px */
    --text-2xl: clamp(1.5rem, 1.3rem + 0.6vw, 1.75rem);       /* 24-28px */
    --text-3xl: clamp(2rem, 1.7rem + 0.9vw, 2.5rem);          /* 32-40px */

    /* 字重 */
    --weight-regular: 400;
    --weight-medium: 500;
    --weight-semibold: 600;
    --weight-bold: 700;

    /* 行高 (中英文混排优化) */
    --leading-tight: 1.25;   /* 标题 */
    --leading-snug: 1.4;     /* 小段落 */
    --leading-normal: 1.6;   /* 正文 */
    --leading-relaxed: 1.75; /* 长文 */

    /* 字距 (Letter-spacing) */
    --tracking-tighter: -0.03em;  /* 大标题 */
    --tracking-tight: -0.015em;   /* 中标题 */
    --tracking-normal: 0;
    --tracking-wide: 0.02em;      /* 小标签/大写 */
}

/* 标题层级 */
h1, .heading-1 {
    font-size: var(--text-3xl);
    font-weight: var(--weight-bold);
    line-height: var(--leading-tight);
    letter-spacing: var(--tracking-tighter);
    color: var(--text-primary);
}

h2, .heading-2 {
    font-size: var(--text-2xl);
    font-weight: var(--weight-semibold);
    line-height: var(--leading-tight);
    letter-spacing: var(--tracking-tight);
    color: var(--text-primary);
}

h3, .heading-3 {
    font-size: var(--text-xl);
    font-weight: var(--weight-semibold);
    line-height: var(--leading-snug);
    color: var(--text-primary);
}

/* 正文 */
body {
    font-size: var(--text-base);
    line-height: var(--leading-normal);
    font-weight: var(--weight-regular);
}

/* 数字与代码 */
.mono, code, pre, .stat-val {
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;  /* 等宽数字 */
}

/* 平滑字体渲染 */
* {
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    text-rendering: optimizeLegibility;
}
```

**应用场景**：
- **页面标题 (h1)**：使用 `--text-3xl` 配合紧缩字距
- **卡片标题 (h3)**：使用 `--text-xl` 配合中等字重
- **指标数值 (.stat-val)**：使用等宽数字，避免跳动

---

### 3. 空间与节奏系统

**现状问题**：
- 间距使用魔法数字 (padding: 16px 等)
- 缺少统一的空间尺度

**优化方案**：

```css
:root {
    /* 8px 基准空间系统 */
    --space-0: 0;
    --space-1: 0.25rem;  /* 4px */
    --space-2: 0.5rem;   /* 8px */
    --space-3: 0.75rem;  /* 12px */
    --space-4: 1rem;     /* 16px */
    --space-5: 1.25rem;  /* 20px */
    --space-6: 1.5rem;   /* 24px */
    --space-8: 2rem;     /* 32px */
    --space-10: 2.5rem;  /* 40px */
    --space-12: 3rem;    /* 48px */
    --space-16: 4rem;    /* 64px */
    --space-20: 5rem;    /* 80px */

    /* 圆角 */
    --radius-xs: 4px;
    --radius-sm: 6px;
    --radius-md: 8px;
    --radius-lg: 12px;
    --radius-xl: 16px;
    --radius-2xl: 24px;
    --radius-full: 999px;  /* 胶囊按钮 */
}

/* 卡片内边距 */
.card {
    padding: var(--space-6);
    border-radius: var(--radius-lg);
}

.card-header {
    padding-bottom: var(--space-4);
    margin-bottom: var(--space-5);
}

/* 按钮内边距 */
.btn {
    padding: var(--space-3) var(--space-6);
    border-radius: var(--radius-md);
}

.btn-sm {
    padding: var(--space-2) var(--space-4);
    border-radius: var(--radius-sm);
}
```

---

### 4. 交互动效优化

**现状问题**：
- 缺少流畅的过渡动画
- 无微交互反馈

**优化方案**：

```css
:root {
    /* 缓动曲线 */
    --ease-out-expo: cubic-bezier(0.16, 1, 0.3, 1);
    --ease-in-out-circ: cubic-bezier(0.85, 0, 0.15, 1);
    --spring: cubic-bezier(0.34, 1.56, 0.64, 1);  /* 弹性 */

    /* 动画时长 */
    --duration-fast: 150ms;
    --duration-base: 200ms;
    --duration-slow: 300ms;
    --duration-slower: 500ms;
}

/* 通用过渡 */
* {
    transition-timing-function: var(--ease-out-expo);
}

/* 按钮交互 */
.btn {
    position: relative;
    transition: 
        background-color var(--duration-base),
        transform var(--duration-fast),
        box-shadow var(--duration-base);
}

.btn:hover {
    transform: translateY(-1px);
    box-shadow: var(--shadow-md), var(--glow-primary);
}

.btn:active {
    transform: translateY(0) scale(0.98);
    transition-duration: var(--duration-fast);
}

/* 卡片悬停 */
.card {
    transition: 
        background-color var(--duration-base),
        box-shadow var(--duration-base),
        transform var(--duration-base);
}

.interactive-card:hover {
    background: var(--surface-4);
    transform: translateY(-2px);
    box-shadow: var(--shadow-lg);
}

/* 加载骨架屏动画 */
@keyframes skeleton-shimmer {
    0% {
        background-position: -200% 0;
    }
    100% {
        background-position: 200% 0;
    }
}

.skeleton {
    background: linear-gradient(
        90deg,
        var(--surface-3) 0%,
        var(--surface-4) 50%,
        var(--surface-3) 100%
    );
    background-size: 200% 100%;
    animation: skeleton-shimmer 1.5s ease-in-out infinite;
}

/* Toast 淡入淡出 */
@keyframes toast-slide-in {
    from {
        opacity: 0;
        transform: translateY(var(--space-4)) scale(0.95);
    }
    to {
        opacity: 1;
        transform: translateY(0) scale(1);
    }
}

#toast {
    animation: toast-slide-in var(--duration-base) var(--spring);
}

/* 侧边栏抽屉滑入 */
@keyframes drawer-slide-in {
    from {
        transform: translateX(100%);
    }
    to {
        transform: translateX(0);
    }
}

.editor-drawer {
    animation: drawer-slide-in var(--duration-slow) var(--ease-out-expo);
}

/* 页面切换淡入 */
@keyframes fade-in {
    from {
        opacity: 0;
        transform: translateY(var(--space-3));
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

.view-page.active {
    animation: fade-in var(--duration-base) var(--ease-out-expo);
}
```

---

## 🔧 组件级优化

### 1. Dashboard 仪表盘优化

**KPI 指标卡片增强**：

```css
.stat-card {
    background: var(--surface-2);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    transition: all var(--duration-base);
    position: relative;
    overflow: hidden;
}

/* 微妙的渐变背景 */
.stat-card::before {
    content: '';
    position: absolute;
    top: 0;
    right: 0;
    width: 40%;
    height: 100%;
    background: radial-gradient(
        ellipse at top right,
        rgba(77, 202, 220, 0.05),
        transparent 70%
    );
    pointer-events: none;
}

.stat-card:hover {
    background: var(--surface-3);
    border-color: var(--border-default);
    transform: translateY(-2px);
    box-shadow: var(--shadow-md);
}

/* 指标数值 - 大号数字，表格数字对齐 */
.stat-val {
    font-size: clamp(1.75rem, 1.5rem + 0.8vw, 2.25rem);
    font-weight: var(--weight-bold);
    font-variant-numeric: tabular-nums;
    color: var(--text-primary);
    margin: var(--space-2) 0;
    line-height: 1.1;
}

/* 图标 - 降低对比度 */
.stat-icon {
    width: 20px;
    height: 20px;
    opacity: 0.4;
    stroke: var(--primary-300);
}
```

**实时日志终端优化**：

```css
.terminal-box {
    background: var(--surface-0);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: var(--space-4);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    line-height: var(--leading-relaxed);
    max-height: 400px;
    overflow-y: auto;
    color: var(--text-secondary);
}

/* 自定义滚动条 */
.terminal-box::-webkit-scrollbar {
    width: 8px;
}

.terminal-box::-webkit-scrollbar-track {
    background: transparent;
}

.terminal-box::-webkit-scrollbar-thumb {
    background: var(--surface-4);
    border-radius: var(--radius-full);
}

.terminal-box::-webkit-scrollbar-thumb:hover {
    background: var(--surface-5);
}

/* 日志行 - 交替背景 */
.log-line {
    padding: var(--space-1) 0;
    border-bottom: 1px solid rgba(255, 255, 255, 0.02);
}

.log-line:hover {
    background: rgba(255, 255, 255, 0.02);
}

/* 日志时间戳 */
.log-time {
    color: var(--text-tertiary);
    margin-right: var(--space-3);
    font-variant-numeric: tabular-nums;
}
```

---

### 2. 节点表格优化

**现状问题**：
- 表格信息密度高，但视觉层次不清晰
- 缺少行悬停与选中态反馈

**优化方案**：

```css
/* 表格容器 */
.table-wrap {
    background: var(--surface-2);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    overflow: hidden;
}

table {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--text-sm);
}

/* 表头 */
thead {
    background: var(--surface-3);
    border-bottom: 2px solid var(--border-strong);
}

thead th {
    padding: var(--space-4) var(--space-5);
    text-align: left;
    font-weight: var(--weight-semibold);
    font-size: var(--text-xs);
    text-transform: uppercase;
    letter-spacing: var(--tracking-wide);
    color: var(--text-tertiary);
}

/* 表格行 */
tbody tr {
    border-bottom: 1px solid var(--border-subtle);
    transition: background-color var(--duration-fast);
}

tbody tr:hover {
    background: var(--surface-3);
}

tbody td {
    padding: var(--space-4) var(--space-5);
    color: var(--text-secondary);
}

/* 节点 IP - 等宽字体 */
.node-ip {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    color: var(--primary-300);
}

/* 延迟徽章 */
.badge-latency {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-full);
    font-size: var(--text-xs);
    font-weight: var(--weight-medium);
    font-variant-numeric: tabular-nums;
}

.badge-latency.fast {
    background: var(--success-soft);
    color: var(--success);
}

.badge-latency.medium {
    background: var(--warning-soft);
    color: var(--warning);
}

.badge-latency.slow {
    background: var(--danger-soft);
    color: var(--danger);
}

/* 收藏按钮 */
.btn-favorite {
    background: transparent;
    border: none;
    cursor: pointer;
    font-size: 18px;
    transition: transform var(--duration-fast);
}

.btn-favorite:hover {
    transform: scale(1.2);
}

.btn-favorite.active {
    color: var(--warning);
    filter: drop-shadow(0 0 6px rgba(233, 165, 104, 0.4));
}
```

---

### 3. 按钮系统重构

**优化方案**：

```css
/* 基础按钮 */
.btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-6);
    border: none;
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    line-height: 1;
    cursor: pointer;
    transition: all var(--duration-base);
    white-space: nowrap;
}

/* 主要按钮 */
.btn-primary {
    background: var(--primary-300);
    color: var(--surface-0);
}

.btn-primary:hover {
    background: var(--primary-400);
    box-shadow: var(--shadow-sm), var(--glow-primary);
    transform: translateY(-1px);
}

.btn-primary:active {
    transform: scale(0.98);
}

/* 轮廓按钮 */
.btn-outline {
    background: transparent;
    border: 1px solid var(--border-default);
    color: var(--text-secondary);
}

.btn-outline:hover {
    background: var(--surface-3);
    border-color: var(--border-strong);
}

/* 危险按钮 */
.btn-danger {
    background: var(--danger);
    color: white;
}

.btn-danger:hover {
    background: #e98b8b;
    box-shadow: var(--shadow-sm), var(--glow-danger);
}

/* 按钮尺寸 */
.btn-xs {
    padding: var(--space-1) var(--space-3);
    font-size: var(--text-xs);
}

.btn-sm {
    padding: var(--space-2) var(--space-4);
    font-size: var(--text-sm);
}

.btn-lg {
    padding: var(--space-4) var(--space-8);
    font-size: var(--text-lg);
}

/* 图标按钮 */
.btn-icon {
    width: 36px;
    height: 36px;
    padding: 0;
    border-radius: var(--radius-md);
}

/* 禁用状态 */
.btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    transform: none !important;
}
```

---

## 📐 布局优化

### 1. 侧边栏优化

```css
.sidebar {
    width: 260px;
    background: var(--surface-2);
    border-right: 1px solid var(--border-subtle);
    display: flex;
    flex-direction: column;
    height: 100vh;
    position: sticky;
    top: 0;
    z-index: var(--z-sidebar);
}

/* Logo 区域 */
.sidebar-brand {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-6);
    border-bottom: 1px solid var(--border-subtle);
}

.logo-icon {
    width: 40px;
    height: 40px;
    background: var(--primary-300);
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
}

.logo-icon svg {
    width: 24px;
    height: 24px;
    fill: var(--surface-0);
}

.brand-title {
    font-size: var(--text-lg);
    font-weight: var(--weight-bold);
    color: var(--text-primary);
}

.brand-sub {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
}

/* 导航列表 */
.nav-list {
    flex: 1;
    padding: var(--space-4) var(--space-3);
    overflow-y: auto;
}

.nav-item {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
    padding: var(--space-3) var(--space-4);
    border: none;
    border-radius: var(--radius-md);
    background: transparent;
    color: var(--text-secondary);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    text-align: left;
    cursor: pointer;
    transition: all var(--duration-fast);
    margin-bottom: var(--space-1);
}

.nav-item svg {
    width: 20px;
    height: 20px;
    opacity: 0.6;
    transition: opacity var(--duration-fast);
}

.nav-item:hover {
    background: var(--surface-3);
    color: var(--text-primary);
}

.nav-item:hover svg {
    opacity: 1;
}

.nav-item.active {
    background: var(--primary-soft);
    color: var(--primary-300);
    font-weight: var(--weight-semibold);
}

.nav-item.active svg {
    opacity: 1;
    stroke: var(--primary-300);
}

/* 侧边栏底部 */
.sidebar-footer {
    padding: var(--space-4);
    border-top: 1px solid var(--border-subtle);
}
```

---

### 2. 响应式网格系统

```css
/* Dashboard 2列布局 */
.dashboard-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(500px, 1fr));
    gap: var(--space-6);
    margin-top: var(--space-6);
}

/* KPI 指标 4列网格 */
.stat-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: var(--space-4);
}

/* 节点卡片自适应网格 */
.sb-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: var(--space-4);
}

/* 响应式断点 */
@media (max-width: 1024px) {
    .dashboard-grid {
        grid-template-columns: 1fr;
    }
}

@media (max-width: 768px) {
    .stat-grid {
        grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
    }
}
```

---

## 🎭 细节打磨

### 1. 微交互细节

```css
/* Glassmorphism 毛玻璃效果（节制使用） */
.glass-card {
    background: rgba(255, 255, 255, 0.03);
    backdrop-filter: blur(20px) saturate(180%);
    border: 1px solid rgba(255, 255, 255, 0.08);
    box-shadow: 
        inset 0 1px 0 rgba(255, 255, 255, 0.05),
        var(--shadow-md);
}

/* 聚光灯边框效果 */
.spotlight-border {
    position: relative;
    border: 1px solid var(--border-default);
}

.spotlight-border::before {
    content: '';
    position: absolute;
    inset: -1px;
    border-radius: inherit;
    padding: 1px;
    background: linear-gradient(
        135deg,
        transparent 30%,
        var(--primary-300) 50%,
        transparent 70%
    );
    -webkit-mask: 
        linear-gradient(#fff 0 0) content-box, 
        linear-gradient(#fff 0 0);
    -webkit-mask-composite: xor;
    mask-composite: exclude;
    opacity: 0;
    transition: opacity var(--duration-base);
}

.spotlight-border:hover::before {
    opacity: 0.6;
}

/* 数据标签闪烁动画（新数据到达） */
@keyframes data-pulse {
    0%, 100% {
        opacity: 1;
    }
    50% {
        opacity: 0.6;
    }
}

.data-updated {
    animation: data-pulse 600ms ease-in-out;
}

/* 徽章光晕 */
.badge-glow {
    box-shadow: 0 0 16px currentColor;
}
```

---

### 2. 性能优化技巧

```css
/* GPU 加速动画 */
.gpu-accelerated {
    transform: translateZ(0);
    will-change: transform;
}

/* 减少重绘 */
.no-reflow {
    contain: layout style paint;
}

/* 滚动性能优化 */
.smooth-scroll-container {
    overflow-y: auto;
    scroll-behavior: smooth;
    -webkit-overflow-scrolling: touch;
}

/* 图片懒加载占位 */
.img-skeleton {
    background: linear-gradient(
        90deg,
        var(--surface-3) 0%,
        var(--surface-4) 50%,
        var(--surface-3) 100%
    );
    background-size: 200% 100%;
    animation: skeleton-shimmer 1.5s ease-in-out infinite;
}
```

---

## 📱 移动端优化

### 1. 底部导航栏增强

```css
.bottom-nav {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    display: none;
    grid-template-columns: repeat(5, 1fr);
    background: var(--surface-2);
    border-top: 1px solid var(--border-subtle);
    padding: env(safe-area-inset-bottom, var(--space-2)) var(--space-2) var(--space-2);
    z-index: var(--z-sidebar);
}

@media (max-width: 768px) {
    .bottom-nav {
        display: grid;
    }

    .sidebar {
        display: none;
    }

    .main-content {
        margin-bottom: calc(56px + env(safe-area-inset-bottom, 0));
    }
}

.bottom-nav-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-1);
    padding: var(--space-2);
    background: transparent;
    border: none;
    color: var(--text-tertiary);
    font-size: var(--text-xs);
    cursor: pointer;
    transition: color var(--duration-fast);
}

.bottom-nav-item svg {
    width: 22px;
    height: 22px;
    stroke-width: 2;
}

.bottom-nav-item.active {
    color: var(--primary-300);
}
```

---

### 2. 触控优化

```css
/* 增大触控目标 */
@media (max-width: 768px) {
    .btn {
        min-height: 44px;  /* iOS 建议最小触控尺寸 */
    }

    .nav-item {
        min-height: 48px;
    }

    /* 表格横向滚动 */
    .table-wrap {
        overflow-x: auto;
        -webkit-overflow-scrolling: touch;
    }

    table {
        min-width: 800px;  /* 防止压缩 */
    }
}

/* 禁用双击缩放（保留单击） */
button, a {
    touch-action: manipulation;
}
```

---

## 🚀 实施计划

### 阶段一：色彩与排版基础 (Week 1)
1. **替换 CSS 变量**：更新 `:root` 中的色彩系统与字体系统
2. **测试对比度**：确保 WCAG AA 级别对比度合规
3. **验证效果**：在各个页面检查新色彩表现

### 阶段二：组件级重构 (Week 2-3)
1. **按钮与表单组件**：重写按钮、输入框、选择器样式
2. **卡片与表格**：优化 Dashboard 卡片、节点表格
3. **导航与布局**：增强侧边栏、底部导航栏

### 阶段三：动画与交互 (Week 4)
1. **添加过渡动画**：按钮悬停、卡片展开、页面切换
2. **微交互反馈**：加载状态、骨架屏、Toast 提示
3. **性能测试**：确保动画帧率稳定在 60fps

### 阶段四：响应式与适配 (Week 5)
1. **移动端测试**：在 iOS Safari、Android Chrome 测试
2. **触控优化**：调整触控目标尺寸、手势支持
3. **跨浏览器验证**：Firefox、Edge、Safari 兼容性测试

### 阶段五：无障碍与打磨 (Week 6)
1. **键盘导航**：确保 Tab 顺序合理、快捷键支持
2. **屏幕阅读器**：ARIA 标签完善、角色定义
3. **最终打磨**：细节调整、代码精简、文档完善

---

## 📊 预期效果

### 视觉提升
- **专业感 +40%**：深邃的色彩层次 + 精细的排版细节
- **可读性 +30%**：流式字号 + 优化行高 + 中文字体适配
- **现代感 +50%**：微妙动画 + 玻璃态 + 光晕效果

### 性能指标
- **FCP (First Contentful Paint)**: < 1.2s
- **TTI (Time to Interactive)**: < 2.5s
- **动画帧率**: 稳定 60fps
- **Lighthouse 评分**: 90+ (Performance / Accessibility)

### 用户体验
- **交互反馈延迟**: < 100ms
- **页面切换流畅度**: 无卡顿
- **移动端适配**: 100% 触控友好

---

## 🎓 参考资源

### 设计系统参考
- **Tailwind CSS** - 色彩与空间系统
- **Radix UI** - 组件交互模式
- **Linear** - 深色主题专业感
- **Vercel Dashboard** - 技术工具界面范式

### 技术文档
- [MDN CSS](https://developer.mozilla.org/en-US/docs/Web/CSS)
- [WCAG 2.1 无障碍指南](https://www.w3.org/WAI/WCAG21/quickref/)
- [Web.dev 性能优化](https://web.dev/fast/)

---

## 📝 附录：快速参考

### CSS 变量速查表

```css
/* 色彩 */
var(--surface-0)      /* 深空底色 */
var(--surface-2)      /* 主卡片 */
var(--primary-300)    /* 主色 */
var(--text-primary)   /* 主标题 */

/* 字体 */
var(--text-xl)        /* 卡片标题 */
var(--weight-semibold)/* 中等粗细 */
var(--leading-normal) /* 正文行高 */

/* 空间 */
var(--space-4)        /* 16px */
var(--space-6)        /* 24px */
var(--radius-lg)      /* 12px 圆角 */

/* 动画 */
var(--duration-base)  /* 200ms */
var(--ease-out-expo)  /* 缓动曲线 */
var(--shadow-md)      /* 中等阴影 */
```

---

**最后更新**: 2026-09-15  
**版本**: v1.0  
**维护者**: AimiliVPN 开发团队
