// 1. 中央集権的な設定 (Centralized configuration)
const CONFIG = {
    UI: { // New sub-object for UI related constants
        TEAM1_EXECUTE_X: 0.45,
        TEAM2_EXECUTE_X: 0.55,
        TEAM1_HOME_X: 0.0,
        TEAM2_HOME_X: 1.0,
        // New UI constants for icon positioning
        MEDAROT_ICON_VERTICAL_INITIAL_OFFSET: 25,
        MEDAROT_ICON_VERTICAL_SPACING: 25
    },
    RULES: { // New sub-object for game rules and balance
        MAX_GAUGE: 100,
        UPDATE_INTERVAL: 50,
        PLAYERS_PER_TEAM: 3,
        PART_HP_BASE: 50,
        LEGS_HP_BONUS: 10,
        BASE_DAMAGE: 20,
        TEAM1_BASE_SPEED: 1.0, // Moved and renamed from TEAMS.team1.baseSpeed
        TEAM2_BASE_SPEED: 0.9,  // Moved and renamed from TEAMS.team2.baseSpeed
        // New RULES constants for game logic
        MINIMUM_DAMAGE: 1,
        PROPULSION_BONUS_FACTOR: 0.05,
        SPEED_RANDOM_FACTOR: 0.2
    },
    TEAMS: { // TEAMS now only contains display/theming info
        team1: { name: 'Team 1', color: '#63b3ed', textColor: 'text-blue-300' },
        team2: { name: 'Team 2', color: '#f56565', textColor: 'text-red-300' }
    }
};

// メダルクラス (Medal Class)
class Medal {
    constructor(data) {
        this.id = data.id;
        this.name = data.name_jp;
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
        this.loggedConfigPositions = false;

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
                        id: partData.id,
                        name_jp: partData.name_jp,
                        category_jp: partData.category_jp,
                        sub_category_jp: partData.sub_category_jp,
                        slot: slotKey,
                        hp: parseInt(partData.base_hp) || CONFIG.RULES.PART_HP_BASE,
                        maxHp: parseInt(partData.base_hp) || CONFIG.RULES.PART_HP_BASE,
                        charge: parseInt(partData.charge) || 0,
                        cooldown: parseInt(partData.cooldown) || 0,
                        isBroken: false
                    };

                    if (slotKey === 'legs') {
                        this.parts[slotKey].hp += CONFIG.RULES.LEGS_HP_BONUS;
                        this.parts[slotKey].maxHp += CONFIG.RULES.LEGS_HP_BONUS;
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
        if (this.parts[partKey]) {
            this.selectedActionType = this.parts[partKey].sub_category_jp;
            this.currentActionCharge = this.parts[partKey].charge;
            this.currentActionCooldown = this.parts[partKey].cooldown;
        } else {
            console.error(`Selected partKey ${partKey} does not exist on Medarot ${this.id}. Cannot select action.`);
            this.selectedActionType = null;
            this.currentActionCharge = null;
            this.currentActionCooldown = null;
            return;
        }
        this.gauge = 0;
        this.state = 'action_charging';
    }

    getAvailableAttackParts() {
        return Object.entries(this.parts)
            .filter(([key, part]) => part && !part.isBroken && ['head', 'rightArm', 'leftArm'].includes(key))
            .map(([key, _]) => key);
    }

    isReadyForSelection() {
        return this.state === 'ready_select';
    }

    applyDamage(damage, partKey) {
        const part = this.parts[partKey];
        if (!part) return false;

        let effectiveDamage = damage;
        if (this.legDefenseParam && this.legDefenseParam > 0) {
            effectiveDamage = Math.max(CONFIG.RULES.MINIMUM_DAMAGE, damage - (this.legDefenseParam || 0));
        }
        part.hp = Math.max(0, part.hp - effectiveDamage);

        if (part.hp === 0) {
            part.isBroken = true;
            if (partKey === 'head') {
                this.state = 'broken';
                return true;
            }
        }
        return false;
    }

    processTurn() {
        if (this.parts.head && this.parts.head.isBroken && this.state !== 'broken') this.state = 'broken';

        const statesToPause = ['ready_select', 'ready_execute', 'broken'];
        if (statesToPause.includes(this.state)) return;

        const baseChargeRate = this.speed;
        const propulsionBonus = (this.legPropulsion || 0) * CONFIG.RULES.PROPULSION_BONUS_FACTOR;
        this.gauge += baseChargeRate + propulsionBonus;

        if (this.state === 'idle_charging') {
            if (this.gauge >= CONFIG.RULES.MAX_GAUGE) {
                this.gauge = CONFIG.RULES.MAX_GAUGE;
                this.state = 'ready_select';
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
                            this.startCooldown();
                            return;
                        }
                    }
                } else if (this.selectedPartKey) {
                    console.error(`${this.name} has an invalid selectedPartKey: ${this.selectedPartKey} during charge completion. Forcing cooldown.`);
                    this.startCooldown();
                    return;
                }
                this.state = 'ready_execute';
            }
        } else if (this.state === 'action_cooldown') {
            if (this.gauge >= this.currentActionCooldown) {
                this.gauge = this.currentActionCooldown;
                this.state = 'ready_select';
            }
        }
    }

    createIconDOM(verticalPosition) {
        const icon = document.createElement('div');
        icon.id = `${this.id}-icon`;
        icon.className = 'player-icon';
        icon.style.backgroundColor = this.color;
        icon.style.top = `${verticalPosition}%`;
        icon.style.transform = 'translate(-50%, -50%)';
        icon.textContent = this.name.substring(this.name.length - 1);
        this.iconElement = icon;
        return icon;
    }

    createInfoPanelDOM() {
        const info = document.createElement('div');
        info.className = 'player-info';
        const teamConfig = CONFIG.TEAMS[this.team];

        const partSlotNamesJP = { head: '頭部', rightArm: '右腕', leftArm: '左腕', legs: '脚部' };
        let partsHTML = '';
        Object.keys(this.parts).forEach(key => {
             if(!this.parts[key]) {
                console.warn(`[DEBUG] createInfoPanelDOM for ${this.id}: Part data for key '${key}' is missing.`);
                partsHTML += `<div class="part-info-wrapper" id="${this.id}-${key}-wrapper">Part data missing</div>`;
                return;
            }
            partsHTML += `
                <div class="part-info-wrapper" id="${this.id}-${key}-wrapper">
                    <div class="part-name-display" id="${this.id}-${key}-name"></div>
                    <div class="part-hp-section">
                        <div class="part-hp-bar-container" id="${this.id}-${key}-bar-container">
                            <div class="part-hp-bar" id="${this.id}-${key}-bar"></div>
                        </div>
                        <span class="part-hp-numeric" id="${this.id}-${key}-hp-numeric"></span>
                    </div>
                </div>
            `;
        });

        info.innerHTML = `
            <div class="player-name ${teamConfig.textColor}">${this.name} ${this.isLeader ? '(L)' : ''}</div>
            <div class="parts-container">${partsHTML}</div>
        `;

        Object.entries(this.parts).forEach(([key, part]) => {
            if(!part) return;
            const wrapperElement = info.querySelector(`#${this.id}-${key}-wrapper`);
            const nameDisplayElement = info.querySelector(`#${this.id}-${key}-name`);
            const barElement = info.querySelector(`#${this.id}-${key}-bar`);
            const numericHpElement = info.querySelector(`#${this.id}-${key}-hp-numeric`);

            if (nameDisplayElement) {
                 nameDisplayElement.textContent = `${partSlotNamesJP[key] || key}: ${part.name_jp || 'N/A'}`;
            }
            if (numericHpElement) {
                numericHpElement.textContent = `(${part.hp}/${part.maxHp})`;
            }

            this.partDOMElements[key] = {
                wrapper: wrapperElement, nameDisplay: nameDisplayElement,
                bar: barElement, numericHp: numericHpElement
            };
        });
        return info;
    }

    updateDisplay() {
        console.log(`[DEBUG] updateDisplay called for: ${this.id}`);
        this.updatePosition();
        this.updateInfoPanel();
    }

    updatePosition() {
        if (!this.iconElement) {
            console.warn(`[DEBUG] updatePosition for ${this.id}: iconElement is null. Skipping.`);
            return;
        }
        console.log(`[DEBUG] updatePosition for ${this.id} (Team: ${this.team}, State: ${this.state}, Gauge: ${this.gauge})`);

        if (!this.loggedConfigPositions && (this.id === 'p1' || this.id === 'p4')) {
            console.log(`[DEBUG] CONFIG relevant for positioning: T1_HOME_X=${CONFIG.UI.TEAM1_HOME_X}, T1_EXEC_X=${CONFIG.UI.TEAM1_EXECUTE_X}, T2_HOME_X=${CONFIG.UI.TEAM2_HOME_X}, T2_EXEC_X=${CONFIG.UI.TEAM2_EXECUTE_X}`);
            this.loggedConfigPositions = true;
        }

        let currentMaxGauge = CONFIG.RULES.MAX_GAUGE;
        if (this.state === 'action_charging' && this.currentActionCharge != null) {
            currentMaxGauge = this.currentActionCharge;
        } else if (this.state === 'action_cooldown' && this.currentActionCooldown != null) {
            currentMaxGauge = this.currentActionCooldown;
        } else if (this.state === 'ready_execute' && this.currentActionCharge != null) {
            currentMaxGauge = this.currentActionCharge;
        }

        let progress = 0;
        if (currentMaxGauge && currentMaxGauge > 0) {
            progress = Math.min(1, (this.gauge || 0) / currentMaxGauge);
        } else if ((this.gauge || 0) === 0 && currentMaxGauge === 0) {
            progress = 1;
        }

        let positionXRatio;
        switch (this.state) {
            case 'action_charging':
                if (this.team === 'team1') {
                    positionXRatio = CONFIG.UI.TEAM1_HOME_X + progress * (CONFIG.UI.TEAM1_EXECUTE_X - CONFIG.UI.TEAM1_HOME_X);
                } else {
                    positionXRatio = CONFIG.UI.TEAM2_HOME_X - progress * (CONFIG.UI.TEAM2_HOME_X - CONFIG.UI.TEAM2_EXECUTE_X);
                }
                break;
            case 'idle_charging':
            case 'action_cooldown':
                if (this.team === 'team1') {
                    positionXRatio = CONFIG.UI.TEAM1_EXECUTE_X - progress * (CONFIG.UI.TEAM1_EXECUTE_X - CONFIG.UI.TEAM1_HOME_X);
                } else {
                    positionXRatio = CONFIG.UI.TEAM2_EXECUTE_X + progress * (CONFIG.UI.TEAM2_HOME_X - CONFIG.UI.TEAM2_EXECUTE_X);
                }
                break;
            case 'ready_execute':
                positionXRatio = (this.team === 'team1') ? CONFIG.UI.TEAM1_EXECUTE_X : CONFIG.UI.TEAM2_EXECUTE_X;
                break;
            case 'ready_select':
                positionXRatio = (this.team === 'team1') ? CONFIG.UI.TEAM1_HOME_X : CONFIG.UI.TEAM2_HOME_X;
                break;
            case 'broken':
                if (this.iconElement.style.left && this.iconElement.style.left !== '') {
                    positionXRatio = parseFloat(this.iconElement.style.left) / 100;
                } else {
                    positionXRatio = (this.team === 'team1') ? CONFIG.UI.TEAM1_HOME_X : CONFIG.UI.TEAM2_HOME_X;
                }
                break;
            default:
                console.warn(`[DEBUG] Unknown state for ${this.id}: ${this.state}. Defaulting to home position.`);
                positionXRatio = (this.team === 'team1') ? CONFIG.UI.TEAM1_HOME_X : CONFIG.UI.TEAM2_HOME_X;
                break;
        }

        if (typeof positionXRatio !== 'number' || isNaN(positionXRatio)) {
            console.error(`[DEBUG] Invalid positionXRatio calculated for ${this.id}: ${positionXRatio}. Defaulting to home.`);
            positionXRatio = (this.team === 'team1') ? CONFIG.UI.TEAM1_HOME_X : CONFIG.UI.TEAM2_HOME_X;
        }

        console.log(`[DEBUG] For ${this.id} (State: ${this.state}), calculated positionXRatio: ${positionXRatio.toFixed(3)}`);
        this.iconElement.style.left = `${positionXRatio * 100}%`;

        this.iconElement.classList.toggle('ready-select', this.isReadyForSelection());
        this.iconElement.classList.toggle('ready-execute', this.state === 'ready_execute');
        this.iconElement.classList.toggle('broken', this.state === 'broken');
    }

    updateInfoPanel() {
        Object.entries(this.parts).forEach(([key, part]) => {
            const elements = this.partDOMElements[key];
            if (!elements || !elements.bar || !elements.numericHp || !elements.wrapper) return;
            if (!part || typeof part.hp !== 'number' || typeof part.maxHp !== 'number' || part.maxHp === 0) {
                elements.bar.style.width = '0%';
                elements.numericHp.textContent = `(ERR/ERR)`;
                elements.wrapper.classList.toggle('broken', true);
                if(elements.bar) elements.bar.style.backgroundColor = '#4a5568';
                return;
            }
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

// ゲーム管理クラス (Game Manager Class)
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
            modalExecuteAttackButton: document.getElementById('modalExecuteAttackButton'),
            modalCancelActionButton: document.getElementById('modalCancelActionButton'),
            battleStartConfirmButton: document.getElementById('battleStartConfirmButton'),
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
            for (let i = 0; i < CONFIG.RULES.PLAYERS_PER_TEAM; i++) {
                const medarotIdNumber = teamIndex * CONFIG.RULES.PLAYERS_PER_TEAM + i + 1;
                const medarotDisplayId = `p${medarotIdNumber}`;
                let medarotPartsConfig; let selectedMedalDataForCurrentMedarot;

                const currentTeamBaseSpeed = teamId === 'team1' ? CONFIG.RULES.TEAM1_BASE_SPEED : CONFIG.RULES.TEAM2_BASE_SPEED;

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
                            const medalIndex = (teamIndex * CONFIG.RULES.PLAYERS_PER_TEAM + i) % this.medalsData.length;
                            selectedMedalDataForCurrentMedarot = this.medalsData.length > 0 ? this.medalsData[medalIndex % this.medalsData.length] : null;
                        }
                    } else {
                        console.warn(`METABEE_SET for ${medarotDisplayId} is incomplete. Falling back to default loadout for p1.`);
                        const loadoutIndex = 0;
                        medarotPartsConfig = defaultLoadouts[loadoutIndex];
                        const medalIndex = (teamIndex * CONFIG.RULES.PLAYERS_PER_TEAM + i) % this.medalsData.length;
                        selectedMedalDataForCurrentMedarot = this.medalsData.length > 0 ? this.medalsData[medalIndex % this.medalsData.length] : null;
                    }
                } else {
                    const loadoutIndex = teamIndex * CONFIG.RULES.PLAYERS_PER_TEAM + i;
                    medarotPartsConfig = defaultLoadouts[loadoutIndex % defaultLoadouts.length];
                    const medalIndex = (teamIndex * CONFIG.RULES.PLAYERS_PER_TEAM + i) % this.medalsData.length;
                    selectedMedalDataForCurrentMedarot = this.medalsData.length > 0 ? this.medalsData[medalIndex % this.medalsData.length] : null;
                }
                let medalForMedarot;
                if (selectedMedalDataForCurrentMedarot) medalForMedarot = new Medal(selectedMedalDataForCurrentMedarot);
                else {
                    console.warn(`No medal data found for Medarot ${medarotDisplayId}. Using a default fallback medal.`);
                    medalForMedarot = new Medal({ id: 'M_FALLBACK', name_jp: 'フォールバックメダル', personality_jp: 'ランダムターゲット', medaforce_jp: 'なし', attribute_jp: '無', skill_shoot: '1', skill_fight: '1', skill_scan: '1', skill_support: '1' });
                }
                this.medarots.push(new Medarot( medarotDisplayId, `Medarot ${medarotIdNumber}`, teamId, currentTeamBaseSpeed + (Math.random() * CONFIG.RULES.SPEED_RANDOM_FACTOR), medalForMedarot, { isLeader: i === 0, color: teamConfig.color }, medarotPartsConfig,  this.partsData ));
            }
        });
    }

    setupUI() {
        this.dom.battlefield.innerHTML = `
            <div class="center-line"></div>
            <div id="team1-execute-line" class="execute-line"></div>
            <div id="team2-execute-line" class="execute-line"></div>
        `;
        Object.entries(CONFIG.TEAMS).forEach(([teamId, teamConfig]) => {
            const panel = document.getElementById(`${teamId}InfoPanel`);
            panel.innerHTML = `<h2 class="text-xl font-bold mb-3 ${teamConfig.textColor}">${teamConfig.name}</h2>`;
        });
        this.medarots.forEach(medarot => {
            const idNum = parseInt(medarot.id.substring(1));
            const indexInTeam = (idNum - 1) % CONFIG.RULES.PLAYERS_PER_TEAM;
            const vPos = CONFIG.UI.MEDAROT_ICON_VERTICAL_INITIAL_OFFSET + indexInTeam * CONFIG.UI.MEDAROT_ICON_VERTICAL_SPACING;
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
        this.medarots.forEach(m => { m.gauge = CONFIG.RULES.MAX_GAUGE; m.state = 'ready_select'; m.updateDisplay(); });
        this.dom.startButton.disabled = true; this.dom.startButton.textContent = "シミュレーション実行中...";
        this.dom.resetButton.style.display = "inline-block";
        this.resumeSimulation();
    }

    pauseSimulation() { clearInterval(this.simulationInterval); this.simulationInterval = null; }
    resumeSimulation() {
        if (this.simulationInterval) return;
        this.simulationInterval = setInterval(() => this.gameLoop(), CONFIG.RULES.UPDATE_INTERVAL);
    }

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
        if (this.phase === 'INITIAL_SELECTION' && this.medarots.every(m => m.state !== 'ready_select')) {
            this.phase = 'BATTLE_START_CONFIRM'; this.pauseSimulation(); this.showModal('battle_start_confirm');
        } else if (this.phase === 'BATTLE') {
            this.medarots.forEach(m => { m.processTurn(); m.updateDisplay(); });
        }
    }

    handleActionExecution(medarot) {
        this.activeMedarot = medarot; this.pauseSimulation(); this.prepareAndShowExecutionModal(medarot);
    }

    // Helper method to get a random part key from a given enemy
    _selectRandomPartOnEnemy(enemyMedarot) {
        if (!enemyMedarot || !enemyMedarot.parts) return null;
        const availableParts = Object.entries(enemyMedarot.parts)
                                 .filter(([pKey, pVal]) => pVal && !pVal.isBroken)
                                 .map(([pKey, _]) => pKey);
        if (availableParts.length === 0) return null;
        return availableParts[Math.floor(Math.random() * availableParts.length)];
    }

    // Helper method to select a random enemy and a random part on that enemy
    _selectRandomTargetEnemyAndPart(attacker) {
        if (!attacker) return null;

        const enemies = this.medarots.filter(m => m.team !== attacker.team && m.state !== 'broken');
        if (enemies.length === 0) {
            console.log(`[TARGETING_HELPER] ${attacker.name} found no enemies.`);
            return null;
        }

        const randomEnemy = enemies[Math.floor(Math.random() * enemies.length)];
        const randomPartKeyOnEnemy = this._selectRandomPartOnEnemy(randomEnemy);

        if (!randomPartKeyOnEnemy) {
            console.log(`[TARGETING_HELPER] ${attacker.name} found enemy ${randomEnemy.name} but it has no targetable parts.`);
            return { targetEnemy: randomEnemy, targetPartKey: null }; // Return enemy but no part
        }

        console.log(`[TARGETING_HELPER] ${attacker.name} auto-selected target: ${randomEnemy.name}'s ${randomPartKeyOnEnemy}`);
        return { targetEnemy: randomEnemy, targetPartKey: randomPartKeyOnEnemy };
    }

    handleActionSelection(medarot) {
        if (medarot.team === 'team2') {
            const availableAttackPartsKeys = medarot.getAvailableAttackParts();
            if (availableAttackPartsKeys.length === 0) {
                medarot.state = 'broken';
                medarot.currentTargetedEnemy = null; medarot.currentTargetedPartKey = null;
                return;
            }
            const partKey = availableAttackPartsKeys[Math.floor(Math.random() * availableAttackPartsKeys.length)];
            const attackingPartForCPU = medarot.parts[partKey];

            let initialTargetForCPU = null;
            if (attackingPartForCPU && attackingPartForCPU.category_jp === '格闘') {
                initialTargetForCPU = this.findEnemyTarget(medarot, 'Fight');
            } else if (attackingPartForCPU) {
                initialTargetForCPU = this.findEnemyTarget(medarot, attackingPartForCPU.sub_category_jp);
            }

            if (initialTargetForCPU && partKey) {
                medarot.selectAction(partKey);
                if (attackingPartForCPU.category_jp === '射撃' && medarot.medal && medarot.medal.personality === 'ランダムターゲット') {
                    if (initialTargetForCPU.state !== 'broken') {
                        const randomPartKeyCPU = this._selectRandomPartOnEnemy(initialTargetForCPU); // Use new helper
                        if (randomPartKeyCPU) {
                            medarot.currentTargetedEnemy = initialTargetForCPU;
                            medarot.currentTargetedPartKey = randomPartKeyCPU;
                            console.log(`${medarot.name} (CPU) targeted ${initialTargetForCPU.name}'s ${randomPartKeyCPU} with ${attackingPartForCPU.name_jp}`);
                        } else {
                            medarot.currentTargetedEnemy = null; medarot.currentTargetedPartKey = null;
                            console.log(`${medarot.name} (CPU) found enemy ${initialTargetForCPU.name} for shooting, but it has no targetable parts.`);
                        }
                    } else {
                         medarot.currentTargetedEnemy = null; medarot.currentTargetedPartKey = null;
                         console.log(`${medarot.name} (CPU) found no valid initial target for shooting or target is broken.`);
                    }
                }
            } else {
                medarot.state = 'broken';
                medarot.currentTargetedEnemy = null;
                medarot.currentTargetedPartKey = null;
                console.log(`${medarot.name} (CPU) could not find a target or part. State set to broken/idle.`);
            }
        } else {
            this.activeMedarot = medarot;
            this.pauseSimulation();
            this.showModal('selection', medarot);
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
        if (enemiesWithDistance.length === 0) return enemies.find(e => e.isLeader) || enemies[0] || null;
        if (actionType === 'Fight') {
            enemiesWithDistance.sort((a, b) => a.distance - b.distance);
            return enemiesWithDistance[0].medarot;
        } else if (actionType === 'Shoot' || actionType === '狙い撃ち' || actionType === '撃つ') {
            enemiesWithDistance.sort((a, b) => b.distance - a.distance);
            return enemiesWithDistance[0].medarot;
        } else {
            return enemies.find(e => e.isLeader) || enemies[0];
        }
    }

    handlePartSelection(partKey) {
        if (!this.activeMedarot) return;
        const attacker = this.activeMedarot;
        attacker.selectAction(partKey);

        if (attacker.selectedPartKey && attacker.parts[attacker.selectedPartKey]) {
            const selectedPartInfo = attacker.parts[attacker.selectedPartKey];
            if (selectedPartInfo.category_jp === '射撃' && attacker.medal && attacker.medal.personality === 'ランダムターゲット') {
                const targetSelection = this._selectRandomTargetEnemyAndPart(attacker); // Use new helper

                if (targetSelection && targetSelection.targetEnemy && targetSelection.targetPartKey) {
                    attacker.pendingTargetEnemy = targetSelection.targetEnemy;
                    attacker.pendingTargetPartKey = targetSelection.targetPartKey;
                    // console.log is already in the helper, but this adds player context
                    console.log(`${attacker.name} (Player) pending target: ${attacker.pendingTargetEnemy.name}'s ${attacker.pendingTargetPartKey} with ${selectedPartInfo.name_jp}`);

                    if (this.dom.aimingArrow) this.drawArrow(attacker, attacker.pendingTargetEnemy);
                    this.dom.modalTitle.textContent = 'ターゲット確認';
                    const targetPartName = attacker.pendingTargetEnemy.parts[attacker.pendingTargetPartKey] ? attacker.pendingTargetEnemy.parts[attacker.pendingTargetPartKey].name_jp : '不明部位';
                    this.dom.modalActorName.textContent = `${attacker.name}の${selectedPartInfo.name_jp}。ターゲット: ${attacker.pendingTargetEnemy.name}の${targetPartName}`;
                    this.dom.partSelectionContainer.style.display = 'none';
                    this.dom.modalExecuteAttackButton.style.display = 'inline-block';
                    this.dom.modalCancelActionButton.style.display = 'inline-block';
                    return;
                } else if (targetSelection && targetSelection.targetEnemy) { // Enemy found, but no parts
                    attacker.pendingTargetEnemy = null; attacker.pendingTargetPartKey = null;
                    console.log(`${attacker.name} (Player) found enemy ${targetSelection.targetEnemy.name} but it has no targetable parts (via helper).`);
                    if (this.dom.aimingArrow) this.clearArrow();
                    this.dom.modalActorName.textContent = `${targetSelection.targetEnemy.name}には狙えるパーツがありません。別の行動を選択してください。`;
                    this.dom.partSelectionContainer.style.display = 'flex';
                    this.dom.modalExecuteAttackButton.style.display = 'none';
                    this.dom.modalCancelActionButton.style.display = 'none';
                    return;
                } else { // No enemy found
                    attacker.pendingTargetEnemy = null; attacker.pendingTargetPartKey = null;
                    console.log(`${attacker.name} (Player) found no enemies to target (via helper).`);
                    if (this.dom.aimingArrow) this.clearArrow();
                    this.dom.modalActorName.textContent = '狙える敵がいません。';
                    this.dom.partSelectionContainer.style.display = 'flex';
                    this.dom.modalExecuteAttackButton.style.display = 'none';
                    this.dom.modalCancelActionButton.style.display = 'none';
                    return;
                }
            } else {
                this.activeMedarot = null;
                this.hideModal();
                this.resumeSimulation();
            }
        } else {
            console.error(`Error in handlePartSelection: activeMedarot ${attacker.name} has invalid selectedPartKey ${attacker.selectedPartKey}`);
            this.activeMedarot = null;
            this.hideModal();
        }
    }

    handleModalExecuteAttack() {
        if (!this.activeMedarot || !this.activeMedarot.pendingTargetEnemy) {
            console.warn("ExecuteAttack called without activeMedarot or pending target.");
            this.hideModal();
            return;
        }
        const attacker = this.activeMedarot;
        attacker.currentTargetedEnemy = attacker.pendingTargetEnemy;
        attacker.currentTargetedPartKey = attacker.pendingTargetPartKey;
        attacker.pendingTargetEnemy = null;
        attacker.pendingTargetPartKey = null;
        this.clearArrow();
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
        if (this.dom.aimingArrow) this.clearArrow();
        this.dom.modalExecuteAttackButton.style.display = 'none';
        this.dom.modalCancelActionButton.style.display = 'none';
        this.dom.partSelectionContainer.style.display = 'flex';
        this.dom.modalTitle.textContent = '行動選択';
        if (this.activeMedarot) {
             this.dom.modalActorName.textContent = `${this.activeMedarot.name}のターン。`;
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
                console.log(`${medarot.name}'s pre-selected shooting target is invalid/missing. Using default targeting for shooting.`);
                targetEnemyMedarot = this._findStrategicEnemyMedarot(medarot, 'FARTHEST_X');
                if (targetEnemyMedarot) {
                    targetPartKeyForAttack = this._selectRandomPartOnEnemy(targetEnemyMedarot);
                } else {
                     console.log(`${medarot.name} could not find a fallback target for shooting.`);
                }
            }
        } else if (attackingPartCategory === '格闘') {
            targetEnemyMedarot = this._findStrategicEnemyMedarot(medarot, 'CLOSEST_X');
            if (targetEnemyMedarot && targetEnemyMedarot.state !== 'broken') {
                targetPartKeyForAttack = this._selectRandomPartOnEnemy(targetEnemyMedarot);
                if (targetPartKeyForAttack) {
                    console.log(`${medarot.name} executing fighting action against closest target: ${targetEnemyMedarot.name}, hitting part ${targetPartKeyForAttack}`);
                } else {
                     console.log(`${medarot.name} found closest enemy ${targetEnemyMedarot.name} for fighting, but it has no targetable parts.`);
                }
            } else if (targetEnemyMedarot) {
                 console.log(`${medarot.name} found closest enemy ${targetEnemyMedarot.name} but it's broken.`);
            } else {
                console.log(`${medarot.name} found no valid closest enemy for fighting action.`);
            }
        } else {
            // For other action types (Support, Disrupt, Defend), targeting might be different
            targetEnemyMedarot = this._findStrategicEnemyMedarot(medarot, 'LEADER_PRIORITY');
            if (targetEnemyMedarot) {
                targetPartKeyForAttack = this._selectRandomPartOnEnemy(targetEnemyMedarot);
            } else {
                console.log(`${medarot.name} using non-attack category ${attackingPartCategory}, could not find a fallback target.`);
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
            damage: CONFIG.RULES.BASE_DAMAGE
        };
        this.showModal('execution', medarot);
    }

    showModal(type, medarot = null, data = {}) {
        const modal = this.dom.modal;
        const title = this.dom.modalTitle;
        const actorName = this.dom.modalActorName;

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
                this.dom.partSelectionContainer.innerHTML = '';
                medarot.getAvailableAttackParts().forEach(partKey => {
                    const part = medarot.parts[partKey];
                    if (!part) return;
                    const button = document.createElement('button');
                    button.className = 'part-action-button';
                    button.textContent = `${part.name_jp} (${part.sub_category_jp || 'N/A'})`;
                    button.onclick = () => this.handlePartSelection(partKey);
                    this.dom.partSelectionContainer.appendChild(button);
                });
                this.dom.partSelectionContainer.style.display = 'flex';
                break;
            case 'execution':
                title.textContent = '攻撃実行！';
                const attackerPart = medarot.parts[medarot.selectedPartKey];
                if (!attackerPart) {
                    actorName.innerHTML = `${medarot.name}の攻撃！詳細は不明です。`;
                } else {
                    const partCategory = attackerPart.category_jp || '不明カテゴリ';
                    const partSubCategory = attackerPart.sub_category_jp || '不明サブカテゴリ';
                    const partName = attackerPart.name_jp || '不明パーツ';
                    const { target, partKey: targetPartKeyOnEnemy, damage } = medarot.preparedAttack || {};

                    if (target && targetPartKeyOnEnemy && target.parts && target.parts[targetPartKeyOnEnemy]) {
                        const targetPartName = target.parts[targetPartKeyOnEnemy].name_jp || '不明部位';
                        actorName.innerHTML = `「${medarot.name}の${partCategory}！ ${partSubCategory}行動${partName}！」<br><small>（${target.name}の${targetPartName}に ${damage} ダメージ！）</small>`;
                    } else {
                         actorName.innerHTML = `「${medarot.name}の${partCategory}！ ${partSubCategory}行動${partName}！」<br><small>（ターゲット情報エラー）</small>`;
                    }
                }
                this.dom.modalConfirmButton.style.display = 'inline-block';
                this.dom.modalConfirmButton.textContent = '了解';
                break;
            case 'battle_start_confirm':
                title.textContent = '戦闘開始！';
                actorName.textContent = '';
                this.dom.battleStartConfirmButton.style.display = 'inline-block';
                break;
            case 'game_over':
                title.textContent = `${CONFIG.TEAMS[data.winningTeam].name} の勝利！`;
                actorName.textContent = 'ロボトル終了！';
                this.dom.modalConfirmButton.style.display = 'inline-block';
                this.dom.modalConfirmButton.textContent = 'リセット';
                modal.classList.add('game-over-modal');
                break;
        }
        modal.classList.remove('hidden');
    }

    hideModal() {
        if (this.dom.aimingArrow) this.clearArrow();
        this.dom.modal.classList.add('hidden');
    }

    drawArrow(attackerMedarot, targetMedarot) {
        if (!attackerMedarot || !targetMedarot || !attackerMedarot.iconElement || !targetMedarot.iconElement || !this.dom.aimingArrow) {
            if (this.dom.aimingArrow) this.clearArrow();
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
}

document.addEventListener('DOMContentLoaded', async () => {
    const game = new GameManager();
    await game.init();
});
