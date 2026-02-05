# 校级程序设计竞赛动态 Rating 系统

一个为高校程序设计竞赛设计的动态 Rating 系统，采用 Growth-Focused Rating (GFR) 算法，支持 Hydro OJ 格式 CSV 导入。

## 功能特性

- **动态 Rating 计算**：基于 GFR 算法，支持涨分与扣分
- **CSV 批量导入**：支持 Hydro OJ 导出格式，提供示例CSV下载
- **段位系统**：Codeforces 风格段位（Newbie → Grandmaster）
- **全量重算**：修改参数后可重新计算历史 Rating
- **可视化统计**：年级分布、段位分布、Rating 曲线
- **管理后台**：比赛管理、学生管理、参数调优、段位配置、网站设置
- **学生查询**：主页支持按学号/姓名/班级/邮箱搜索学生
- **主题定制**：支持自定义主题色和网站标题
- **安全防护**：JWT 登录、登录限流、基础安全响应头

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.21+ / Gin / GORM / SQLite |
| 前端 | React 19 / TypeScript / Vite / Ant Design / Recharts |
| 状态管理 | Zustand |
| 路由 | React Router v7 |

## 项目结构

```
rating/
├── backend/
│   ├── cmd/server/          # 入口
│   ├── internal/
│   │   ├── config/          # 配置加载
│   │   ├── database/        # 数据库初始化
│   │   ├── handlers/        # HTTP 处理器
│   │   ├── models/          # 数据模型
│   │   └── services/
│   │       ├── csv/         # CSV 解析器
│   │       └── rating/      # Rating 算法
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── api/             # API 客户端
│   │   ├── components/      # 公共组件
│   │   ├── pages/           # 页面
│   │   └── stores/          # Zustand 状态
│   └── vite.config.ts
└── data/                    # SQLite 数据库
```

## 快速开始

### 1. 启动后端

```bash
cd backend
go mod tidy
go run cmd/server/main.go
```

默认监听 `http://localhost:8080`

### 2. 启动前端（开发模式）

```bash
cd frontend
npm install
npm run dev
```

默认监听 `http://localhost:5173`，自动代理 `/api` 到后端。

### 3. 构建前端

```bash
cd frontend
npm run build
```

产物在 `frontend/dist/`。

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVER_PORT` | `8080` | 后端端口 |
| `GIN_MODE` | `debug` | Gin 模式 (debug/release) |
| `DB_PATH` | `../data/rating.db` | SQLite 路径 |
| `ADMIN_PASSWORD` | `admin123456` | 管理员密码（必须为 bcrypt 哈希） |
| `JWT_SECRET` | `rating-system-secret-key` | Token 签名密钥 |
| `TOKEN_EXP` | `24` | Token 过期时间（小时） |

## API 接口

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/leaderboard` | 排行榜（支持 `?grade=` 筛选） |
| GET | `/api/v1/students` | 学生查询（`?query=` 支持学号/姓名/班级/邮箱） |
| GET | `/api/v1/students/:id` | 学生详情 |
| GET | `/api/v1/students/:id/history` | Rating 历史 |
| GET | `/api/v1/contests` | 比赛列表 |
| GET | `/api/v1/tiers` | 段位配置 |
| GET | `/api/v1/statistics` | 统计数据 |
| GET | `/api/v1/site-config` | 网站配置 |

### 管理接口（需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/admin/login` | 登录 |
| GET | `/api/v1/admin/config` | 获取算法配置 |
| PUT | `/api/v1/admin/config` | 更新算法配置 |
| POST | `/api/v1/admin/contests` | 上传比赛 CSV |
| PUT | `/api/v1/admin/contests/:id` | 编辑比赛 |
| DELETE | `/api/v1/admin/contests/:id` | 删除比赛 |
| GET | `/api/v1/admin/students` | 学生列表 |
| POST | `/api/v1/admin/students` | 新增学生 |
| PUT | `/api/v1/admin/students/:id` | 编辑学生 |
| DELETE | `/api/v1/admin/students/:id` | 删除学生 |
| PUT | `/api/v1/admin/tiers` | 更新段位配置 |
| POST | `/api/v1/admin/replay/apply` | 触发全量重算 |
| POST | `/api/v1/admin/replay/preview` | 预览算法效果 |
| GET | `/api/v1/admin/replay/status` | 重算状态 |
| PUT | `/api/v1/admin/site-config` | 更新网站配置 |

## CSV 格式

支持 Hydro OJ 导出格式。管理后台提供 **"下载示例CSV"** 按钮获取模板。

```csv
#,用户,电子邮件,学校,名称,学号,"解决
总耗时",#1 A题,#1 罚时
1,user1,user1@example.com,计算机学院,张三,2301001,"5
01:23:45",✓,10
2,user2,user2@example.com,软件学院,李四,2302002,"4
02:00:00",✓,20
```

| 列 | 说明 |
|-----|------|
| # | 排名 |
| 用户 | 用户名 |
| 电子邮件 | 邮箱 |
| 学校 | 班级/学院 |
| 名称 | 姓名 |
| 学号 | 学号（前2位解析年级） |
| 解决/总耗时 | 解题数和总时间（多行字段） |
| 后续列 | 每题状态和罚时（可选） |

**注意**：0 分选手会被自动过滤。

## Rating 算法

### Growth-Focused Rating (GFR)

专为大规模比赛（500+人）设计，让更多人获得正向激励。

**Rating 变化由四部分组成**：

```
Rating变化 = (参与奖励 + 排名奖励 + 技能差异 + Top奖励) × 阻尼系数
```

| 组成部分 | 说明 |
|---------|------|
| 参与奖励 | 所有参赛者都能获得的基础分 |
| 排名奖励 | 根据排名百分位获得，使用可调曲线 |
| 技能差异 | 超出期望排名时的额外加分（基于Elo） |
| Top奖励 | 前X%选手的额外加成 |

**核心特性**：
- **可涨可跌**：表现低于预期会产生负增量
- **参与分**：所有参赛者都能获得参与奖励
- **可调曲线**：灵活控制分数分布

### 算法参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `k_factor` | 48 | K 因子（影响技能差异幅度） |
| `growth_inertia` | 0.8 | 成长阻尼（0~2，对总变化缩放） |
| `initial_rating` | 0 | 初始 Rating |
| `participation_bonus` | 5 | 参与奖励（所有人获得） |
| `rank_bonus_max` | 30 | 排名奖励上限 |
| `rank_bonus_curve` | 1.5 | 排名奖励曲线（>1更平缓，<1更陡峭） |
| `top_bonus_threshold` | 0.1 | Top奖励阈值（前10%） |
| `top_bonus_multiplier` | 0.5 | Top奖励乘数 |

### 效果预估（500人参赛，默认参数）

| 排名 | 大约涨分 |
|------|---------|
| 最后一名 | +4分 |
| 第250名 | +15分 |
| 前10% | +40分以上 |
| 第1名 | +60分以上 |

## 段位配置

默认 Codeforces 风格：

| 段位 | Rating 区间 | 颜色 |
|------|-------------|------|
| Newbie | 0 - 1199 | 灰色 |
| Pupil | 1200 - 1399 | 绿色 |
| Specialist | 1400 - 1599 | 青色 |
| Expert | 1600 - 1899 | 蓝色 |
| Candidate Master | 1900 - 2099 | 紫色 |
| Master | 2100 - 2399 | 橙色 |
| Grandmaster | 2400+ | 红色 |

## 数据模型

### Student

| 字段 | 类型 | 说明 |
|------|------|------|
| student_id | string | 学号或用户名（主键） |
| name | string | 姓名 |
| grade | int | 年级（从学号解析） |
| current_rating | float64 | 当前 Rating |
| max_rating | float64 | 历史最高 Rating |
| match_count | int | 参赛次数 |

### Contest

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| name | string | 比赛名称 |
| date | time | 比赛日期 |
| weight | float64 | 权重（影响 K 值） |
| participant_count | int | 参赛人数 |

### Result

| 字段 | 类型 | 说明 |
|------|------|------|
| student_id | string | 学号 |
| contest_id | uint | 比赛 ID |
| rank | int | 排名 |
| solved | int | 解题数 |
| performance | float64 | 表现分 |
| rating_before | float64 | 赛前 Rating |
| rating_after | float64 | 赛后 Rating |
| delta | float64 | 变化值 |

## 使用流程

1. **登录管理后台**：使用管理员密码登录 `/admin/login`
2. **下载示例CSV**：点击"下载示例CSV"了解格式
3. **上传比赛**：上传 Hydro OJ 导出的 CSV 文件
4. **调整参数**：在"算法实验室"调整参数并预览效果
5. **执行重算**：在"算法实验室"触发全量重算
6. **查看排行榜**：公开页面查看 Rating 排名和统计

## 注意事项

- 修改算法参数后需点击"全量重算"使新参数生效
- 删除比赛后需重算以更新 Rating
- 学号前2位用于解析入学年级（如 23 表示 2023 级）
- 生产环境务必修改 `ADMIN_PASSWORD` 和 `JWT_SECRET`
- `ADMIN_PASSWORD` 需使用 bcrypt 哈希（例如 `$2a$...` 或 `bcrypt:` 前缀）

生成 bcrypt 哈希示例（在 `backend` 目录执行）：

```bash
go run golang.org/x/crypto/bcrypt@latest admin123456
```
