# App Runner icon

Source of truth: `build/icon.svg`（256 网格，1024 主图由它直接渲染）。

Palette: 砂岩蓝 —— 砂岩底 `#f6f4ee`、石板蓝 `#2f6491`（渐变 `#4a83b4 → #1c3f5e`）、骨白 `#fbf6e9`、浅蓝 `#a9d0ec`。

Design: 石板蓝渐变圆角方块 + 骨白 `❯` 提示符 + 浅蓝光标块。无文字、无边框，圆角外透明；16 px 下仍能读出「提示符 + 光标」。

生成物：

| 文件 | 说明 |
| --- | --- |
| `build/appicon.png` | 1024×1024，Wails 应用与托盘图标 |
| `build/windows/icon.ico` | 256/128/64/48/32/16 |
| `frontend/public/appicon.png` | 64×64 favicon |

重新生成：改 `build/icon.svg` 后用任意 SVG 栅格化工具输出 1024 PNG 覆盖 `build/appicon.png`，再执行 `wails3 task generate:icons`（即 `wails3 generate icons -input appicon.png -windowsfilename windows/icon.ico`）；favicon 另导出 64×64 到 `frontend/public/appicon.png`。
