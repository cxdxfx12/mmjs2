# Changelog

## Unreleased - 2026-07-05

### Fixed
- 活动规则：前端使用 `el-date-picker`，并在保存前校验开始/结束日期，结束日期不可早于开始日期。（文件：`frontend/src/views/Rules.vue`）
- 后端规则保存增加校验：`campaign` 类型要求 `campaign_name`、`campaign_start`、`campaign_end`；结束日期不得早于开始日期，并将日期规范为 `YYYY-MM-DD`。（文件：`internal/rules/repository.go`）
- 前端保存规则时若后端返回无效 `id`（0）则视为失败，UI 将显示错误信息而非误报成功。（文件：`frontend/src/stores/app.ts`）

### Changed
- 调整规则编辑弹窗布局：对话框宽度和表单标签宽度增大，允许标签换行，减少遮挡。（文件：`frontend/src/views/Rules.vue`）
- 在“拉均重/偏差加价配置”中将“计算逻辑”以红色突出显示，便于用户注意。（文件：`frontend/src/views/Rules.vue`）

### Notes
- 已验证前端构建与后端 `go build` 均通过。前端构建产生 chunk 警告（体积较大），非错误。
