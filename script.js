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
    constructor(data) {
        this.id = data.id; // Store ID
        this.name = data.name_jp; // Use name_jp from CSV
        this.personality = data.personality_jp;
        this.medaforce = data.medaforce_jp;
        this.attribute = data.attribute_jp;
        this.skillLevels = {
            shoot: parseInt(data.skill_shoot) || 0,
            fight: parseInt(data.skill_fight) || 0,
            scan: parseInt(data.skill_scan) || 0,
            support: parseInt(data.skill_support) || 0
        };
    }
}

// メダロットクラス (Medarot Class)
class Medarot {
    constructor(id, name, team, speed, medal, options, partsConfig, partsData) {
        this.id = id;
        this.name = name;
        this.team = team;
        this.speed = speed;
        this.medal = medal;
        this.isLeader = options.isLeader;
        this.color = options.color;
        this.partsConfig = partsConfig;
        this.allPartsData = partsData;
        this.iconElement = null;
        this.partDOMElements = {};

        this.currentActionCharge = null;
        this.currentActionCooldown = null;
        this.fullReset();
    }

    fullReset() {
        this.gauge = 0;
        this.state = 'idle_charging';
        this.selectedActionType = null;
        this.selectedPartKey = null;
        this.preparedAttack = null;
        this.currentActionCharge = null;
        this.currentActionCooldown = null;
        this.currentTargetedEnemy = null;
        this.currentTargetedPartKey = null;
        this.pendingTargetEnemy = null;
        this.pendingTargetPartKey = null;

        this.legMovementType = null;
        this.legAccuracy = 0;
        this.legMobility = 0;
        this.legPropulsion = 0;
        this.legDefenseParam = 0;

        this.parts = {};

        const defaultPartStructure = {
            id: 'N/A', name_jp: '未装備', category_jp: 'なし', sub_category_jp: 'なし',
            slot: '', hp: 1, maxHp: 1, charge: 0, cooldown: 0, isBroken: true
        };

        const slots = ['head', 'rightArm', 'leftArm', 'legs'];
        slots.forEach(slotKey => {
            const partId = this.partsConfig ? this.partsConfig[slotKey] : null;
            let partDataFound = false;

            if (partId && this.allPartsData && this.allPartsData[slotKey] && Array.isArray(this.allPartsData[slotKey])) {
                const partData = this.allPartsData[slotKey].find(p => p.id === partId);
                if (partData) {
                    this.parts[slotKey] = {
                        id: partData.id, name_jp: partData.name_jp, category_jp: partData.category_jp,
                        sub_category_jp: partData.sub_category_jp, slot: slotKey,
                        hp: parseInt(partData.base_hp) || CONFIG.PART_HP_BASE,
                        maxHp: parseInt(partData.base_hp) || CONFIG.PART_HP_BASE,
                        charge: parseInt(partData.charge) || 0,
                        cooldown: parseInt(partData.cooldown) || 0,
                        isBroken: false
                    };

                    if (slotKey === 'legs') {
                        this.parts[slotKey].hp += CONFIG.LEGS_HP_BONUS;
                        this.parts[slotKey].maxHp += CONFIG.LEGS_HP_BONUS;
                        this.legMovementType = partData.movement_type_jp;
                        this.legAccuracy = parseInt(partData.accuracy) || 0;
                        this.legMobility = parseInt(partData.mobility) || 0;
                        this.legPropulsion = parseInt(partData.propulsion) || 0;
                        this.legDefenseParam = parseInt(partData.defense_param) || 0;
                    }
                    partDataFound = true;
                } else {
                    console.warn(`Part ID '${partId}' for slot '${slotKey}' not found in loaded parts data. Equipping default.`);
                }
            } else {
                if (partId) {
                    console.warn(`Parts data for slot '${slotKey}' missing or invalid, or partId '${partId}' configured but data structure issue. Equipping default.`);
                }
            }

            if (!partDataFound) {
                this.parts[slotKey] = { ...defaultPartStructure, name_jp: `パーツ無 (${partId || '未指定'})`, slot: slotKey };
                if (slotKey === 'legs') {
                    this.legMovementType = null; this.legAccuracy = 0; this.legMobility = 0;
                    this.legPropulsion = 0; this.legDefenseParam = 0;
                }
            }
        });
    }

    startCooldown() {
        this.gauge = 0;
        this.state = 'action_cooldown';
        this.selectedActionType = null;
        this.selectedPartKey = null;
        this.preparedAttack = null;
        this.currentTargetedEnemy = null;
        this.currentTargetedPartKey = null;
        this.pendingTargetEnemy = null;
        this.pendingTargetPartKey = null;
    }

    selectAction(partKey) {
        this.currentTargetedEnemy = null;
        this.currentTargetedPartKey = null;
        this.pendingTargetEnemy = null;
        this.pendingTargetPartKey = null;
        this.selectedPartKey = partKey;
        this.selectedActionType = this.parts[partKey].sub_category_jp;
        this.currentActionCharge = this.parts[partKey].charge;
        this.currentActionCooldown = this.parts[partKey].cooldown;
        this.gauge = 0;
        this.state = 'action_charging';
    }

    getAvailableAttackParts() {
        return Object.entries(this.parts)
            .filter(([key, part]) => part && !part.isBroken && ['head', 'rightArm', 'leftArm'].includes(key))
            .map(([key, _]) => key);
    }

    isReadyForSelection() { return this.state === 'ready_select'; }

    applyDamage(damage, partKey) {
        const part = this.parts[partKey];
        if (!part) return false;
        let effectiveDamage = damage;
        if (this.legDefenseParam && this.legDefenseParam > 0) {
            effectiveDamage = Math.max(1, damage - (this.legDefenseParam || 0));
        }
        part.hp = Math.max(0, part.hp - effectiveDamage);
        if (part.hp === 0) {
            part.isBroken = true;
            if (partKey === 'head') { this.state = 'broken'; return true; }
        }
        return false;
    }

    processTurn() {
        if (this.parts.head && this.parts.head.isBroken && this.state !== 'broken') this.state = 'broken';
        const statesToPause = ['ready_select', 'ready_execute', 'broken'];
        if (statesToPause.includes(this.state)) return;
        const baseChargeRate = this.speed;
        const propulsionBonus = (this.legPropulsion || 0) * 0.05;
        this.gauge += baseChargeRate + propulsionBonus;

        if (this.state === 'idle_charging') {
            if (this.gauge >= CONFIG.MAX_GAUGE) {
                this.gauge = CONFIG.MAX_GAUGE; this.state = 'ready_select';
            }
        } else if (this.state === 'action_charging') {
            if (this.gauge >= this.currentActionCharge) {
                this.gauge = this.currentActionCharge;
                if (this.selectedPartKey && this.parts[this.selectedPartKey]) {
                    const currentSelectedPart = this.parts[this.selectedPartKey];
                    if (currentSelectedPart.category_jp === '射撃') {
                        let targetIsValid = false;
                        if (this.currentTargetedEnemy && this.currentTargetedEnemy.state !== 'broken' &&
                            this.currentTargetedPartKey &&
                            this.currentTargetedEnemy.parts[this.currentTargetedPartKey] &&
                            !this.currentTargetedEnemy.parts[this.currentTargetedPartKey].isBroken) {
                            targetIsValid = true;
                        }
                        if (!targetIsValid) {
                            console.log(`${this.name}'s shooting target (${this.currentTargetedEnemy ? this.currentTargetedEnemy.name : 'N/A'}'s ${this.currentTargetedPartKey || 'N/A'}) is no longer valid. Action cancelled, starting cooldown.`);
                            this.startCooldown(); return;
                        }
                    }
                } else if (this.selectedPartKey) {
                    console.error(`${this.name} has an invalid selectedPartKey: ${this.selectedPartKey} during charge completion. Forcing cooldown.`);
                    this.startCooldown(); return;
                }
                this.state = 'ready_execute';
            }
        } else if (this.state === 'action_cooldown') {
            if (this.gauge >= this.currentActionCooldown) {
                this.gauge = this.currentActionCooldown; this.state = 'ready_select';
            }
        }
    }

    createIconDOM(verticalPosition) {
        const icon = document.createElement('div');
        icon.id = `${this.id}-icon`; icon.className = 'player-icon';
        icon.style.backgroundColor = this.color; icon.style.top = `${verticalPosition}%`;
        icon.style.transform = 'translate(-50%, -50%)';
        icon.textContent = this.name.substring(this.name.length - 1);
        this.iconElement = icon; return icon;
    }

    createInfoPanelDOM() {
        const info = document.createElement('div');
        info.className = 'player-info';
        const teamConfig = CONFIG.TEAMS[this.team];
        const partSlotNamesJP = { head: '頭部', rightArm: '右腕', leftArm: '左腕', legs: '脚部' };
        let partsHTML = '';
        Object.keys(this.parts).forEach(key => {
            partsHTML += `
                <div class="part-info-wrapper" id="${this.id}-${key}-wrapper">
                    <div class="part-name-display" id="${this.id}-${key}-name"></div>
                    <div class="part-hp-section">
                        <div class="part-hp-bar-container" id="${this.id}-${key}-bar-container">
                            <div class="part-hp-bar" id="${this.id}-${key}-bar"></div>
                        </div>
                        <span class="part-hp-numeric" id="${this.id}-${key}-hp-numeric"></span>
                    </div>
                </div>`;
        });
        info.innerHTML = `<div class="player-name ${teamConfig.textColor}">${this.name} ${this.isLeader ? '(L)' : ''}</div><div class="parts-container">${partsHTML}</div>`;
        Object.entries(this.parts).forEach(([key, part]) => {
            const wrapperElement = info.querySelector(`#${this.id}-${key}-wrapper`);
            const nameDisplayElement = info.querySelector(`#${this.id}-${key}-name`);
            const barElement = info.querySelector(`#${this.id}-${key}-bar`);
            const numericHpElement = info.querySelector(`#${this.id}-${key}-hp-numeric`);
            if (nameDisplayElement) nameDisplayElement.textContent = `${partSlotNamesJP[key] || key}: ${part.name_jp || 'N/A'}`;
            if (numericHpElement) numericHpElement.textContent = `(${part.hp}/${part.maxHp})`;
            this.partDOMElements[key] = { wrapper: wrapperElement, nameDisplay: nameDisplayElement, bar: barElement, numericHp: numericHpElement };
        });
        return info;
    }

    updateDisplay() { this.updatePosition(); this.updateInfoPanel(); }

    updatePosition() {
        if (!this.iconElement) return;
        let currentMaxGauge = CONFIG.MAX_GAUGE;
        if (this.state === 'action_charging' && this.currentActionCharge) currentMaxGauge = this.currentActionCharge;
        else if (this.state === 'action_cooldown' && this.currentActionCooldown) currentMaxGauge = this.currentActionCooldown;
        else if (this.state === 'ready_execute' && this.currentActionCharge) currentMaxGauge = this.currentActionCharge;
        const progress = Math.min(1, this.gauge / currentMaxGauge);
        let positionXRatio = (this.team === 'team1') ? 0 : 1;
        if (this.state === 'action_charging') positionXRatio = (this.team === 'team1') ? (progress * 0.5) : (1 - (progress * 0.5));
        else if (this.state === 'idle_charging' || this.state === 'action_cooldown') positionXRatio = (this.team === 'team1') ? (0.5 - (progress * 0.5)) : (0.5 + (progress * 0.5));
        else if (this.state === 'ready_execute') positionXRatio = 0.5;
        else if (this.state === 'ready_select') positionXRatio = (this.team === 'team1') ? 0 : 1;
        this.iconElement.style.left = `${positionXRatio * 100}%`;
        this.iconElement.classList.toggle('ready-select', this.isReadyForSelection());
        this.iconElement.classList.toggle('ready-execute', this.state === 'ready_execute');
        this.iconElement.classList.toggle('broken', this.state === 'broken');
    }

    updateInfoPanel() {
        Object.entries(this.parts).forEach(([key, part]) => {
            const elements = this.partDOMElements[key];
            if (!elements || !elements.bar || !elements.numericHp || !elements.wrapper) return;
            const hpPercentage = (part.hp / part.maxHp) * 100;
            elements.bar.style.width = `${hpPercentage}%`;
            elements.numericHp.textContent = `(${part.hp}/${part.maxHp})`;
            elements.wrapper.classList.toggle('broken', part.isBroken);
            if (part.isBroken) elements.bar.style.backgroundColor = '#4a5568';
            else {
                if (hpPercentage > 50) elements.bar.style.backgroundColor = '#68d391';
                else if (hpPercentage > 20) elements.bar.style.backgroundColor = '#f6e05e';
                else elements.bar.style.backgroundColor = '#f56565';
            }
        });
    }
}

class GameManager {
    constructor() {
        this.partsData = { head: [], rightArm: [], leftArm: [], legs: [] };
        this.medalsData = [];
        this.medarots = [];
        this.simulationInterval = null;
        this.activeMedarot = null;
        this.phase = 'IDLE';
        this.dom = {
            startButton: document.getElementById('startButton'),
            resetButton: document.getElementById('resetButton'),
            battlefield: document.getElementById('battlefield'),
            modal: document.getElementById('actionModal'),
            modalTitle: document.getElementById('modalTitle'),
            modalActorName: document.getElementById('modalActorName'),
            partSelectionContainer: document.getElementById('partSelectionContainer'),
            modalConfirmButton: document.getElementById('modalConfirmButton'),
            battleStartConfirmButton: document.getElementById('battleStartConfirmButton'),
            modalExecuteAttackButton: document.getElementById('modalExecuteAttackButton'),
            modalCancelActionButton: document.getElementById('modalCancelActionButton'),
            aimingArrow: document.getElementById('aiming-arrow')
        };
        Object.values(CONFIG.TEAMS).forEach(team => {
            this.dom[team.name.replace(/\s/g, '')] = document.getElementById(`${team.name.replace(/\s/g, '')}InfoPanel`);
        });
    }

    async init() {
        await this.loadGameData(); this.createMedarots(); this.setupUI(); this.bindEvents();
    }

    parseCsvText(csvText) {
        const lines = csvText.trim().split('\n');
        if (lines.length < 1) return [];
        const headers = lines[0].split(',').map(h => h.trim());
        if (lines.length < 2) return [];
        return lines.slice(1).map(line => {
            const values = line.split(','); const entry = {};
            headers.forEach((header, index) => { entry[header] = values[index] ? values[index].trim() : ''; });
            return entry;
        }).filter(entry => entry.id);
    }

    async loadGameData() {
        const filesToLoad = [
            { key: 'medals', path: 'medals.csv', target: 'medalsData' },
            { key: 'head', path: 'head_parts.csv', target: 'partsData', slot: 'head' },
            { key: 'rightArm', path: 'right_arm_parts.csv', target: 'partsData', slot: 'rightArm' },
            { key: 'leftArm', path: 'left_arm_parts.csv', target: 'partsData', slot: 'leftArm' },
            { key: 'legs', path: 'legs_parts.csv', target: 'partsData', slot: 'legs' }
        ];
        this.medalsData = [];
        this.partsData = { head: [], rightArm: [], leftArm: [], legs: [] };
        for (const file of filesToLoad) {
            try {
                const response = await fetch(file.path);
                if (!response.ok) { console.error(`Failed to load ${file.path}: ${response.statusText}`); continue; }
                const csvText = await response.text();
                const parsedData = this.parseCsvText(csvText);
                if (file.target === 'medalsData') this.medalsData = parsedData;
                else if (file.target === 'partsData' && file.slot) this.partsData[file.slot] = parsedData;
            } catch (error) { console.error(`Error loading or parsing ${file.path}:`, error); }
        }
    }

    createMedarots() {
        this.medarots = [];
        const defaultLoadouts = [
            { head: "P002", rightArm: "P003", leftArm: "P003", legs: "P002" },
            { head: "P002", rightArm: "P002", leftArm: "P001", legs: "P003" },
            { head: "P005", rightArm: "P001", leftArm: "P004", legs: "P004" },
            { head: "P001", rightArm: "P007", leftArm: "P003", legs: "P005" },
            { head: "P004", rightArm: "P004", leftArm: "P001", legs: "P006" },
            { head: "P005", rightArm: "P006", leftArm: "P006", legs: "P007" }
        ];
        Object.entries(CONFIG.TEAMS).forEach(([teamId, teamConfig], teamIndex) => {
            for (let i = 0; i < CONFIG.PLAYERS_PER_TEAM; i++) {
                const medarotIdNumber = teamIndex * CONFIG.PLAYERS_PER_TEAM + i + 1;
                const medarotDisplayId = `p${medarotIdNumber}`;
                let medarotPartsConfig; let selectedMedalDataForCurrentMedarot;
                if (teamIndex === 0 && i === 0) {
                    console.log(`Attempting to equip METABEE_SET for Medarot ${medarotDisplayId}`);
                    const targetSetId = "METABEE_SET"; const targetPartIdInSet = "P001";
                    const partsConfigForSet = {}; let setComplete = true;
                    const slotsToEquip = ['head', 'rightArm', 'leftArm', 'legs'];
                    for (const slot of slotsToEquip) {
                        if (this.partsData[slot] && Array.isArray(this.partsData[slot])) {
                            const part = this.partsData[slot].find(p => p.set_id === targetSetId && p.id === targetPartIdInSet);
                            if (part) partsConfigForSet[slot] = part.id;
                            else { setComplete = false; console.warn(`Part not found for METABEE_SET: slot ${slot}, set_id ${targetSetId}, part_id ${targetPartIdInSet}`); break; }
                        } else { setComplete = false; console.warn(`Parts data for slot ${slot} is missing or not an array.`); break; }
                    }
                    if (setComplete) {
                        medarotPartsConfig = partsConfigForSet;
                        selectedMedalDataForCurrentMedarot = this.medalsData.find(m => m.id === "M001");
                        if (!selectedMedalDataForCurrentMedarot) {
                            console.warn("METABEE_SET's conventional medal (M001) not found. Falling back to default medal selection for p1.");
                            const medalIndex = (teamIndex * CONFIG.PLAYERS_PER_TEAM + i) % this.medalsData.length;
                            selectedMedalDataForCurrentMedarot = this.medalsData[medalIndex];
                        }
                    } else {
                        console.warn(`METABEE_SET for ${medarotDisplayId} is incomplete. Falling back to default loadout for p1.`);
                        const loadoutIndex = teamIndex * CONFIG.PLAYERS_PER_TEAM + i;
                        medarotPartsConfig = defaultLoadouts[loadoutIndex % defaultLoadouts.length];
                        const medalIndex = (teamIndex * CONFIG.PLAYERS_PER_TEAM + i) % this.medalsData.length;
                        selectedMedalDataForCurrentMedarot = this.medalsData[medalIndex];
                    }
                } else {
                    const loadoutIndex = teamIndex * CONFIG.PLAYERS_PER_TEAM + i;
                    medarotPartsConfig = defaultLoadouts[loadoutIndex % defaultLoadouts.length];
                    const medalIndex = (teamIndex * CONFIG.PLAYERS_PER_TEAM + i) % this.medalsData.length;
                    selectedMedalDataForCurrentMedarot = this.medalsData[medalIndex];
                }
                let medalForMedarot;
                if (selectedMedalDataForCurrentMedarot) medalForMedarot = new Medal(selectedMedalDataForCurrentMedarot);
                else {
                    console.warn(`No medal data found for Medarot ${medarotDisplayId}. Using a default fallback medal.`);
                    medalForMedarot = new Medal({ id: 'M_FALLBACK', name_jp: 'フォールバックメダル', personality_jp: 'ノーマル', medaforce_jp: 'なし', attribute_jp: '無', skill_shoot: '1', skill_fight: '1', skill_scan: '1', skill_support: '1' });
                }
                this.medarots.push(new Medarot( medarotDisplayId, `Medarot ${medarotIdNumber}`, teamId, teamConfig.baseSpeed + (Math.random() * 0.2), medalForMedarot, { isLeader: i === 0, color: teamConfig.color }, medarotPartsConfig,  this.partsData ));
            }
        });
    }

    setupUI() {
        this.dom.battlefield.innerHTML = '<div class="center-line"></div>';
        Object.entries(CONFIG.TEAMS).forEach(([teamId, teamConfig]) => {
            const panel = document.getElementById(`${teamId}InfoPanel`);
            panel.innerHTML = `<h2 class="text-xl font-bold mb-3 ${teamConfig.textColor}">${teamConfig.name}</h2>`;
        });
        this.medarots.forEach(medarot => {
            const idNum = parseInt(medarot.id.substring(1));
            const indexInTeam = (idNum - 1) % CONFIG.PLAYERS_PER_TEAM;
            const vPos = 25 + indexInTeam * 25;
            this.dom.battlefield.appendChild(medarot.createIconDOM(vPos));
            const panel = document.getElementById(`${medarot.team}InfoPanel`);
            panel.appendChild(medarot.createInfoPanelDOM());
            medarot.updateDisplay();
        });
    }

    bindEvents() {
        this.dom.startButton.addEventListener('click', () => this.start());
        this.dom.resetButton.addEventListener('click', () => this.reset());
        this.dom.modalConfirmButton.addEventListener('click', () => this.handleModalConfirm());
        this.dom.battleStartConfirmButton.addEventListener('click', () => this.handleBattleStartConfirm());
        this.dom.modalExecuteAttackButton.addEventListener('click', () => this.handleModalExecuteAttack());
        this.dom.modalCancelActionButton.addEventListener('click', () => this.handleModalCancelAction());
    }

    start() {
        if (this.phase !== 'IDLE') return;
        this.phase = 'INITIAL_SELECTION';
        this.medarots.forEach(m => { m.gauge = CONFIG.MAX_GAUGE; m.state = 'ready_select'; m.updateDisplay(); });
        this.dom.startButton.disabled = true; this.dom.startButton.textContent = "シミュレーション実行中...";
        this.dom.resetButton.style.display = "inline-block";
        this.resumeSimulation();
    }

    pauseSimulation() { clearInterval(this.simulationInterval); this.simulationInterval = null; }
    resumeSimulation() { if (this.simulationInterval) return; this.simulationInterval = setInterval(() => this.gameLoop(), CONFIG.UPDATE_INTERVAL); }
    reset() {
        this.pauseSimulation();
        this.phase = 'IDLE'; this.activeMedarot = null;
        this.hideModal();
        this.medarots.forEach(m => m.fullReset());
        this.setupUI();
        this.dom.startButton.disabled = false; this.dom.startButton.textContent = "シミュレーション開始";
        this.dom.resetButton.style.display = "none";
    }

    gameLoop() {
        if (this.activeMedarot || !['INITIAL_SELECTION', 'BATTLE'].includes(this.phase)) return;
        const medarotToExecute = this.medarots.find(m => m.state === 'ready_execute');
        if (medarotToExecute) return this.handleActionExecution(medarotToExecute);
        const medarotToSelect = this.medarots.find(m => m.isReadyForSelection());
        if (medarotToSelect) return this.handleActionSelection(medarotToSelect);
        if (this.phase === 'INITIAL_SELECTION') {
            if (this.medarots.every(m => m.state !== 'ready_select')) {
                this.phase = 'BATTLE_START_CONFIRM'; this.pauseSimulation(); this.showModal('battle_start_confirm');
            }
        } else if (this.phase === 'BATTLE') {
            this.medarots.forEach(m => { m.processTurn(); m.updateDisplay(); });
        }
    }

    handleActionExecution(medarot) {
        this.activeMedarot = medarot; this.pauseSimulation(); this.prepareAndShowExecutionModal(medarot);
    }

    handleActionSelection(medarot) {
        if (medarot.team === 'team2') {
            const availableAttackPartsForCpu = medarot.getAvailableAttackParts();
            const partKey = availableAttackPartsForCpu.length > 0 ? availableAttackPartsForCpu[Math.floor(Math.random() * availableAttackPartsForCpu.length)] : null;
            let initialTargetForCPU = null;
            if (partKey && medarot.parts[partKey]) {
                 initialTargetForCPU = this.findEnemyTarget(medarot, medarot.parts[partKey].category_jp === '格闘' ? 'Fight' : medarot.parts[partKey].sub_category_jp);
            }

            if (initialTargetForCPU && partKey) {
                medarot.selectAction(partKey);
                if (medarot.parts[partKey]) {
                    const selectedPartInfoCPU = medarot.parts[partKey];
                    if (selectedPartInfoCPU.category_jp === '射撃' && medarot.medal && medarot.medal.personality === 'ランダムターゲット') {
                        if (initialTargetForCPU.state !== 'broken') {
                            const availablePartsCPU = Object.entries(initialTargetForCPU.parts).filter(([pK, pV]) => pV && !pV.isBroken).map(([pK, _]) => pK);
                            if (availablePartsCPU.length > 0) {
                                const randomPartKeyCPU = availablePartsCPU[Math.floor(Math.random() * availablePartsCPU.length)];
                                medarot.currentTargetedEnemy = initialTargetForCPU;
                                medarot.currentTargetedPartKey = randomPartKeyCPU;
                                console.log(`${medarot.name} (CPU) random targeted ${initialTargetForCPU.name}'s ${randomPartKeyCPU} with ${selectedPartInfoCPU.name_jp}`);
                            } else {
                                medarot.currentTargetedEnemy = null; medarot.currentTargetedPartKey = null;
                                console.log(`${medarot.name} (CPU) found enemy ${initialTargetForCPU.name} for shooting, but it has no targetable parts.`);
                            }
                        } else {
                             medarot.currentTargetedEnemy = null; medarot.currentTargetedPartKey = null;
                             console.log(`${medarot.name} (CPU) initial target ${initialTargetForCPU.name} is broken. Cannot shoot.`);
                        }
                    }
                } else {
                     console.warn(`${medarot.name} (CPU) had partKey ${partKey}, but part info is missing after selectAction.`);
                     medarot.currentTargetedEnemy = null; medarot.currentTargetedPartKey = null;
                }
            } else {
                medarot.state = 'broken';
                medarot.currentTargetedEnemy = null; medarot.currentTargetedPartKey = null;
                console.log(`${medarot.name} (CPU) could not find a target or part. State set to broken/idle.`);
            }
        } else {
            this.activeMedarot = medarot; this.pauseSimulation(); this.showModal('selection', medarot);
        }
    }

    findEnemyTarget(attacker, actionType) {
        const enemies = this.medarots.filter(p => p.team !== attacker.team && p.state !== 'broken');
        if (enemies.length === 0) return null;
        const calculateDistance = (m1, m2) => {
            if (!m1.iconElement || !m1.iconElement.style.left || !m2.iconElement || !m2.iconElement.style.left) return Infinity;
            const x1 = parseFloat(m1.iconElement.style.left); const x2 = parseFloat(m2.iconElement.style.left);
            if (isNaN(x1) || isNaN(x2)) return Infinity;
            return Math.abs(x1 - x2);
        };
        const enemiesWithDistance = enemies.map(e => ({ medarot: e, distance: calculateDistance(attacker, e) })).filter(e => isFinite(e.distance));
        if (enemiesWithDistance.length === 0 && enemies.length > 0) return enemies.find(e => e.isLeader) || enemies[0];
        if (enemiesWithDistance.length === 0) return null;
        if (actionType === 'Shoot') { enemiesWithDistance.sort((a, b) => b.distance - a.distance); return enemiesWithDistance[0].medarot; }
        else if (actionType === 'Fight') { enemiesWithDistance.sort((a, b) => a.distance - b.distance); return enemiesWithDistance[0].medarot; }
        else { return enemies.find(e => e.isLeader) || enemies[0]; }
    }

    handlePartSelection(partKey) {
        if (!this.activeMedarot) return;
        const attacker = this.activeMedarot;
        attacker.selectAction(partKey);

        if (attacker.selectedPartKey && attacker.parts[attacker.selectedPartKey]) {
            const selectedPartInfo = attacker.parts[attacker.selectedPartKey];
            if (selectedPartInfo.category_jp === '射撃' && attacker.medal && attacker.medal.personality === 'ランダムターゲット') {
                const enemies = this.medarots.filter(m => m.team !== attacker.team && m.state !== 'broken');
                if (enemies.length > 0) {
                    const randomEnemy = enemies[Math.floor(Math.random() * enemies.length)];
                    const availableParts = Object.entries(randomEnemy.parts).filter(([pK, pV]) => pV && !pV.isBroken).map(([pK, _]) => pK);
                    if (availableParts.length > 0) {
                        const randomPartKeyOnEnemy = availableParts[Math.floor(Math.random() * availableParts.length)];
                        attacker.pendingTargetEnemy = randomEnemy;
                        attacker.pendingTargetPartKey = randomPartKeyOnEnemy;
                        console.log(`${attacker.name} (Player) pending target: ${randomEnemy.name}'s ${randomPartKeyOnEnemy} with ${selectedPartInfo.name_jp}`);

                        this.drawArrow(attacker, attacker.pendingTargetEnemy); // Draw arrow
                        this.dom.modalTitle.textContent = 'ターゲット確認';
                        this.dom.modalActorName.textContent = `${attacker.name}の${selectedPartInfo.name_jp}。ターゲット: ${attacker.pendingTargetEnemy.name}${attacker.pendingTargetPartKey ? 'の' + attacker.pendingTargetEnemy.parts[attacker.pendingTargetPartKey].name_jp : ''}`;

                        this.dom.partSelectionContainer.style.display = 'none';
                        this.dom.modalExecuteAttackButton.style.display = 'inline-block';
                        this.dom.modalCancelActionButton.style.display = 'inline-block';
                        return;
                    } else {
                        console.log(`${attacker.name} (Player) found enemy ${randomEnemy.name} but it has no targetable parts.`);
                        this.clearArrow(); // Clear arrow if no target
                         // Revert to part selection or show error
                        this.dom.modalExecuteAttackButton.style.display = 'none';
                        this.dom.modalCancelActionButton.style.display = 'none';
                        this.dom.partSelectionContainer.style.display = 'flex';
                        this.dom.modalTitle.textContent = '行動選択';
                        this.dom.modalActorName.textContent = `${attacker.name}のターン。ターゲット選択失敗。`;
                        return; // Stay in modal, part selection re-shown
                    }
                } else {
                    console.log(`${attacker.name} (Player) found no enemies to target.`);
                    this.clearArrow(); // Clear arrow if no target
                    this.dom.modalExecuteAttackButton.style.display = 'none';
                    this.dom.modalCancelActionButton.style.display = 'none';
                    this.dom.partSelectionContainer.style.display = 'flex';
                    this.dom.modalTitle.textContent = '行動選択';
                    this.dom.modalActorName.textContent = `${attacker.name}のターン。ターゲットなし。`;
                    return; // Stay in modal, part selection re-shown
                }
            }
        } else if (attacker.selectedPartKey) {
            console.warn(`${attacker.name} (Player) has selectedPartKey ${attacker.selectedPartKey}, but part info is missing.`);
        }
        // If not shooting with random target, or if no target found and not handled above, proceed as before
        this.activeMedarot = null;
        this.hideModal(); // This will also call clearArrow
        this.resumeSimulation();
    }

    handleModalExecuteAttack() {
        if (!this.activeMedarot || !this.activeMedarot.pendingTargetEnemy) {
            console.warn("ExecuteAttack called without activeMedarot or pending target.");
            this.hideModal(); // This will also call clearArrow
            this.resumeSimulation();
            return;
        }
        const attacker = this.activeMedarot;
        attacker.currentTargetedEnemy = attacker.pendingTargetEnemy;
        attacker.currentTargetedPartKey = attacker.pendingTargetPartKey;

        attacker.pendingTargetEnemy = null;
        attacker.pendingTargetPartKey = null;

        this.activeMedarot = null;
        this.hideModal();
        this.resumeSimulation();
    }

    handleModalCancelAction() {
        if (this.activeMedarot) {
            this.activeMedarot.pendingTargetEnemy = null;
            this.activeMedarot.pendingTargetPartKey = null;
            this.activeMedarot.currentTargetedEnemy = null;
            this.activeMedarot.currentTargetedPartKey = null;
        }
        this.clearArrow();

        this.dom.modalExecuteAttackButton.style.display = 'none';
        this.dom.modalCancelActionButton.style.display = 'none';
        this.dom.partSelectionContainer.style.display = 'flex';
        this.dom.modalTitle.textContent = '行動選択';
        if (this.activeMedarot) {
             this.dom.modalActorName.textContent = `${this.activeMedarot.name}のターン。`;
        }
    }

    drawArrow(attackerMedarot, targetMedarot) {
        if (!attackerMedarot || !targetMedarot || !attackerMedarot.iconElement || !targetMedarot.iconElement || !this.dom.aimingArrow) {
            this.clearArrow();
            return;
        }
        const attackerRect = attackerMedarot.iconElement.getBoundingClientRect();
        const targetRect = targetMedarot.iconElement.getBoundingClientRect();
        const startX = attackerRect.left + (attackerRect.width / 2) + window.scrollX;
        const startY = attackerRect.top + (attackerRect.height / 2) + window.scrollY;
        const endX = targetRect.left + (targetRect.width / 2) + window.scrollX;
        const endY = targetRect.top + (targetRect.height / 2) + window.scrollY;
        const deltaX = endX - startX;
        const deltaY = endY - startY;
        const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
        const angle = Math.atan2(deltaY, deltaX) * (180 / Math.PI);
        const arrow = this.dom.aimingArrow;
        arrow.style.width = `${distance}px`;
        arrow.style.left = `${startX}px`;
        arrow.style.top = `${startY}px`;
        arrow.style.transform = `rotate(${angle}deg)`;
        arrow.style.display = 'block';
    }

    clearArrow() {
        if (this.dom.aimingArrow) {
            this.dom.aimingArrow.style.display = 'none';
        }
    }

    handleBattleStartConfirm() {
        this.phase = 'BATTLE';
        this.medarots.forEach(m => m.processTurn());
        this.hideModal();
        this.resumeSimulation();
    }

    handleModalConfirm() {
        if (this.phase === 'GAME_OVER') return this.reset();
        if (!this.activeMedarot) return;

        const attacker = this.activeMedarot;
        if (attacker.preparedAttack) {
            const { target, partKey, damage } = attacker.preparedAttack;
            if (target.applyDamage(damage, partKey) && target.isLeader) {
                this.phase = 'GAME_OVER';
                this.showModal('game_over', null, { winningTeam: attacker.team });
                this.pauseSimulation();
                return;
            }
        }
        attacker.startCooldown();
        this.activeMedarot = null;
        this.hideModal();
        this.resumeSimulation();
    }

    prepareAndShowExecutionModal(medarot) {
        const attackingPart = medarot.parts[medarot.selectedPartKey];
        if (!attackingPart) {
            console.error(`${medarot.name} has no valid selected part for execution. PartKey: ${medarot.selectedPartKey}`);
            medarot.startCooldown();
            return;
        }
        const attackingPartCategory = attackingPart.category_jp;

        let targetEnemyMedarot = null;
        let targetPartKeyForAttack = null;

        if (attackingPartCategory === '射撃') {
            if (medarot.currentTargetedEnemy && medarot.currentTargetedEnemy.state !== 'broken' &&
                medarot.currentTargetedPartKey &&
                medarot.currentTargetedEnemy.parts[medarot.currentTargetedPartKey] &&
                !medarot.currentTargetedEnemy.parts[medarot.currentTargetedPartKey].isBroken) {

                targetEnemyMedarot = medarot.currentTargetedEnemy;
                targetPartKeyForAttack = medarot.currentTargetedPartKey;
                console.log(`${medarot.name} executing shooting action against pre-selected target: ${targetEnemyMedarot.name}'s ${targetPartKeyForAttack}`);
            } else {
                console.log(`${medarot.name}'s pre-selected shooting target is invalid or missing. Using default targeting for shooting.`);
                targetEnemyMedarot = this.findEnemyTarget(medarot, attackingPart.sub_category_jp);
                if (targetEnemyMedarot) {
                    const availableTargetParts = Object.keys(targetEnemyMedarot.parts).filter(key => targetEnemyMedarot.parts[key] && !targetEnemyMedarot.parts[key].isBroken);
                    if (availableTargetParts.length > 0) {
                        targetPartKeyForAttack = availableTargetParts[Math.floor(Math.random() * availableTargetParts.length)];
                    }
                }
            }
        } else if (attackingPartCategory === '格闘') {
            targetEnemyMedarot = this.findEnemyTarget(medarot, 'Fight');
            if (targetEnemyMedarot && targetEnemyMedarot.state !== 'broken') {
                const availableTargetParts = Object.keys(targetEnemyMedarot.parts)
                                             .filter(key => targetEnemyMedarot.parts[key] && !targetEnemyMedarot.parts[key].isBroken);
                if (availableTargetParts.length > 0) {
                    targetPartKeyForAttack = availableTargetParts[Math.floor(Math.random() * availableTargetParts.length)];
                    console.log(`${medarot.name} executing fighting action against closest target: ${targetEnemyMedarot.name}, hitting part ${targetPartKeyForAttack}`);
                } else {
                     console.log(`${medarot.name} found closest enemy ${targetEnemyMedarot.name} for fighting, but it has no targetable parts.`);
                }
            } else {
                console.log(`${medarot.name} found no valid closest enemy for fighting action.`);
            }
        } else {
            targetEnemyMedarot = this.findEnemyTarget(medarot, attackingPart.sub_category_jp);
            if (targetEnemyMedarot) {
                const availableTargetParts = Object.keys(targetEnemyMedarot.parts).filter(key => targetEnemyMedarot.parts[key] && !targetEnemyMedarot.parts[key].isBroken);
                if (availableTargetParts.length > 0) {
                    targetPartKeyForAttack = availableTargetParts[Math.floor(Math.random() * availableTargetParts.length)];
                }
            }
        }

        if (!targetEnemyMedarot || !targetPartKeyForAttack) {
            console.log(`${medarot.name} could not determine a valid target/part for action ${attackingPart.name_jp}. Starting cooldown.`);
            medarot.startCooldown();
            return;
        }

        medarot.preparedAttack = {
            target: targetEnemyMedarot,
            partKey: targetPartKeyForAttack,
            damage: CONFIG.BASE_DAMAGE
        };

        this.showModal('execution', medarot);
    }

    showModal(type, medarot = null, data = {}) {
        const modal = this.dom.modal;
        const title = this.dom.modalTitle;
        const actorName = this.dom.modalActorName;
        const partContainer = this.dom.partSelectionContainer;
        const confirmBtn = this.dom.modalConfirmButton;
        const startBtn = this.dom.battleStartConfirmButton;

        this.dom.partSelectionContainer.style.display = 'none';
        this.dom.modalConfirmButton.style.display = 'none';
        this.dom.battleStartConfirmButton.style.display = 'none';
        this.dom.modalExecuteAttackButton.style.display = 'none';
        this.dom.modalCancelActionButton.style.display = 'none';

        modal.className = 'modal';

        switch (type) {
            case 'selection':
                title.textContent = '行動選択';
                actorName.textContent = `${medarot.name}のターン。`;
                partContainer.innerHTML = '';
                medarot.getAvailableAttackParts().forEach(partKey => {
                    const part = medarot.parts[partKey];
                    const button = document.createElement('button');
                    button.className = 'part-action-button';
                    button.textContent = `${part.name_jp} (${part.sub_category_jp})`;
                    button.onclick = () => this.handlePartSelection(partKey);
                    partContainer.appendChild(button);
                });
                partContainer.style.display = 'flex';
                break;
            case 'execution':
                title.textContent = '攻撃実行！';
                const attackerPart = medarot.parts[medarot.selectedPartKey];

                if (!attackerPart) {
                    console.error("Attacker's selected part not found!", medarot);
                    actorName.innerHTML = `${medarot.name}の攻撃！詳細は不明です。`;
                } else {
                    const partCategory = attackerPart.category_jp || '不明カテゴリ';
                    const partSubCategory = attackerPart.sub_category_jp || '不明サブカテゴリ';
                    const partName = attackerPart.name_jp || '不明パーツ';

                    const { target, partKey: targetPartKey, damage } = medarot.preparedAttack;
                    const targetPartName = target.parts && target.parts[targetPartKey] ? target.parts[targetPartKey].name_jp : '不明部位';

                    actorName.innerHTML = `「${medarot.name}の${partCategory}！ ${partSubCategory}行動${partName}！」<br><small>（${target.name}の${targetPartName}に ${damage} ダメージ！）</small>`;
                }

                confirmBtn.style.display = 'inline-block';
                confirmBtn.textContent = '了解';
                break;
            case 'battle_start_confirm':
                title.textContent = '戦闘開始！';
                actorName.textContent = '';
                startBtn.style.display = 'inline-block';
                break;
            case 'game_over':
                title.textContent = `${CONFIG.TEAMS[data.winningTeam].name} の勝利！`;
                actorName.textContent = 'ロボトル終了！';
                confirmBtn.style.display = 'inline-block';
                confirmBtn.textContent = 'リセット';
                modal.classList.add('game-over-modal');
                break;
        }
        modal.classList.remove('hidden');
    }

    hideModal() {
        this.clearArrow(); // Clear arrow when modal hides
        this.dom.modal.classList.add('hidden');
    }
}

document.addEventListener('DOMContentLoaded', async () => {
    const game = new GameManager();
    await game.init();
});
