// panneaux_lib.mjs — REPRODUCTION FIDELE de la composition des panneaux d'equipe du rejeu
// (web, commit 43a01721e). Chaque fonction cite sa source TS ; aucune regle n'est ajoutee.
//
//   buildPlayers         apps/web/src/lib/replay/rosterLogic.ts:107-149
//   rosterEntryKey       apps/web/src/lib/replay/rosterLogic.ts:67-69
//   trackWindow          apps/web/src/lib/replay/replayLogic.ts:428-434
//   buildSeats           apps/web/src/features/match-replay/model/seatLogic.ts:147-180
//   campsParCote         seatLogic.ts:211-229 ; cleDeCamp seatLogic.ts:239-249
//   enveloppeDePresence  seatLogic.ts:252-262
//   groupSeatsByTeam     seatLogic.ts:277-311
//   seatOccupantAt       seatLogic.ts:123-134
//   rendu d'une colonne  apps/web/src/features/match-replay/ui/ReplayTeams.tsx:197-238
//                        (absent -> rien ; parti -> ReplaySeatLeft ; present -> ReplayPlayerCard)
//   scoreboard           apps/go-api/internal/platform/duckdb/queries_match.go:47-112 (filtre
//                        « tout a zero » compris) + match_view_builders_team.go:116-119 (t{team_id})
import fs from 'fs';

export const REPO = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration';
export const SPD = 'C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/5e1d6f3f-b6be-4233-b361-e8a896dd728d/scratchpad/sondes/equipes_b1ad85eb';

export function loadDoc(id8) {
  return JSON.parse(fs.readFileSync(`${REPO}/data/cache/replays/halo_infinite/${id8}.json`, 'utf8'));
}

function readTsv(path) {
  const lines = fs.readFileSync(path, 'utf8').split(/\r?\n/).filter((l) => l && !l.startsWith('('));
  const head = lines[0].split('\t');
  return lines.slice(1).map((l) => {
    const c = l.split('\t');
    const o = {};
    head.forEach((h, i) => { o[h] = c[i] === 'NULL' ? null : c[i]; });
    return o;
  });
}

let _parts = null;
export function participants(id8) {
  if (!_parts) {
    _parts = new Map();
    for (const r of readTsv(`${SPD}/participants.tsv`)) {
      if (!_parts.has(r.id8)) _parts.set(r.id8, []);
      _parts.get(r.id8).push(r);
    }
  }
  return _parts.get(id8) ?? [];
}

let _reg = null;
export function registry(id8) {
  if (!_reg) {
    _reg = new Map();
    for (const r of readTsv(`${SPD}/registry.tsv`)) _reg.set(r.id8, r);
  }
  return _reg.get(id8);
}

// Q12 : les lignes « tout a zero » (kills, deaths, assists, personal_score = 0 avec au moins une
// colonne non nulle) SONT EXCLUES du scoreboard servi au web.
export function scoreboard(id8) {
  const out = [];
  for (const r of participants(id8)) {
    const zero = +r.k === 0 && +r.d === 0 && +r.a === 0 && +r.ps === 0 && r.anynn === 'true';
    if (zero) continue;
    out.push({
      xuid: r.xuid,
      gamertag: r.gamertag,
      is_bot: r.xuid.startsWith('bid('),
      team_side: r.team_id == null ? undefined : `t${r.team_id}`,
    });
  }
  return out;
}

const BOT_SUFFIX = ' [bot]';
export const stripBotSuffix = (g) => (g.endsWith(BOT_SUFFIX) ? g.slice(0, -BOT_SUFFIX.length) : g);
const botKey = (name) => `bot:${name}`;
export const rosterEntryKey = (e) => e.xuid || (e.bot && e.name ? botKey(e.name) : '');

export function trackWindow(track) {
  const last = track.points[track.points.length - 1];
  return { start: track.startFrame ?? 0, end: track.endFrame ?? (last ? last.t : 0) };
}

export function buildPlayers(doc, board) {
  const byXUID = new Map();
  for (const entry of doc.roster ?? []) {
    const key = entry.xuid || (entry.bot && entry.name ? botKey(entry.name) : '');
    if (!key) continue;
    byXUID.set(key, { xuid: key, bot: entry.bot || undefined, filmName: entry.name, lives: [] });
  }
  for (const track of doc.tracks ?? []) {
    const key = track.xuid || (track.bot ? botKey(track.bot) : '');
    if (!key) continue;
    let p = byXUID.get(key);
    if (!p) {
      p = { xuid: key, bot: track.bot ? true : undefined, filmName: track.bot, lives: [] };
      byXUID.set(key, p);
    }
    p.lives.push(track);
  }
  const byX = new Map(board.map((r) => [r.xuid, r]));
  const byName = new Map(board.filter((r) => r.is_bot).map((r) => [stripBotSuffix(r.gamertag), r]));
  for (const p of byXUID.values()) {
    p.board = p.bot ? byName.get(stripBotSuffix(p.filmName ?? '')) : byX.get(p.xuid);
    p.lives.sort((a, b) => trackWindow(a).start - trackWindow(b).start);
  }
  return [...byXUID.values()];
}

function enveloppeDePresence(p) {
  if (p.lives.length === 0) return null;
  let from = Infinity;
  let to = -Infinity;
  for (const life of p.lives) {
    const w = trackWindow(life);
    if (w.start < from) from = w.start;
    if (w.end > to) to = w.end;
  }
  return { fromFrame: from, toFrame: to };
}

function campsParCote(players, parIdentite) {
  const out = new Map();
  const douteux = new Set();
  for (const p of players) {
    const team = parIdentite.get(p.xuid)?.team;
    const side = p.board?.team_side;
    if (team === undefined || team === null || side == null || douteux.has(side)) continue;
    const vu = out.get(side);
    if (vu === undefined) out.set(side, team);
    else if (vu !== team) { out.delete(side); douteux.add(side); }
  }
  return out;
}

function cleDeCamp(e, p, parCote) {
  if (e?.team !== undefined && e.team !== null) return `f${e.team}`;
  const side = p.board?.team_side;
  if (side == null) return '';
  const film = parCote.get(side);
  return film !== undefined ? `f${film}` : `s:${side}`;
}

export function buildSeats(players, doc) {
  const parIdentite = new Map();
  for (const e of doc.roster ?? []) {
    const cle = rosterEntryKey(e);
    if (cle) parIdentite.set(cle, e);
  }
  const parCote = campsParCote(players, parIdentite);
  const sieges = new Map();
  for (const p of players) {
    const presence = enveloppeDePresence(p);
    if (!presence) continue;
    const e = parIdentite.get(p.xuid);
    const numero = e?.seat ?? e?.filmIndex ?? -1;
    const cle = numero >= 0 ? `siege:${numero}` : `joueur:${p.xuid}`;
    let s = sieges.get(cle);
    if (!s) { s = { key: cle, seat: numero, side: null, teamKey: '', occupants: [] }; sieges.set(cle, s); }
    s.occupants.push({ ...presence, player: p, apparie: e?.seatSource === 'apparie' });
    if (s.side === null) s.side = p.board?.team_side ?? null;
    if (s.teamKey === '') s.teamKey = cleDeCamp(e, p, parCote);
  }
  const out = [...sieges.values()];
  for (const s of out) s.occupants.sort((a, b) => a.fromFrame - b.fromFrame);
  const rang = (s) => (s.seat < 0 ? Number.MAX_SAFE_INTEGER : s.seat);
  return out.sort((a, b) => (rang(a) !== rang(b) ? rang(a) - rang(b) : a.key.localeCompare(b.key)));
}

function ordreDesCamps(ka, sa, ra, kb, sb, rb) {
  if ((ka === '') !== (kb === '')) return ka === '' ? 1 : -1;
  if (sa !== null && sb !== null && sa !== sb) return sa.localeCompare(sb);
  if (ka !== kb) return ka.localeCompare(kb);
  return ra - rb;
}

export function groupSeatsByTeam(seats) {
  const groups = new Map();
  for (const s of seats) {
    let g = groups.get(s.teamKey);
    if (!g) { g = { side: s.side, seats: [], rang: groups.size, key: s.teamKey }; groups.set(s.teamKey, g); }
    if (g.side === null) g.side = s.side;
    g.seats.push(s);
  }
  return [...groups.entries()]
    .sort(([ka, a], [kb, b]) => ordreDesCamps(ka, a.side, a.rang, kb, b.side, b.rang))
    .map(([k, g]) => ({ key: k, side: g.side, seats: g.seats }));
}

export function seatOccupantAt(seat, frame) {
  let sorti = -1;
  for (let i = 0; i < seat.occupants.length; i++) {
    const o = seat.occupants[i];
    if (frame < o.fromFrame) break;
    if (frame <= o.toFrame) return { player: o.player, kind: 'present', o };
    sorti = i;
  }
  if (sorti < 0) return { player: null, kind: 'absent' };
  const attendu = sorti + 1 < seat.occupants.length;
  return { player: seat.occupants[sorti].player, kind: attendu ? 'parti' : 'present', o: seat.occupants[sorti], fantome: !attendu };
}

// Le nom que la FICHE ecrit (playerCardReadings.ts:99-100) : gamertag du tableau de score, sinon
// celui du film, suffixe bot retire, sinon « Joueur inconnu » (i18n `unknownPlayer`).
export function playerLabel(p) {
  const gt = p.board?.gamertag || p.filmName || null;
  return gt ? stripBotSuffix(gt) : 'Joueur inconnu';
}

// La colonne telle que ReplayTeams la rend a l'image `frame`.
export function panneaux(groups, frame) {
  return groups.map((g) => {
    const tuiles = [];
    for (const seat of g.seats) {
      const lu = seatOccupantAt(seat, frame);
      if (lu.kind === 'absent' || lu.player === null) continue;
      tuiles.push({ seat: seat.seat, kind: lu.kind, fantome: !!lu.fantome, name: playerLabel(lu.player), xuid: lu.player.xuid, o: lu.o });
    }
    return { key: g.key, side: g.side, tuiles };
  });
}

export const displayToFrame = (doc, s) => Math.round((s * 1000 + (doc.t0FilmMs - doc.originMs)) / doc.frameIntervalMs);
export const frameToDisplay = (doc, f) => {
  const ms = Math.max(0, f * doc.frameIntervalMs - (doc.t0FilmMs - doc.originMs));
  const s = ms / 1000;
  return `${Math.floor(s / 60)}:${(s % 60).toFixed(1).padStart(4, '0')}`;
};
