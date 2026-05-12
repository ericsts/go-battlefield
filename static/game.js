'use strict';

// ── Constants ──────────────────────────────────────────────────────────────
const N = 10;
const SHIPS = [
  { name: 'Carrier',    size: 5 },
  { name: 'Battleship', size: 4 },
  { name: 'Cruiser',    size: 3 },
  { name: 'Submarine',  size: 3 },
  { name: 'Destroyer',  size: 2 },
];

// ── State ──────────────────────────────────────────────────────────────────
const S = {
  phase: 'lobby',
  playerId: null,
  myTurn: false,

  // placement
  placed: [],       // { name, size, x, y, horizontal }
  selected: null,   // { name, size }
  horizontal: true,
  hoverX: -1, hoverY: -1,

  // battle
  myBoard: null,
  enemyBoard: null,
  mySunk: 0,
  enemySunk: 0,
};

let ws;

// ── Boot ───────────────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  const path = window.location.pathname;
  if (path.startsWith('/game/')) {
    const roomId = path.slice('/game/'.length);
    showScreen('connecting');
    connectWS(roomId);
  }
});

function createGame() {
  window.location.href = '/create';
}

// ── WebSocket ──────────────────────────────────────────────────────────────
function connectWS(roomId) {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  ws = new WebSocket(`${proto}://${location.host}/ws/${roomId}`);
  ws.onmessage = e => onMsg(JSON.parse(e.data));
  ws.onerror   = ()  => disconnect('Connection error.');
  ws.onclose   = ()  => { if (S.phase !== 'over') disconnect('Connection lost.'); };
}

function send(type, data) {
  if (ws && ws.readyState === WebSocket.OPEN)
    ws.send(JSON.stringify({ type, data }));
}

// ── Message Handling ───────────────────────────────────────────────────────
function onMsg(msg) {
  switch (msg.type) {
    case 'joined':
      S.playerId = msg.player_id;
      if (msg.waiting) {
        document.getElementById('share-link').value = window.location.href;
        showScreen('waiting');
      }
      break;

    case 'phase_change':
      if (msg.phase === 'placement') {
        S.phase = 'placement';
        showScreen('placement');
        initPlacement();
      }
      break;

    case 'waiting_for_opponent':
      document.getElementById('placement-status').textContent = 'Waiting for opponent…';
      document.getElementById('ready-btn').textContent = 'Waiting…';
      break;

    case 'game_start':
      S.phase = 'battle';
      S.myTurn = msg.your_turn;
      S.mySunk = 0;
      S.enemySunk = 0;
      initBattle();
      showScreen('battle');
      break;

    case 'shot_result':
      applyShot('enemy', msg);
      S.myTurn = msg.your_turn;
      if (msg.winner) endGame(true);
      else updateTurn();
      break;

    case 'enemy_shot':
      applyShot('mine', msg);
      S.myTurn = msg.your_turn;
      if (msg.winner) endGame(false);
      else updateTurn();
      break;

    case 'opponent_disconnected':
      disconnect('Your opponent disconnected.');
      break;

    case 'error':
      toast(msg.message || 'Error', '');
      break;
  }
}

// ── Shot Application ───────────────────────────────────────────────────────
function applyShot(target, msg) {
  const board  = target === 'enemy' ? S.enemyBoard : S.myBoard;
  const gridId = target === 'enemy' ? 'enemy-board' : 'my-board';

  if (msg.hit) {
    board[msg.y][msg.x] = 'hit';
    setCellClass(gridId, msg.x, msg.y, 'hit');

    if (msg.sunk && msg.sunk_cells) {
      if (target === 'enemy') {
        msg.sunk_cells.forEach(([cx, cy]) => {
          board[cy][cx] = 'sunk';
          setCellClass(gridId, cx, cy, 'sunk');
        });
        S.enemySunk++;
        toast(`💥 You sunk their ${msg.ship}!`, 'sunk-toast');
      } else {
        msg.sunk_cells.forEach(([cx, cy]) => {
          board[cy][cx] = 'ship-sunk';
          setCellClass(gridId, cx, cy, 'ship-sunk');
        });
        S.mySunk++;
        toast(`❗ Your ${msg.ship} was sunk!`, 'sunk-toast');
      }
      updateFleetStatus();
    } else {
      toast(target === 'enemy' ? '🔥 Hit!' : '🔥 Your ship was hit!', 'hit-toast');
    }
  } else {
    board[msg.y][msg.x] = 'miss';
    setCellClass(gridId, msg.x, msg.y, 'miss');
    toast(target === 'enemy' ? 'Miss.' : 'Opponent missed.', 'miss-toast');
  }

  if (target === 'enemy') refreshEnemyClickability();
}

function setCellClass(gridId, x, y, state) {
  const grid = document.getElementById(gridId);
  const cell = grid.children[y * N + x];
  if (cell) cell.className = `cell ${state}`;
}

// ── Battle ─────────────────────────────────────────────────────────────────
function initBattle() {
  S.myBoard    = make2D('empty');
  S.enemyBoard = make2D('empty');
  S.placed.forEach(ship => {
    cells(ship).forEach(([x, y]) => { S.myBoard[y][x] = 'ship'; });
  });
  buildBoard('my-board', S.myBoard, false);
  buildBoard('enemy-board', S.enemyBoard, true);
  updateTurn();
  updateFleetStatus();
}

function buildBoard(id, board, clickable) {
  const grid = document.getElementById(id);
  grid.innerHTML = '';
  for (let y = 0; y < N; y++) {
    for (let x = 0; x < N; x++) {
      const div = document.createElement('div');
      div.className = `cell ${board[y][x]}`;
      if (clickable) {
        div.dataset.x = x;
        div.dataset.y = y;
      }
      grid.appendChild(div);
    }
  }
  if (clickable) {
    grid.addEventListener('click', e => {
      const cell = e.target.closest('[data-x]');
      if (!cell || !S.myTurn || S.phase !== 'battle') return;
      const cx = +cell.dataset.x, cy = +cell.dataset.y;
      if (S.enemyBoard[cy][cx] === 'empty') shoot(cx, cy);
    });
  }
}

function refreshEnemyClickability() {
  const grid = document.getElementById('enemy-board');
  for (let y = 0; y < N; y++) {
    for (let x = 0; x < N; x++) {
      const cell = grid.children[y * N + x];
      if (!cell) continue;
      const st = S.enemyBoard[y][x];
      cell.className = `cell ${st}`;
      cell.dataset.x = x;
      cell.dataset.y = y;
    }
  }
}

function shoot(x, y) {
  S.myTurn = false;
  updateTurn();
  send('shoot', { x, y });
}

function updateTurn() {
  const el = document.getElementById('turn-banner');
  if (S.myTurn) {
    el.textContent = '🎯 Your Turn — Fire!';
    el.className = 'turn-banner my-turn';
  } else {
    el.textContent = "⏳ Opponent's Turn…";
    el.className = 'turn-banner their-turn';
  }
  refreshEnemyClickability();
}

function updateFleetStatus() {
  document.getElementById('my-fleet-status').textContent =
    S.mySunk > 0 ? `${S.mySunk} ship${S.mySunk > 1 ? 's' : ''} sunk` : '';
  document.getElementById('enemy-fleet-status').textContent =
    S.enemySunk > 0 ? `${S.enemySunk} ship${S.enemySunk > 1 ? 's' : ''} sunk` : '';
}

// ── Game Over ──────────────────────────────────────────────────────────────
function endGame(won) {
  S.phase = 'over';
  document.getElementById('gameover-icon').textContent = won ? '🏆' : '💀';
  document.getElementById('gameover-title').textContent = won ? 'Victory!' : 'Defeated!';
  document.getElementById('gameover-sub').textContent =
    won ? 'You sunk the entire enemy fleet.' : 'Your fleet was destroyed.';
  showScreen('game-over');
}

function disconnect(msg) {
  if (S.phase === 'over') return;
  document.getElementById('disconnect-msg').textContent = msg;
  document.getElementById('disconnect-overlay').classList.add('show');
}

// ── Ship Placement ─────────────────────────────────────────────────────────
//
// Cells are created ONCE and their classes are updated in-place.
// We never call innerHTML = '' during hover — that would cause the browser
// to fire mouseenter on freshly created cells, creating an infinite loop
// that makes clicks unreachable.

function initPlacement() {
  S.placed     = [];
  S.selected   = null;
  S.horizontal = true;
  S.hoverX     = -1;
  S.hoverY     = -1;

  buildPlacementGrid();
  renderShipList();
  document.getElementById('ready-btn').disabled   = true;
  document.getElementById('ready-btn').textContent = 'Ready!';
  document.getElementById('placement-status').textContent = '';
  document.getElementById('rotate-label').textContent = '⟳ Horizontal';
}

function buildPlacementGrid() {
  const grid = document.getElementById('placement-grid');
  grid.innerHTML = '';

  for (let y = 0; y < N; y++) {
    for (let x = 0; x < N; x++) {
      const div = document.createElement('div');
      div.className = 'cell';
      div.dataset.x = x;
      div.dataset.y = y;
      grid.appendChild(div);
    }
  }

  // Single listeners on the grid element — no per-cell listeners
  grid.addEventListener('mousemove', onPlacementMouseMove);
  grid.addEventListener('mouseleave', onPlacementMouseLeave);
  grid.addEventListener('click', onPlacementGridClick);
}

function onPlacementMouseMove(e) {
  const cell = e.target.closest('[data-x]');
  if (!cell) return;
  const x = +cell.dataset.x, y = +cell.dataset.y;
  if (x !== S.hoverX || y !== S.hoverY) {
    S.hoverX = x;
    S.hoverY = y;
    refreshPlacementClasses();
  }
}

function onPlacementMouseLeave() {
  S.hoverX = -1;
  S.hoverY = -1;
  refreshPlacementClasses();
}

function onPlacementGridClick(e) {
  const cell = e.target.closest('[data-x]');
  if (!cell) return;
  placementClick(+cell.dataset.x, +cell.dataset.y);
}

// Updates CSS classes on existing cells — never recreates them
function refreshPlacementClasses() {
  const grid = document.getElementById('placement-grid');

  // Compute preview set
  const previewSet = new Set();
  let previewValid = false;
  if (S.selected && S.hoverX >= 0) {
    const pc = cells({ ...S.selected, x: S.hoverX, y: S.hoverY, horizontal: S.horizontal });
    previewValid = canPlace(S.selected, S.hoverX, S.hoverY, S.horizontal, S.placed);
    pc.forEach(([cx, cy]) => {
      if (cx >= 0 && cx < N && cy >= 0 && cy < N)
        previewSet.add(cy * N + cx);
    });
  }

  for (let i = 0; i < grid.children.length; i++) {
    const cell = grid.children[i];
    const x = +cell.dataset.x, y = +cell.dataset.y;

    if (previewSet.has(y * N + x)) {
      cell.className = `cell ${previewValid ? 'preview-ok' : 'preview-bad'}`;
    } else if (shipAt(x, y, S.placed)) {
      cell.className = 'cell ship';
    } else {
      cell.className = 'cell';
    }
  }
}

function placementClick(x, y) {
  if (S.selected) {
    if (canPlace(S.selected, x, y, S.horizontal, S.placed)) {
      S.placed.push({ ...S.selected, x, y, horizontal: S.horizontal });
      S.selected = null;
      renderShipList();
      refreshPlacementClasses();
      checkReady();
    }
  } else {
    // pick up an already-placed ship
    const idx = S.placed.findIndex(p => cells(p).some(([cx, cy]) => cx === x && cy === y));
    if (idx !== -1) {
      const ship    = S.placed.splice(idx, 1)[0];
      S.selected   = { name: ship.name, size: ship.size };
      S.horizontal = ship.horizontal;
      renderShipList();
      refreshPlacementClasses();
      checkReady();
    }
  }
}

function renderShipList() {
  const list = document.getElementById('ship-list');
  list.innerHTML = '';
  SHIPS.forEach(ship => {
    const placed = S.placed.some(p => p.name === ship.name);
    const sel    = S.selected && S.selected.name === ship.name;

    const div = document.createElement('div');
    div.className = `ship-item${placed ? ' placed' : ''}${sel ? ' selected' : ''}`;

    const segs = Array.from({ length: ship.size }, () => '<span class="ship-seg"></span>').join('');
    div.innerHTML = `<div class="ship-name">${ship.name} (${ship.size})</div>
                     <div class="ship-bar-wrap">${segs}</div>`;

    if (!placed) {
      div.addEventListener('click', () => {
        S.selected = ship;
        renderShipList();
        refreshPlacementClasses();
      });
    }
    list.appendChild(div);
  });
}

function toggleRotate() {
  S.horizontal = !S.horizontal;
  document.getElementById('rotate-label').textContent =
    S.horizontal ? '⟳ Horizontal' : '⟳ Vertical';
  refreshPlacementClasses();
}

function clearShips() {
  S.placed   = [];
  S.selected = null;
  renderShipList();
  refreshPlacementClasses();
  checkReady();
}

function checkReady() {
  const all = SHIPS.every(s => S.placed.some(p => p.name === s.name));
  document.getElementById('ready-btn').disabled = !all;
}

function submitReady() {
  send('place_ships', {
    ships: S.placed.map(s => ({ name: s.name, x: s.x, y: s.y, horizontal: s.horizontal })),
  });
  document.getElementById('ready-btn').disabled   = true;
  document.getElementById('ready-btn').textContent = 'Waiting…';
}

// ── Random Placement ───────────────────────────────────────────────────────
function randomPlace() {
  for (let attempt = 0; attempt < 50; attempt++) {
    const result = tryRandom();
    if (result) {
      S.placed   = result;
      S.selected = null;
      renderShipList();
      refreshPlacementClasses();
      checkReady();
      return;
    }
  }
}

function tryRandom() {
  const placed = [];
  for (const ship of SHIPS) {
    let ok = false;
    for (let t = 0; t < 200; t++) {
      const h    = Math.random() < 0.5;
      const maxX = h ? N - ship.size : N - 1;
      const maxY = h ? N - 1 : N - ship.size;
      const x    = Math.floor(Math.random() * (maxX + 1));
      const y    = Math.floor(Math.random() * (maxY + 1));
      if (canPlace(ship, x, y, h, placed)) {
        placed.push({ ...ship, x, y, horizontal: h });
        ok = true;
        break;
      }
    }
    if (!ok) return null;
  }
  return placed;
}

// ── Helpers ────────────────────────────────────────────────────────────────
function cells(ship) {
  return Array.from({ length: ship.size }, (_, i) => [
    ship.horizontal ? ship.x + i : ship.x,
    ship.horizontal ? ship.y     : ship.y + i,
  ]);
}

function shipAt(x, y, placed) {
  return placed.find(p => cells(p).some(([cx, cy]) => cx === x && cy === y));
}

function canPlace(ship, x, y, horizontal, placed) {
  for (const [cx, cy] of cells({ ...ship, x, y, horizontal })) {
    if (cx < 0 || cx >= N || cy < 0 || cy >= N) return false;
    if (shipAt(cx, cy, placed)) return false;
  }
  return true;
}

function make2D(val) {
  return Array.from({ length: N }, () => Array(N).fill(val));
}

// ── Toast ──────────────────────────────────────────────────────────────────
let toastTimer;
function toast(msg, cls) {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className   = `toast show ${cls || ''}`;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => el.classList.remove('show'), 2800);
}

// ── Screen ─────────────────────────────────────────────────────────────────
function showScreen(name) {
  document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
  document.getElementById(`screen-${name}`).classList.add('active');
}

// ── Keyboard ───────────────────────────────────────────────────────────────
document.addEventListener('keydown', e => {
  if ((e.key === 'r' || e.key === 'R') && S.phase === 'placement') toggleRotate();
});

// ── Clipboard ──────────────────────────────────────────────────────────────
function copyLink() {
  const inp = document.getElementById('share-link');
  inp.select();
  navigator.clipboard?.writeText(inp.value).catch(() => document.execCommand('copy'));
  toast('Link copied!', '');
}
