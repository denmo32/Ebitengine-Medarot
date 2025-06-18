// 1. Centralized configuration
const CONFIG = {
    MAX_GAUGE: 100,
    UPDATE_INTERVAL: 50,
    PLAYERS_PER_TEAM: 3,
    PART_HP_BASE: 50,
    LEGS_HP_BONUS: 10,
    BASE_DAMAGE: 20,
    TEAMS: {
        team1: { name: 'Team 1', color: '#63b3ed', baseSpeed: 1.0, textColor: 'text-blue-300' },
        team2: { name: 'Team 2', color: '#f56565', baseSpeed: 0.9, textColor: 'text-red-300' }
    }
};
class Player {
    constructor(id, name, team, speed, options) {
        this.id = id;
        this.name = name;
        this.team = team;
        this.speed = speed;
        this.isLeader = options.isLeader;
        this.color = options.color;
        this.iconElement = null;
        this.partDOMElements = {}; // 3. Cache DOM references
        this.fullReset();
    }
    // --- State Management ---
    fullReset() {
        this.gauge = 0;
        this.state = 'charging';
        this.selectedActionType = null;
        this.selectedPartKey = null;
        this.preparedAttack = null;
        const hp = CONFIG.PART_HP_BASE;
        const legsHp = hp + CONFIG.LEGS_HP_BONUS;
        this.parts = {
            head: { name: 'Head', hp, maxHp: hp, action: 'Scan', isBroken: false },
            rightArm: { name: 'Right Arm', hp, maxHp: hp, action: 'Shoot', isBroken: false },
            leftArm: { name: 'Left Arm', hp, maxHp: hp, action: 'Fight', isBroken: false },
            legs: { name: 'Legs', hp: legsHp, maxHp: legsHp, action: 'Move', isBroken: false }
        };
    }
    startCooldown() {
        this.gauge = 0;
        this.state = 'charging';
        this.selectedActionType = null;
        this.selectedPartKey = null;
        this.preparedAttack = null;
    }
    selectAction(partKey) {
        this.selectedPartKey = partKey;
        this.selectedActionType = this.parts[partKey].action;
        this.gauge = 0;
        this.state = 'selected_charging';
    }
    // 2. Separation of concerns (Strengthening Player class roles)
    getAvailableAttackParts() {
        return Object.entries(this.parts)
            .filter(([key, part]) => !part.isBroken && ['head', 'rightArm', 'leftArm'].includes(key))
            .map(([key, _]) => key);
    }
    isReadyForSelection() {
        return this.state === 'ready_select' || this.state === 'cooldown_complete';
    }
    applyDamage(damage, partKey) {
        const part = this.parts[partKey];
        if (!part) return false;
        part.hp = Math.max(0, part.hp - damage);
        if (part.hp === 0) {
            part.isBroken = true;
            if (partKey === 'head') {
                this.state = 'broken';
                return true; // Head part destroyed
            }
        }
        return false;
    }
    processTurn() {
        if (this.parts.head.isBroken && this.state !== 'broken') this.state = 'broken';
        const statesToPause = ['ready_select', 'ready_execute', 'cooldown_complete', 'broken'];
        if (statesToPause.includes(this.state)) return;
        this.gauge += this.speed;
        if (this.gauge >= CONFIG.MAX_GAUGE) {
            this.gauge = CONFIG.MAX_GAUGE;
            if (this.state === 'charging') this.state = 'cooldown_complete';
            else if (this.state === 'selected_charging') this.state = 'ready_execute';
        }
    }
    // --- UI Related ---
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
        let partsHTML = '';
        Object.keys(this.parts).forEach(key => {
            partsHTML += `<div id="${this.id}-${key}-part" class="part-hp"></div>`;
        });
        info.innerHTML = `
            <div class="player-name ${teamConfig.textColor}">${this.name} ${this.isLeader ? '(L)' : ''}</div>
            <div class="parts-container">${partsHTML}</div>
        `;
        // 3. Cache DOM references
        Object.entries(this.parts).forEach(([key, part]) => {
            const partEl = info.querySelector(`#${this.id}-${key}-part`);
            partEl.innerHTML = `
                <span class="part-name">${part.name.substring(0,1)}</span>
                <div class="part-hp-bar-container"><div class="part-hp-bar"></div></div>
            `;
            this.partDOMElements[key] = {
                container: partEl,
                name: partEl.querySelector('.part-name'),
                bar: partEl.querySelector('.part-hp-bar')
            };
        });
        return info;
    }
    updateDisplay() {
        this.updatePosition();
        this.updateInfoPanel();
    }
    updatePosition() {
        if (!this.iconElement) return;
        const progress = this.gauge / CONFIG.MAX_GAUGE;
        let positionXRatio = (this.team === 'team1') ? 0 : 1;
        if (this.state === 'selected_charging') {
            positionXRatio = (this.team === 'team1') ? (progress * 0.5) : (1 - (progress * 0.5));
        } else if (this.state === 'charging') {
            positionXRatio = (this.team === 'team1') ? (0.5 - (progress * 0.5)) : (0.5 + (progress * 0.5));
        } else if (this.state === 'ready_execute') {
            positionXRatio = 0.5;
        }
        this.iconElement.style.left = `${positionXRatio * 100}%`;
        this.iconElement.classList.toggle('ready-select', this.isReadyForSelection());
        this.iconElement.classList.toggle('ready-execute', this.state === 'ready_execute');
        this.iconElement.classList.toggle('broken', this.state === 'broken');
    }
    updateInfoPanel() {
        Object.entries(this.parts).forEach(([key, part]) => {
            const elements = this.partDOMElements[key];
            if (!elements) return;
            const hpPercentage = (part.hp / part.maxHp) * 100;
            elements.bar.style.width = `${hpPercentage}%`;
            elements.container.classList.toggle('broken', part.isBroken);
            if (part.isBroken) {
                elements.bar.style.backgroundColor = '#4a5568';
            } else {
                if (hpPercentage > 50) elements.bar.style.backgroundColor = '#68d391';
                else if (hpPercentage > 20) elements.bar.style.backgroundColor = '#f6e05e';
                else elements.bar.style.backgroundColor = '#f56565';
            }
        });
    }
}
class GameManager {
    constructor() {
        this.players = [];
        this.simulationInterval = null;
        this.activePlayer = null;
        this.phase = 'IDLE'; // IDLE, INITIAL_SELECTION, BATTLE_START_CONFIRM, BATTLE, GAME_OVER
        this.dom = {
            startButton: document.getElementById('startButton'),
            resetButton: document.getElementById('resetButton'),
            battlefield: document.getElementById('battlefield'),
            modal: document.getElementById('actionModal'),
            modalTitle: document.getElementById('modalTitle'),
            modalActorName: document.getElementById('modalActorName'),
            partSelectionContainer: document.getElementById('partSelectionContainer'),
            modalConfirmButton: document.getElementById('modalConfirmButton'),
            battleStartConfirmButton: document.getElementById('battleStartConfirmButton')
        };
        Object.values(CONFIG.TEAMS).forEach(team => {
            this.dom[team.name.replace(/\s/g, '')] = document.getElementById(`${team.name.replace(/\s/g, '')}InfoPanel`);
        });
    }
    init() { this.createPlayers(); this.setupUI(); this.bindEvents(); }
    createPlayers() {
        this.players = [];
        Object.entries(CONFIG.TEAMS).forEach(([teamId, teamConfig], teamIndex) => {
            for (let i = 0; i < CONFIG.PLAYERS_PER_TEAM; i++) {
                const id = teamIndex * CONFIG.PLAYERS_PER_TEAM + i + 1;
                this.players.push(new Player(
                    `p${id}`, `Medarot ${id}`, teamId,
                    teamConfig.baseSpeed + (Math.random() * 0.2),
                    { isLeader: i === 0, color: teamConfig.color }
                ));
            }
        });
    }
    setupUI() {
        this.dom.battlefield.innerHTML = '<div class="center-line"></div>';
        Object.entries(CONFIG.TEAMS).forEach(([teamId, teamConfig]) => {
            const panel = document.getElementById(`${teamId}InfoPanel`);
            panel.innerHTML = `<h2 class="text-xl font-bold mb-3 ${teamConfig.textColor}">${teamConfig.name}</h2>`;
        });
        this.players.forEach(player => {
            const idNum = parseInt(player.id.substring(1));
            const indexInTeam = (idNum - 1) % CONFIG.PLAYERS_PER_TEAM;
            const vPos = 25 + indexInTeam * 25;
            this.dom.battlefield.appendChild(player.createIconDOM(vPos));
            const panel = document.getElementById(`${player.team}InfoPanel`);
            panel.appendChild(player.createInfoPanelDOM());
            player.updateDisplay();
        });
    }
    bindEvents() {
        this.dom.startButton.addEventListener('click', () => this.start());
        this.dom.resetButton.addEventListener('click', () => this.reset());
        this.dom.modalConfirmButton.addEventListener('click', () => this.handleModalConfirm());
        this.dom.battleStartConfirmButton.addEventListener('click', () => this.handleBattleStartConfirm());
    }
    start() {
        if (this.phase !== 'IDLE') return;
        this.phase = 'INITIAL_SELECTION';
        this.players.forEach(p => { p.gauge = CONFIG.MAX_GAUGE; p.state = 'ready_select'; p.updateDisplay(); });
        this.dom.startButton.disabled = true; this.dom.startButton.textContent = "Simulation in progress...";
        this.dom.resetButton.style.display = "inline-block";
        this.resumeSimulation();
    }
    pauseSimulation() { clearInterval(this.simulationInterval); this.simulationInterval = null; }
    resumeSimulation() { if (this.simulationInterval) return; this.simulationInterval = setInterval(() => this.gameLoop(), CONFIG.UPDATE_INTERVAL); }
    reset() {
        this.pauseSimulation();
        this.phase = 'IDLE'; this.activePlayer = null;
        this.hideModal();
        this.players.forEach(p => p.fullReset());
        this.setupUI();
        this.dom.startButton.disabled = false; this.dom.startButton.textContent = "Start Simulation";
        this.dom.resetButton.style.display = "none";
    }
    // 4. Simplification of the main loop
    gameLoop() {
        if (this.activePlayer || !['INITIAL_SELECTION', 'BATTLE'].includes(this.phase)) return;
        // Priority 1: Action Execution
        const playerToExecute = this.players.find(p => p.state === 'ready_execute');
        if (playerToExecute) {
            return this.handleActionExecution(playerToExecute);
        }
        // Priority 2: Action Selection
        const playerToSelect = this.players.find(p => p.isReadyForSelection());
        if (playerToSelect) {
            return this.handleActionSelection(playerToSelect);
        }
        // If no one acts
        if (this.phase === 'INITIAL_SELECTION') {
            // If all players have finished selection, proceed to battle start confirmation
            if (this.players.every(p => p.state !== 'ready_select')) {
                this.phase = 'BATTLE_START_CONFIRM';
                this.pauseSimulation();
                this.showModal('battle_start_confirm');
            }
        } else if (this.phase === 'BATTLE') {
            this.players.forEach(p => { p.processTurn(); p.updateDisplay(); });
        }
    }
    handleActionExecution(player) {
        this.activePlayer = player;
        this.pauseSimulation();
        this.prepareAndShowExecutionModal(player);
    }
    handleActionSelection(player) {
        player.state = 'ready_select'; // Normalize cooldown_complete to ready_select
        if (player.team === 'team2') { // CPU logic
            const target = this.findEnemyTarget(player);
            const partKey = player.getAvailableAttackParts()[0]; // For now, just the first part
            if (target && partKey) player.selectAction(partKey); else player.state = 'broken';
        } else { // Human player logic
            this.activePlayer = player;
            this.pauseSimulation();
            this.showModal('selection', player);
        }
    }
    findEnemyTarget(attacker) {
        const enemies = this.players.filter(p => p.team !== attacker.team && p.state !== 'broken');
        if (enemies.length === 0) return null;
        return enemies.find(e => e.isLeader) || enemies[0];
    }
    handlePartSelection(partKey) {
        if (!this.activePlayer) return;
        this.activePlayer.selectAction(partKey);
        this.activePlayer = null;
        this.hideModal();
        this.resumeSimulation();
    }
    handleBattleStartConfirm() {
        this.phase = 'BATTLE';
        this.players.forEach(p => p.processTurn());
        this.hideModal();
        this.resumeSimulation();
    }
    handleModalConfirm() {
        if (this.phase === 'GAME_OVER') return this.reset();
        if (!this.activePlayer) return;
        const attacker = this.activePlayer;
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
        this.activePlayer = null;
        this.hideModal();
        this.resumeSimulation();
    }
    prepareAndShowExecutionModal(player) {
        const target = this.findEnemyTarget(player);
        if (!target) { return player.startCooldown(); }
        const availableTargetParts = Object.keys(target.parts).filter(key => !target.parts[key].isBroken);
        if (availableTargetParts.length === 0) { return player.startCooldown(); }
        const targetPartKey = availableTargetParts[Math.floor(Math.random() * availableTargetParts.length)];
        player.preparedAttack = {
            target: target,
            partKey: targetPartKey,
            damage: CONFIG.BASE_DAMAGE
        };
        this.showModal('execution', player);
    }
    showModal(type, player = null, data = {}) {
        const modal = this.dom.modal;
        const title = this.dom.modalTitle;
        const actorName = this.dom.modalActorName;
        const partContainer = this.dom.partSelectionContainer;
        const confirmBtn = this.dom.modalConfirmButton;
        const startBtn = this.dom.battleStartConfirmButton;
        // Reset all modal elements
        [partContainer, confirmBtn, startBtn].forEach(el => el.style.display = 'none');
        modal.className = 'modal';
        switch (type) {
            case 'selection':
                title.textContent = 'Select Action';
                actorName.textContent = `${player.name}'s turn.`;
                partContainer.innerHTML = '';
                player.getAvailableAttackParts().forEach(partKey => {
                    const part = player.parts[partKey];
                    const button = document.createElement('button');
                    button.className = 'part-action-button';
                    button.textContent = `${part.name} (${part.action})`;
                    button.onclick = () => this.handlePartSelection(partKey);
                    partContainer.appendChild(button);
                });
                partContainer.style.display = 'flex';
                break;
            case 'execution':
                const { target, partKey, damage } = player.preparedAttack;
                title.textContent = 'Execute Attack!';
                actorName.textContent = `${player.name}'s ${player.selectedActionType}! Dealt ${damage} damage to ${target.name}'s ${target.parts[partKey].name}!`;
                confirmBtn.style.display = 'inline-block';
                confirmBtn.textContent = 'OK';
                break;
            case 'battle_start_confirm':
                title.textContent = 'Battle Start!';
                actorName.textContent = '';
                startBtn.style.display = 'inline-block';
                break;
            case 'game_over':
                title.textContent = `${CONFIG.TEAMS[data.winningTeam].name} Wins!`;
                actorName.textContent = 'Robattle Over!';
                confirmBtn.style.display = 'inline-block';
                confirmBtn.textContent = 'Reset';
                modal.classList.add('game-over-modal'); // You might need specific styles for this
                break;
        }
        modal.classList.remove('hidden');
    }
    hideModal() { this.dom.modal.classList.add('hidden'); }
}
document.addEventListener('DOMContentLoaded', () => {
    const game = new GameManager();
    game.init();
});
