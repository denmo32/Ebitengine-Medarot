●概要

メダロットの戦闘システムのようなもの。

Ebitengine製。

2025/06/24: 射撃、格闘攻撃を実装し、簡易なロボトルを実装。バランス滅茶苦茶。



●ファイル構成と各ファイルの役割

1. アプリケーションの初期設定と起動

    main.go: プログラムのエントリーポイント。フォントやゲームデータ、設定を読み込み、ゲームウィンドウを作成してebiten.RunGameでゲームループを開始します。
   
    game.go: ゲームの心臓部。ECSのワールドと全システムを保持し、メインのUpdate/Drawループを管理します。ゲームの状態遷移（例：プレイ中→ゲームオーバー）に応じたシステムの呼び出し分けもここで行います。
   
    config.go: ゲームの静的な設定値（画面サイズ、UIレイアウト、色の定義、ゲームバランスなど）を管理します。
   
    models.go: enumのような、プロジェクト全体で使われる基本的な型定義（PartSlotKey, MedarotState, TeamIDなど）を管理します。
   


2. データとエンティティの管理

    csv_loader.go: medals.csvやparts.csvといった外部ファイルを読み込み、Goの構造体に変換します。
   
    medarot_initializer.go: csv_loaderで読み込んだデータとconfigを基に、メダロットのエンティティを生成し、各種コンポーネントをアタッチして初期化します。
   
    components.go: ECSの「魂」です。エンティティが持つデータ（IdentityComponent, StatusComponentなど）を全てここで定義します。コンポーネントに紐づくヘルパーメソッド（IsBroken()など）もここに含めます。



3. システム（ロジック）

    player_input_system.go: プレイヤーのキーボードやマウス入力を検知し、UI操作や行動選択のキューイングを行います。
   
    ai_system.go: AIが制御するエンティティの行動（どのパーツを、どのターゲットに使うか）を決定します。
   
    gauge_update_system.go: 全てのメダロットのゲージ（チャージ、クールダウン）を更新し、状態遷移（例：ActionCharging -> ReadyToExecuteAction）をトリガーします。
   
    action_execution_system.go: ReadyToExecuteActionタグが付いたエンティティのアクション（攻撃など）を実際に実行します。
   
    game_rule_system.go: 勝敗条件（リーダー機の破壊など）を毎フレームチェックし、ゲームの終了を判定します。
   
    message_system.go: メッセージ表示中にクリックを待ち、コールバックを実行したり、ゲーム状態を次に進めたりします。
   
    render_system.go: ECSのデータを基に、全ての描画処理を行います。



4. ユーティリティ（補助関数）

    action_utils.go: 戦闘ロジックの補助関数。命中計算、ダメージ計算、ターゲット選択など、action_execution_system.goから呼び出される複雑な計算をここにまとめます。
   
    ui_draw.go: 描画の補助関数。ウィンドウ、ボタン、情報パネルといった再利用可能なUIパーツの描画ロジックをここに集約します。
