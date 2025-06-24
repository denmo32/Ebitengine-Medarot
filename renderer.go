package main

// このファイルは、RenderSystem (systems.go内) に描画ロジックが集約されたため、
// 以前の個別の描画関数 (drawBattlefield, drawMedarotIcons など) は削除されました。

// MplusFont のような共有リソースや、汎用的な低レベル描画ヘルパー関数が
// 必要であればここに残すことができますが、現状では RenderSystem が
// 必要な描画処理を Config や ECS World から取得したデータに基づいて行います。

// ui_components.go にある DrawWindow, DrawButton, DrawMessagePanel などは
// RenderSystem から引き続き利用されます。
