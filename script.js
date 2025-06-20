// 1. 中央集権的な設定 (Centralized configuration)
const CONFIG = {
    MAX_GAUGE: 100, // ゲージの最大値 (Maximum gauge value)
    UPDATE_INTERVAL: 50, // 更新間隔（ミリ秒） (Update interval in milliseconds)
    PLAYERS_PER_TEAM: 3, // チームあたりのメダロット数 (Number of Medarots per team)
    PART_HP_BASE: 50, // パーツHPの基本値 (Base HP for parts)
    LEGS_HP_BONUS: 10, // 脚部パーツHPボーナス (Legs part HP bonus)
    BASE_DAMAGE: 20, // 基本ダメージ (Base damage)
    TEAMS: {
        team1: { name: 'Team 1', color: '#63b3ed', baseSpeed: 1.0, textColor: 'text-blue-300' },
        team2: { name: 'Team 2', color: '#f56565', baseSpeed: 0.9, textColor: 'text-red-300' }
    }
};

// メダルクラス (Medal Class)
// メダロットのコアとなるメダルの定義 (Defines the core medal of a Medarot)
class Medal {
    constructor(name, personality, medaforce, attribute, skillLevels) {
        this.name = name; // メダル名 (Medal name) e.g., "Kabuto", "Kuwagata"
        this.personality = personality; // 性格 (Personality) e.g., "Speed"
        this.medaforce = medaforce; // メダフォース (Medaforce) e.g., "Berserk"
        this.attribute = attribute; // 属性 (Attribute) e.g., "Fire"
        this.skillLevels = skillLevels; // 熟練度 (Skill Levels) e.g., { shoot: 0, fight: 0, scan: 0, support: 0 }
    }
}

// メダロットクラス (Medarot Class)
// 以前はPlayerクラスでした (Formerly Player class)
// メダロットの戦闘単位を管理します (Manages a Medarot combat unit)
class Medarot {
    // コンストラクタ (Constructor)
    // id: メダロットの一意なID (Unique ID for the Medarot)
    // name: メダロットの名前 (Name of the Medarot)
    // team: 所属チーム (Team affiliation)
    // speed: メダロットのスピード (Speed of the Medarot)
    // medal: メダロットのメダル (Medal for the Medarot)
    // options: 追加オプション (Additional options, e.g., isLeader)
    constructor(id, name, team, speed, medal, options) {
        this.id = id; // ID (ID)
        this.name = name; // 名前 (Name)
        this.team = team; // チーム (Team)
        this.speed = speed; // スピード (Speed)
        this.medal = medal; // メダル (Medal)
        this.isLeader = options.isLeader; // リーダーかどうか (Is it a leader?)
        this.color = options.color; // チームカラー (Team color)
        this.iconElement = null; // DOM上のアイコン要素 (Icon element on DOM)
        this.partDOMElements = {}; // パーツのDOM要素への参照 (References to part DOM elements) // 3. Cache DOM references

        this.currentActionCharge = null; // 現在選択中アクションのチャージ目標値 (Target charge value for the currently selected action)
        this.currentActionCooldown = null; // 現在選択中アクションのクールダウン目標値 (Target cooldown value for the currently selected action)
        this.fullReset(); // 初期状態にリセット (Reset to initial state)
    }

    // --- 状態管理 (State Management) ---

    // 完全リセット (Full Reset)
    // メダロットの状態とパーツを初期化します (Initializes Medarot's state and parts)
    fullReset() {
        this.gauge = 0; // 現在のゲージ (Current gauge)
        // 初期状態を 'idle_charging' に変更 (Change initial state to 'idle_charging')
        // charging -> idle_charging (初期チャージ)
        // selected_charging -> action_charging (アクション選択後のチャージ)
        // cooldown_complete は ready_select に統合、または action_cooldown 完了時に ready_select に遷移
        // 'charging' (startCooldown後) -> 'action_cooldown'
        this.state = 'idle_charging'; // 現在の状態 (Current state): idle_charging, action_charging, action_cooldown, ready_select, ready_execute, broken
        this.selectedActionType = null; // 選択されたアクションタイプ (Selected action type)
        this.selectedPartKey = null; // 選択されたパーツキー (Selected part key)
        this.preparedAttack = null; // 準備された攻撃情報 (Prepared attack information)
        this.currentActionCharge = null; // アクションチャージ値をリセット (Reset action charge value)
        this.currentActionCooldown = null; // アクションクールダウン値をリセット (Reset action cooldown value)

        const hp = CONFIG.PART_HP_BASE;
        const legsHp = hp + CONFIG.LEGS_HP_BONUS;
        // 各パーツの定義 (Definition of each part) - charge/cooldown値を調整
        this.parts = {
            head: { name: 'Head', hp, maxHp: hp, action: 'Scan', isBroken: false, charge: 60, cooldown: 80 }, // 頭部パーツ (Head part)
            rightArm: { name: 'Right Arm', hp, maxHp: hp, action: 'Shoot', isBroken: false, charge: 75, cooldown: 100 }, // 右腕パーツ (Right arm part)
            leftArm: { name: 'Left Arm', hp, maxHp: hp, action: 'Fight', isBroken: false, charge: 75, cooldown: 100 }, // 左腕パーツ (Left arm part)
            legs: { name: 'Legs', hp: legsHp, maxHp: legsHp, action: 'Move', isBroken: false } // 脚部パーツ (Legs part) - チャージ/クールダウンなし (No charge/cooldown)
        };
    }

    // アクション後のクールダウン開始 (Start Cooldown after action execution)
    // アクション実行後のクールダウン状態に移行します (Transitions to cooldown state after action execution)
    startCooldown() {
        this.gauge = 0; // ゲージをリセット (Reset gauge)
        this.state = 'action_cooldown'; // アクションクールダウン状態へ (To action_cooldown state)
        this.selectedActionType = null; // 選択アクションをクリア (Clear selected action)
        this.selectedPartKey = null; // 選択パーツをクリア (Clear selected part)
        this.preparedAttack = null; // 準備された攻撃をクリア (Clear prepared attack)
        // this.currentActionCooldown は selectAction で設定されたものが使われる (this.currentActionCooldown set in selectAction will be used)
    }

    // アクション選択 (Select Action)
    // partKey: 選択するパーツのキー (Key of the part to select)
    // 選択されたパーツでアクションを準備し、アクションチャージング状態に移行します (Prepares action with the selected part and transitions to action charging state)
    selectAction(partKey) {
        this.selectedPartKey = partKey; // 選択パーツキーを保存 (Save selected part key)
        this.selectedActionType = this.parts[partKey].action; // アクションタイプを設定 (Set action type)

        // 選択されたパーツのチャージ・クールダウン値を保存 (Store charge/cooldown values of the selected part)
        this.currentActionCharge = this.parts[partKey].charge;
        this.currentActionCooldown = this.parts[partKey].cooldown; // このクールダウン値はアクション実行後に使用 (This cooldown value is used after action execution)

        this.gauge = 0; // ゲージをリセット (Reset gauge for action charging)
        this.state = 'action_charging'; // アクションチャージング状態へ (To action_charging state)
    }

    // 2. 関心の分離（メダロットクラスの役割強化）(Separation of concerns (Strengthening Medarot class roles))
    // 利用可能な攻撃パーツ取得 (Get Available Attack Parts)
    // 壊れていない攻撃用パーツ（頭、右腕、左腕）のリストを返します (Returns a list of non-broken attack parts (head, rightArm, leftArm))
    getAvailableAttackParts() {
        return Object.entries(this.parts)
            .filter(([key, part]) => !part.isBroken && ['head', 'rightArm', 'leftArm'].includes(key))
            .map(([key, _]) => key);
    }

    // 選択準備完了か (Is Ready For Selection?)
    // アクション選択が可能な状態か（ready_select）を返します (Returns whether action selection is possible (ready_select))
    isReadyForSelection() {
        // cooldown_complete は無くなり、action_cooldown -> ready_select の流れになる
        return this.state === 'ready_select';
    }

    // ダメージ適用 (Apply Damage)
    // damage: 受けるダメージ量 (Amount of damage to receive)
    // partKey: ダメージを受けるパーツのキー (Key of the part receiving damage)
    // 指定されたパーツにダメージを与え、破壊された場合は状態を更新します。頭部が破壊された場合はメダロットも破壊状態になります。
    // (Applies damage to the specified part and updates its state if broken. If head part is broken, Medarot also becomes broken.)
    // 戻り値: 頭部が破壊された場合は true, それ以外は false (Return value: true if head part is destroyed, false otherwise)
    applyDamage(damage, partKey) {
        const part = this.parts[partKey];
        if (!part) return false; // パーツが存在しない場合は何もしない (If part doesn't exist, do nothing)
        part.hp = Math.max(0, part.hp - damage); // HPを減らす (Reduce HP)
        if (part.hp === 0) {
            part.isBroken = true; // HPが0なら破壊状態に (If HP is 0, set to broken state)
            if (partKey === 'head') {
                this.state = 'broken'; // 頭部破壊ならメダロットも破壊 (If head is broken, Medarot is also broken)
                return true; // 頭部破壊を示す (Indicates head part destroyed)
            }
        }
        return false; // 頭部破壊以外 (Other than head part destruction)
    }

    // ターン処理 (Process Turn)
    // メダロットのターンごとの状態更新を行います（ゲージ増加など）(Performs turn-based state updates for the Medarot (gauge increase, etc.))
    processTurn() {
        // 頭部が壊れていて、まだ破壊状態でないなら破壊状態にする (If head is broken and not yet in broken state, set to broken state)
        if (this.parts.head.isBroken && this.state !== 'broken') this.state = 'broken';

        // ゲージ更新を一時停止する状態 (States to pause gauge update)
        // cooldown_complete を削除 (Removed cooldown_complete)
        const statesToPause = ['ready_select', 'ready_execute', 'broken'];
        if (statesToPause.includes(this.state)) return; // これらの状態ならゲージ更新しない (If in these states, don't update gauge)

        this.gauge += this.speed; // スピードに応じてゲージ増加 (Increase gauge according to speed)

        // 状態ごとのゲージ上限と遷移ロジック (Gauge limits and transition logic for each state)
        if (this.state === 'idle_charging') { // 初期チャージ中 (During initial charge)
            if (this.gauge >= CONFIG.MAX_GAUGE) {
                this.gauge = CONFIG.MAX_GAUGE; // ゲージを上限値に設定 (Set gauge to max value)
                this.state = 'ready_select'; // アクション選択準備完了へ (To ready_select state)
            }
        } else if (this.state === 'action_charging') { // アクション選択後のチャージ中 (During charge after action selection)
            if (this.gauge >= this.currentActionCharge) {
                this.gauge = this.currentActionCharge; // ゲージをアクションチャージ値に設定 (Set gauge to action charge value)
                                                      // UI表示のため、updatePositionでMAX_GAUGEに対する割合に変換されることを想定
                                                      // (For UI display, it's assumed to be converted to a percentage of MAX_GAUGE in updatePosition)
                this.state = 'ready_execute'; // アクション実行準備完了へ (To ready_execute state)
            }
        } else if (this.state === 'action_cooldown') { // アクション後のクールダウン中 (During cooldown after action)
            if (this.gauge >= this.currentActionCooldown) {
                this.gauge = this.currentActionCooldown; // ゲージをアクションクールダウン値に設定 (Set gauge to action cooldown value)
                this.state = 'ready_select'; // アクション選択準備完了へ (To ready_select state)
                // currentActionCharge と currentActionCooldown は次のselectActionまで保持されるが、
                // ready_select状態では使用されないので問題ない
                // (currentActionCharge and currentActionCooldown are retained until the next selectAction,
                // but this is not an issue as they are not used in the ready_select state)
            }
        }
    }

    // --- UI関連 (UI Related) ---

    // アイコンDOM作成 (Create Icon DOM)
    // verticalPosition: 垂直位置 (%) (Vertical position in percentage)
    // メダロットのアイコンDOM要素を作成し返します (Creates and returns the Medarot's icon DOM element)
    createIconDOM(verticalPosition) {
        const icon = document.createElement('div');
        icon.id = `${this.id}-icon`; // アイコンID (Icon ID)
        icon.className = 'player-icon'; // CSSクラス (CSS class)
        icon.style.backgroundColor = this.color; // 背景色をチームカラーに (Background color to team color)
        icon.style.top = `${verticalPosition}%`; // Y座標設定 (Set Y coordinate)
        icon.style.transform = 'translate(-50%, -50%)'; // 中央揃え (Centering)
        icon.textContent = this.name.substring(this.name.length - 1); // 名前の最後の一文字を表示 (Display last character of name)
        this.iconElement = icon; // 要素を保存 (Save element)
        return icon;
    }

    // 情報パネルDOM作成 (Create Info Panel DOM)
    // メダロットの情報（パーツHPなど）を表示するDOM要素を作成し返します (Creates and returns DOM element to display Medarot info (part HP, etc.))
    createInfoPanelDOM() {
        const info = document.createElement('div');
        info.className = 'player-info'; // CSSクラス (CSS class)
        const teamConfig = CONFIG.TEAMS[this.team];
        let partsHTML = '';
        // 各パーツのHP表示エリアを作成 (Create HP display area for each part)
        Object.keys(this.parts).forEach(key => {
            partsHTML += `<div id="${this.id}-${key}-part" class="part-hp"></div>`;
        });
        info.innerHTML = `
            <div class="player-name ${teamConfig.textColor}">${this.name} ${this.isLeader ? '(L)' : ''}</div>
            <div class="parts-container">${partsHTML}</div>
        `;
        // 3. DOM参照をキャッシュ (Cache DOM references)
        Object.entries(this.parts).forEach(([key, part]) => {
            const partEl = info.querySelector(`#${this.id}-${key}-part`);
            partEl.innerHTML = `
                <span class="part-name">${part.name.substring(0,1)}</span>
                <div class="part-hp-bar-container"><div class="part-hp-bar"></div></div>
            `;
            // 各パーツのDOM要素を保存 (Save DOM elements for each part)
            this.partDOMElements[key] = {
                container: partEl, // パーツ全体のコンテナ (Container for the whole part)
                name: partEl.querySelector('.part-name'), // パーツ名表示用span (Span for part name display)
                bar: partEl.querySelector('.part-hp-bar') // HPバー表示用div (Div for HP bar display)
            };
        });
        return info;
    }

    // 表示更新 (Update Display)
    // メダロットのアイコン位置と情報パネルを更新します (Updates Medarot's icon position and info panel)
    updateDisplay() {
        this.updatePosition(); // 位置更新 (Update position)
        this.updateInfoPanel(); // 情報パネル更新 (Update info panel)
    }

    // 位置更新 (Update Position)
    // メダロットのアイコンのX座標をゲージの状態に応じて更新します (Updates X coordinate of Medarot's icon based on gauge state)
    updatePosition() {
        if (!this.iconElement) return; // アイコン要素がなければ何もしない (If no icon element, do nothing)

        let currentMaxGauge = CONFIG.MAX_GAUGE; // デフォルトの最大ゲージ (Default max gauge)
        // 状態に応じて現在の最大ゲージを設定 (Set current max gauge based on state)
        if (this.state === 'action_charging' && this.currentActionCharge) {
            currentMaxGauge = this.currentActionCharge;
        } else if (this.state === 'action_cooldown' && this.currentActionCooldown) {
            currentMaxGauge = this.currentActionCooldown;
        } else if (this.state === 'ready_execute' && this.currentActionCharge) { // ready_execute時もaction_chargingの目標値が最大
             currentMaxGauge = this.currentActionCharge;
        }
        // ready_select や idle_charging は CONFIG.MAX_GAUGE を使う

        const progress = Math.min(1, this.gauge / currentMaxGauge); // ゲージの進捗率 (Progress rate of gauge), 1を超えないように (capped at 1)
        let positionXRatio = (this.team === 'team1') ? 0 : 1; // チームによる初期X位置の割合 (Initial X position ratio by team)

        // 状態に応じたX位置の計算 (Calculation of X position according to state)
        // selected_charging -> action_charging
        // charging (汎用) -> idle_charging または action_cooldown
        if (this.state === 'action_charging') { // アクション選択後のチャージ中 (Charging after action selection)
            positionXRatio = (this.team === 'team1') ? (progress * 0.5) : (1 - (progress * 0.5));
        } else if (this.state === 'idle_charging' || this.state === 'action_cooldown') { // 初期チャージ中またはアクション後クールダウン中
            // 左側チームは0.5から左へ、右側チームは0.5から右へチャージ
            positionXRatio = (this.team === 'team1') ? (0.5 - (progress * 0.5)) : (0.5 + (progress * 0.5));
        } else if (this.state === 'ready_execute') { // 実行準備完了 (Ready to execute)
            positionXRatio = 0.5; // 中央へ (To the center)
        }
        // ready_select の場合は、上記の charging 系で progress が 1 になったときの位置 (0 or 1) に留まるか、
        // もしくは charge 系と同じように 0.5 から外側に向けて表示される。現在のXRatio計算だと後者。
        // For 'ready_select', it will either stay at the position where progress became 1 during charging (0 or 1),
        // or be displayed outward from 0.5 like the charging states. The current XRatio calculation does the latter.
        // For simplicity, ready_select will be at progress = 1 of its previous charging state.
        // If state is ready_select, it means progress reached 1 for either idle_charging or action_cooldown.
        // The position should reflect that (i.e. at the respective "home" side).
        else if (this.state === 'ready_select') {
             positionXRatio = (this.team === 'team1') ? 0 : 1; //ホームポジション
        }


        this.iconElement.style.left = `${positionXRatio * 100}%`; // X座標を適用 (Apply X coordinate)
        // 状態に応じたCSSクラスのトグル (Toggle CSS classes according to state)
        this.iconElement.classList.toggle('ready-select', this.isReadyForSelection());
        this.iconElement.classList.toggle('ready-execute', this.state === 'ready_execute');
        this.iconElement.classList.toggle('broken', this.state === 'broken');
    }

    // 情報パネル更新 (Update Info Panel)
    // 各パーツのHPバー表示を更新します (Updates HP bar display for each part)
    updateInfoPanel() {
        Object.entries(this.parts).forEach(([key, part]) => {
            const elements = this.partDOMElements[key];
            if (!elements) return; // DOM要素がなければ何もしない (If no DOM elements, do nothing)

            const hpPercentage = (part.hp / part.maxHp) * 100; // HPの割合 (HP percentage)
            elements.bar.style.width = `${hpPercentage}%`; // HPバーの幅を更新 (Update HP bar width)
            elements.container.classList.toggle('broken', part.isBroken); // 破壊状態なら 'broken' クラスを付与 (Add 'broken' class if part is broken)

            // HPに応じたバーの色変更 (Change bar color according to HP)
            if (part.isBroken) {
                elements.bar.style.backgroundColor = '#4a5568'; // 破壊時はグレー (Gray when broken)
            } else {
                if (hpPercentage > 50) elements.bar.style.backgroundColor = '#68d391'; // 50%超は緑 (Green if over 50%)
                else if (hpPercentage > 20) elements.bar.style.backgroundColor = '#f6e05e'; // 20%超は黄色 (Yellow if over 20%)
                else elements.bar.style.backgroundColor = '#f56565'; // 20%以下は赤 (Red if 20% or less)
            }
        });
    }
}

// ゲーム管理クラス (Game Manager Class)
// ゲーム全体の進行、UI、メダロット間のインタラクションを管理します (Manages overall game progress, UI, and interactions between Medarots)
class GameManager {
    // コンストラクタ (Constructor)
    constructor() {
        this.medarots = []; // 全メダロットのリスト (List of all Medarots) (formerly this.players)
        this.simulationInterval = null; // シミュレーションのインターバルID (Interval ID for simulation)
        this.activeMedarot = null; // 現在アクティブなメダロット (Currently active Medarot) (formerly this.activePlayer)
        this.phase = 'IDLE'; // 現在のゲームフェーズ (Current game phase): IDLE, INITIAL_SELECTION, BATTLE_START_CONFIRM, BATTLE, GAME_OVER
        // DOM要素への参照 (References to DOM elements)
        this.dom = {
            startButton: document.getElementById('startButton'), // スタートボタン (Start button)
            resetButton: document.getElementById('resetButton'), // リセットボタン (Reset button)
            battlefield: document.getElementById('battlefield'), // バトルフィールド (Battlefield)
            modal: document.getElementById('actionModal'), // モーダルウィンドウ (Modal window)
            modalTitle: document.getElementById('modalTitle'), // モーダルタイトル (Modal title)
            modalActorName: document.getElementById('modalActorName'), // モーダル内の行動主体名 (Actor name in modal)
            partSelectionContainer: document.getElementById('partSelectionContainer'), // パーツ選択コンテナ (Part selection container)
            modalConfirmButton: document.getElementById('modalConfirmButton'), // モーダル確認ボタン (Modal confirm button)
            battleStartConfirmButton: document.getElementById('battleStartConfirmButton') // 戦闘開始確認ボタン (Battle start confirm button)
        };
        // 各チームの情報パネルDOMへの参照 (References to each team's info panel DOM)
        Object.values(CONFIG.TEAMS).forEach(team => {
            this.dom[team.name.replace(/\s/g, '')] = document.getElementById(`${team.name.replace(/\s/g, '')}InfoPanel`);
        });
    }

    // 初期化 (Initialization)
    // ゲームの初期設定（メダロット作成、UI設定、イベント紐付け）を行います (Performs initial game setup (Medarot creation, UI setup, event binding))
    init() { this.createMedarots(); this.setupUI(); this.bindEvents(); } // createPlayers を createMedarots に変更 (Changed createPlayers to createMedarots)

    // メダロット作成 (Create Medarots)
    // 設定に基づいてメダロットのインスタンスを生成します (Generates Medarot instances based on configuration)
    createMedarots() { // Renamed from createPlayers
        this.medarots = []; // メダロットリストを初期化 (Initialize Medarot list)
        Object.entries(CONFIG.TEAMS).forEach(([teamId, teamConfig], teamIndex) => {
            for (let i = 0; i < CONFIG.PLAYERS_PER_TEAM; i++) {
                const id = teamIndex * CONFIG.PLAYERS_PER_TEAM + i + 1;
                // 新しいメダルインスタンスを作成 (Create a new Medal instance)
                // 例：とりあえず全メダロットにカブトメダルを割り当てる (Example: Assign Kabuto medal to all Medarots for now)
                const defaultMedal = new Medal(
                    (i % 2 === 0) ? "Kabuto" : "Kuwagata", //交互にメダルを設定
                    "Speed", "Berserk", (i % 2 === 0) ? "Fire" : "Thunder",
                    { shoot: 10, fight: 5, scan: 0, support: 0 }
                );
                this.medarots.push(new Medarot( // Player を Medarot に変更 (Changed Player to Medarot)
                    `p${id}`, `Medarot ${id}`, teamId,
                    teamConfig.baseSpeed + (Math.random() * 0.2),
                    defaultMedal, // メダル情報をコンストラクタに渡す (Pass medal info to constructor)
                    { isLeader: i === 0, color: teamConfig.color }
                ));
            }
        });
    }

    // UI設定 (Setup UI)
    // バトルフィールドと各メダロットの表示を初期化します (Initializes the battlefield and display for each Medarot)
    setupUI() {
        this.dom.battlefield.innerHTML = '<div class="center-line"></div>'; // 中央線を描画 (Draw center line)
        // 各チームの情報パネルを初期化 (Initialize info panel for each team)
        Object.entries(CONFIG.TEAMS).forEach(([teamId, teamConfig]) => {
            const panel = document.getElementById(`${teamId}InfoPanel`);
            panel.innerHTML = `<h2 class="text-xl font-bold mb-3 ${teamConfig.textColor}">${teamConfig.name}</h2>`;
        });
        // 各メダロットのDOMを作成し配置 (Create and place DOM for each Medarot)
        this.medarots.forEach(medarot => { // player を medarot に変更 (Changed player to medarot)
            const idNum = parseInt(medarot.id.substring(1));
            const indexInTeam = (idNum - 1) % CONFIG.PLAYERS_PER_TEAM;
            const vPos = 25 + indexInTeam * 25; // 垂直位置を計算 (Calculate vertical position)
            this.dom.battlefield.appendChild(medarot.createIconDOM(vPos)); // アイコンをバトルフィールドに追加 (Add icon to battlefield)
            const panel = document.getElementById(`${medarot.team}InfoPanel`); // 対応するチームパネルを取得 (Get corresponding team panel)
            panel.appendChild(medarot.createInfoPanelDOM()); // 情報パネルを追加 (Add info panel)
            medarot.updateDisplay(); // 表示を更新 (Update display)
        });
    }

    // イベント紐付け (Bind Events)
    // UI要素にイベントリスナーを設定します (Sets event listeners to UI elements)
    bindEvents() {
        this.dom.startButton.addEventListener('click', () => this.start());
        this.dom.resetButton.addEventListener('click', () => this.reset());
        this.dom.modalConfirmButton.addEventListener('click', () => this.handleModalConfirm());
        this.dom.battleStartConfirmButton.addEventListener('click', () => this.handleBattleStartConfirm());
    }

    // ゲーム開始 (Start Game)
    // ゲームを開始し、初期選択フェーズに移行します (Starts the game and transitions to initial selection phase)
    start() {
        if (this.phase !== 'IDLE') return; // IDLE状態以外なら何もしない (If not in IDLE state, do nothing)
        this.phase = 'INITIAL_SELECTION'; // 初期選択フェーズへ (To initial selection phase)
        // 全メダロットを準備完了状態にし、表示を更新 (Set all Medarots to ready state and update display)
        this.medarots.forEach(m => { m.gauge = CONFIG.MAX_GAUGE; m.state = 'ready_select'; m.updateDisplay(); }); // player を m に変更
        this.dom.startButton.disabled = true; this.dom.startButton.textContent = "Simulation in progress...";
        this.dom.resetButton.style.display = "inline-block";
        this.resumeSimulation(); // シミュレーション再開 (Resume simulation)
    }

    // シミュレーション一時停止 (Pause Simulation)
    // ゲームループのインターバルをクリアします (Clears the game loop interval)
    pauseSimulation() { clearInterval(this.simulationInterval); this.simulationInterval = null; }

    // シミュレーション再開 (Resume Simulation)
    // ゲームループのインターバルを開始します (Starts the game loop interval)
    resumeSimulation() { if (this.simulationInterval) return; this.simulationInterval = setInterval(() => this.gameLoop(), CONFIG.UPDATE_INTERVAL); }

    // ゲームリセット (Reset Game)
    // ゲーム状態を初期に戻します (Resets the game state to initial)
    reset() {
        this.pauseSimulation(); // シミュレーション停止 (Pause simulation)
        this.phase = 'IDLE'; this.activeMedarot = null; // activePlayer を activeMedarot に変更
        this.hideModal(); // モーダルを隠す (Hide modal)
        this.medarots.forEach(m => m.fullReset()); // 全メダロットをリセット (Reset all Medarots) player を m に変更
        this.setupUI(); // UIを再セットアップ (Re-setup UI)
        this.dom.startButton.disabled = false; this.dom.startButton.textContent = "Start Simulation";
        this.dom.resetButton.style.display = "none";
    }

    // 4. メインループの簡略化 (Simplification of the main loop)
    // ゲームループ (Game Loop)
    // ゲームのメインロジックを処理します (Processes the main logic of the game)
    gameLoop() {
        // アクティブなメダロットがいるか、特定のフェーズでなければ処理中断 (If there's an active Medarot or not in specific phases, interrupt processing)
        if (this.activeMedarot || !['INITIAL_SELECTION', 'BATTLE'].includes(this.phase)) return; // activePlayer を activeMedarot に変更

        // 優先度1: アクション実行 (Priority 1: Action Execution)
        const medarotToExecute = this.medarots.find(m => m.state === 'ready_execute'); // player を m に変更
        if (medarotToExecute) {
            return this.handleActionExecution(medarotToExecute); // medarotToExecute を渡す
        }

        // 優先度2: アクション選択 (Priority 2: Action Selection)
        const medarotToSelect = this.medarots.find(m => m.isReadyForSelection()); // player を m に変更
        if (medarotToSelect) {
            return this.handleActionSelection(medarotToSelect); // medarotToSelect を渡す
        }

        // 誰も行動しない場合 (If no one acts)
        if (this.phase === 'INITIAL_SELECTION') {
            // 全メダロットが選択完了したら、戦闘開始確認へ (If all Medarots have finished selection, proceed to battle start confirmation)
            if (this.medarots.every(m => m.state !== 'ready_select')) { // player を m に変更
                this.phase = 'BATTLE_START_CONFIRM';
                this.pauseSimulation();
                this.showModal('battle_start_confirm');
            }
        } else if (this.phase === 'BATTLE') {
            // 戦闘フェーズなら各メダロットのターン処理と表示更新 (If in battle phase, process turn and update display for each Medarot)
            this.medarots.forEach(m => { m.processTurn(); m.updateDisplay(); }); // player を m に変更
        }
    }

    // アクション実行処理 (Handle Action Execution)
    // medarot: 実行するメダロット (Medarot to execute action)
    handleActionExecution(medarot) { // player を medarot に変更
        this.activeMedarot = medarot; // activePlayer を activeMedarot に変更
        this.pauseSimulation(); // シミュレーション一時停止 (Pause simulation)
        this.prepareAndShowExecutionModal(medarot); // 実行モーダル準備表示 (Prepare and show execution modal)
    }

    // アクション選択処理 (Handle Action Selection)
    // medarot: 選択するメダロット (Medarot to select action)
    handleActionSelection(medarot) { // player を medarot に変更
        // medarot.state = 'ready_select'; // 不要になった正規化処理 (Normalization no longer needed)
        if (medarot.team === 'team2') { // CPUロジック (CPU logic)
            const target = this.findEnemyTarget(medarot); // 敵ターゲットを探す (Find enemy target)
            const partKey = medarot.getAvailableAttackParts()[0]; // とりあえず最初のパーツ (For now, just the first part)
            if (target && partKey) medarot.selectAction(partKey); else medarot.state = 'broken'; // ターゲットとパーツがあればアクション選択、なければ破壊状態 (If target and part exist, select action, otherwise broken state)
        } else { // 人間プレイヤーロジック (Human player logic)
            this.activeMedarot = medarot; // activePlayer を activeMedarot に変更
            this.pauseSimulation();
            this.showModal('selection', medarot); // 選択モーダル表示 (Show selection modal)
        }
    }

    // 敵ターゲット検索 (Find Enemy Target)
    // attacker: 攻撃側のメダロット (Attacking Medarot)
    // actionType: 実行するアクションのタイプ ('Shoot', 'Fight', 'Scan', etc.)
    // 攻撃対象となる敵メダロットを検索します (Searches for an enemy Medarot to target based on action type)
    findEnemyTarget(attacker, actionType) {
        const enemies = this.medarots.filter(p => p.team !== attacker.team && p.state !== 'broken');
        if (enemies.length === 0) return null;

        // 距離計算ヘルパー関数 (Distance calculation helper function)
        const calculateDistance = (medarot1, medarot2) => {
            // アイコン要素とそのstyle.leftが設定されているか確認
            // (Check if iconElement and its style.left are set)
            if (!medarot1.iconElement || !medarot1.iconElement.style.left ||
                !medarot2.iconElement || !medarot2.iconElement.style.left) {
                return Infinity; // 位置情報がない場合は距離無限大として扱う (Treat as infinite distance if position info is missing)
            }
            const x1 = parseFloat(medarot1.iconElement.style.left);
            const x2 = parseFloat(medarot2.iconElement.style.left);

            if (isNaN(x1) || isNaN(x2)) {
                // parseFloatがNaNを返した場合も距離無限大として扱う
                // (Treat as infinite distance if parseFloat returns NaN)
                return Infinity;
            }
            return Math.abs(x1 - x2);
        };

        const enemiesWithDistance = enemies.map(enemy => ({
            medarot: enemy,
            distance: calculateDistance(attacker, enemy)
        })).filter(e => isFinite(e.distance)); // 有効な距離を持つ敵のみフィルタリング (Filter only enemies with valid distances)

        if (enemiesWithDistance.length === 0 && enemies.length > 0) {
            // 距離計算が有効な敵がいなかったが、敵自体は存在する場合のフォールバック
            // (Fallback if no enemies with valid distances were found, but enemies exist)
            return enemies.find(e => e.isLeader) || enemies[0];
        }
        if (enemiesWithDistance.length === 0) return null;


        if (actionType === 'Shoot') { // 射撃 - 最も遠い敵 (Shoot - Farthest enemy)
            enemiesWithDistance.sort((a, b) => b.distance - a.distance); // 距離で降順ソート (Sort by distance in descending order)
            return enemiesWithDistance[0].medarot;
        } else if (actionType === 'Fight') { // 格闘 - 最も近い敵 (Fight - Closest enemy)
            enemiesWithDistance.sort((a, b) => a.distance - b.distance); // 距離で昇順ソート (Sort by distance in ascending order)
            return enemiesWithDistance[0].medarot;
        } else { // デフォルトまたはその他アクションタイプ (Default or other action types, e.g., Scan)
            // 従来のリーダー優先ロジック (Traditional leader priority logic)
            return enemies.find(e => e.isLeader) || enemies[0];
        }
    }

    // パーツ選択ハンドラ (Handle Part Selection)
    // partKey: 選択されたパーツのキー (Key of the selected part)
    // アクティブなメダロットがパーツを選択した際の処理 (Processing when active Medarot selects a part)
    handlePartSelection(partKey) {
        if (!this.activeMedarot) return; // アクティブなメダロットがいなければ何もしない (If no active Medarot, do nothing)
        this.activeMedarot.selectAction(partKey); // アクション選択 (Select action)
        this.activeMedarot = null; // アクティブメダロットをクリア (Clear active Medarot)
        this.hideModal(); // モーダルを隠す (Hide modal)
        this.resumeSimulation(); // シミュレーション再開 (Resume simulation)
    }

    // 戦闘開始確認ハンドラ (Handle Battle Start Confirm)
    // 戦闘開始確認モーダルのボタンが押された際の処理 (Processing when battle start confirm modal button is pressed)
    handleBattleStartConfirm() {
        this.phase = 'BATTLE'; // 戦闘フェーズへ (To battle phase)
        this.medarots.forEach(m => m.processTurn()); // 各メダロットのターン処理 (Process turn for each Medarot) player を m に変更
        this.hideModal();
        this.resumeSimulation();
    }

    // モーダル確認ハンドラ (Handle Modal Confirm)
    // 実行モーダルまたはゲームオーバーモーダルの確認ボタンが押された際の処理 (Processing when confirm button of execution or game over modal is pressed)
    handleModalConfirm() {
        if (this.phase === 'GAME_OVER') return this.reset(); // ゲームオーバーならリセット (If game over, reset)
        if (!this.activeMedarot) return; // アクティブメダロットがいなければ何もしない (If no active Medarot, do nothing)

        const attacker = this.activeMedarot; // 攻撃者 (Attacker)
        if (attacker.preparedAttack) {
            const { target, partKey, damage } = attacker.preparedAttack;
            // ダメージ適用とリーダー破壊判定 (Apply damage and check for leader destruction)
            if (target.applyDamage(damage, partKey) && target.isLeader) {
                this.phase = 'GAME_OVER'; // ゲームオーバーフェーズへ (To game over phase)
                this.showModal('game_over', null, { winningTeam: attacker.team }); // ゲームオーバーモーダル表示 (Show game over modal)
                this.pauseSimulation(); // シミュレーション停止 (Pause simulation)
                return;
            }
        }
        attacker.startCooldown(); // クールダウン開始 (Start cooldown)
        this.activeMedarot = null; // アクティブメダロットをクリア (Clear active Medarot)
        this.hideModal();
        this.resumeSimulation();
    }

    // 実行モーダル準備と表示 (Prepare And Show Execution Modal)
    // medarot: 行動するメダロット (Medarot performing the action)
    // 攻撃対象と攻撃部位を決定し、実行モーダルを表示します (Determines attack target and part, then shows execution modal)
    prepareAndShowExecutionModal(medarot) { // player を medarot に変更
        // アクションタイプに基づいてターゲットを検索 (Find target based on action type)
        const actionType = medarot.selectedActionType;
        const target = this.findEnemyTarget(medarot, actionType);

        if (!target) { // ターゲットが見つからない場合 (If no target is found)
            // アクションを実行できないので、クールダウンに移行 (Cannot perform action, so transition to cooldown)
            // この場合、どのパーツのクールダウンを使うか？ selectedPartKeyはあるはず。
            // (In this case, which part's cooldown to use? selectedPartKey should exist.)
            // selectActionでcurrentActionCooldownは設定されているので、そのままstartCooldownを呼べば良い
            // (currentActionCooldown is set in selectAction, so calling startCooldown directly is fine)
            return medarot.startCooldown();
        }

        // ターゲットの破壊されていないパーツリストを取得 (Get list of target's non-broken parts)
        const availableTargetParts = Object.keys(target.parts).filter(key => !target.parts[key].isBroken);
        if (availableTargetParts.length === 0) { return medarot.startCooldown(); } // 攻撃可能なパーツがなければクールダウン (If no attackable parts, start cooldown)

        // 攻撃対象パーツをランダムに選択 (Randomly select target part)
        const targetPartKey = availableTargetParts[Math.floor(Math.random() * availableTargetParts.length)];
        // 攻撃情報を準備 (Prepare attack information)
        medarot.preparedAttack = {
            target: target,
            partKey: targetPartKey,
            damage: CONFIG.BASE_DAMAGE // 基本ダメージを使用 (Use base damage)
        };
        this.showModal('execution', medarot); // 実行モーダル表示 (Show execution modal)
    }

    // モーダル表示 (Show Modal)
    // type: モーダルの種類 (selection, execution, battle_start_confirm, game_over) (Type of modal)
    // medarot:関連するメダロット (nullの場合あり) (Associated Medarot (can be null))
    // data: 追加データ (ゲームオーバー時の勝利チームなど) (Additional data (e.g., winning team at game over))
    showModal(type, medarot = null, data = {}) { // player を medarot に変更済み
        const modal = this.dom.modal; // モーダル本体 (Modal body)
        const title = this.dom.modalTitle; // モーダルタイトル要素 (Modal title element)
        const actorName = this.dom.modalActorName; // 行動主体名表示要素 (Actor name display element)
        const partContainer = this.dom.partSelectionContainer; // パーツ選択コンテナ要素 (Part selection container element)
        const confirmBtn = this.dom.modalConfirmButton; // モーダル確認ボタン要素 (Modal confirm button element)
        const startBtn = this.dom.battleStartConfirmButton; // 戦闘開始確認ボタン要素 (Battle start confirm button element)

        // モーダル要素を一旦リセット (Reset modal elements first)
        [partContainer, confirmBtn, startBtn].forEach(el => el.style.display = 'none'); // 全ての動的要素を非表示 (Hide all dynamic elements)
        modal.className = 'modal'; // 基本クラスを再適用 (Reapply base class)

        // モーダルの種類に応じて内容と表示を切り替える (Switch content and display according to modal type)
        switch (type) {
            case 'selection': // アクション選択モーダル (Action selection modal)
                title.textContent = 'Select Action'; // タイトル設定 (Set title)
                actorName.textContent = `${medarot.name}'s turn.`; // 行動メダロット名表示 (Display acting Medarot's name)
                partContainer.innerHTML = ''; // パーツボタンコンテナをクリア (Clear parts button container)
                // 利用可能な攻撃パーツのボタンを動的に生成 (Dynamically generate buttons for available attack parts)
                medarot.getAvailableAttackParts().forEach(partKey => {
                    const part = medarot.parts[partKey];
                    const button = document.createElement('button');
                    button.className = 'part-action-button'; // CSSクラス設定 (Set CSS class)
                    button.textContent = `${part.name} (${part.action})`; // ボタンテキスト設定 (Set button text) e.g., "Head (Scan)"
                    button.onclick = () => this.handlePartSelection(partKey); // クリック時のイベントハンドラ設定 (Set click event handler)
                    partContainer.appendChild(button); // コンテナにボタン追加 (Add button to container)
                });
                partContainer.style.display = 'flex'; // パーツ選択コンテナ表示 (Show part selection container)
                break;
            case 'execution': // 攻撃実行モーダル (Attack execution modal)
                const { target, partKey, damage } = medarot.preparedAttack; // 準備された攻撃情報を展開 (Destructure prepared attack info)
                title.textContent = 'Execute Attack!'; // タイトル設定 (Set title)
                actorName.textContent = `${medarot.name}'s ${medarot.selectedActionType}! Dealt ${damage} damage to ${target.name}'s ${target.parts[partKey].name}!`;
                // 結果メッセージ表示 (Display result message) e.g., "Medarot 1's Shoot! Dealt 20 damage to Medarot 2's Head!"
                confirmBtn.style.display = 'inline-block'; // 確認ボタン表示 (Show confirm button)
                confirmBtn.textContent = 'OK'; // ボタンテキスト設定 (Set button text)
                break;
            case 'battle_start_confirm': // 戦闘開始確認モーダル (Battle start confirm modal)
                title.textContent = 'Battle Start!'; // タイトル設定 (Set title)
                actorName.textContent = ''; // 行動主体名はなし (No actor name for this modal)
                startBtn.style.display = 'inline-block'; // 戦闘開始ボタン表示 (Show battle start button)
                break;
            case 'game_over': // ゲームオーバーモーダル (Game over modal)
                title.textContent = `${CONFIG.TEAMS[data.winningTeam].name} Wins!`; // 勝者チーム表示 (Display winning team)
                actorName.textContent = 'Robattle Over!'; // 「ロボトル終了！」メッセージ (Robattle Over! message)
                confirmBtn.style.display = 'inline-block'; // 確認ボタン表示 (Show confirm button)
                confirmBtn.textContent = 'Reset'; // ボタンテキストを「リセット」に (Set button text to "Reset")
                modal.classList.add('game-over-modal'); // ゲームオーバー専用の追加スタイルを適用 (Apply additional style for game over)
                break;
        }
        modal.classList.remove('hidden'); // モーダル全体を表示状態にする (Set the whole modal to visible state)
    }

    // モーダル非表示 (Hide Modal)
    // モーダルを隠します (Hides the modal)
    hideModal() { this.dom.modal.classList.add('hidden'); } // 'hidden' クラスを付与して非表示化 (Add 'hidden' class to hide)
}

// DOMContentLoadedイベントリスナー (DOMContentLoaded event listener)
// HTMLの読み込みと解析が完了した時点で、GameManagerインスタンスを作成し、ゲームを初期化します。
// (When HTML loading and parsing is complete, create a GameManager instance and initialize the game.)
document.addEventListener('DOMContentLoaded', () => {
    const game = new GameManager(); // ゲームマネージャーのインスタンスを作成 (Create an instance of the game manager)
    game.init(); // ゲームを初期化 (Initialize the game)
});
