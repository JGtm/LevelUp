// census.mjs — RECENSEMENT FINAL des positions aberrantes publiees (111 documents, schema 68),
// par categorie, avec la DUREE D'AFFICHAGE FAUTIVE que le client en tire (replayLogic.positionAt :
// interpolation lineaire sans limite de trou ; vehicles : maintien du dernier echantillon jusqu'a
// t1max ; vie close : croix/fleche de mort 2,5 s).
// Lecture sequentielle, un document a la fois. Usage : node census.mjs > census_out.txt
import fs from 'fs';
import path from 'path';

const REPO = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration';
const DIR = `${REPO}/data/cache/replays/halo_infinite`;
const HERE = path.dirname(new URL(import.meta.url).pathname).replace(/^\/([A-Za-z]:)/, '$1');
const maps = {};
for (const line of fs.readFileSync(path.join(HERE, 'maps.tsv'), 'utf8').split('\n').slice(1)) {
  const [id8, map, variant] = line.split('\t');
  if (id8) maps[id8] = { map, variant };
}
const d3 = (a, b) => Math.hypot(a.x - b.x, a.y - b.y, (a.z ?? 0) - (b.z ?? 0));
const DMIN = 25, RATIO = 0.2, VMAX = 30; // m, -, m/s
const DEATH_S = 2.5;

// garde des bornes du document, MEME REGLE que geometry.go (p1..p99 +/- 12 etendues centrales)
function guardOf(vals) {
  vals.sort((a, b) => a - b);
  const n = vals.length;
  const p1 = vals[Math.floor(n / 100)], p99 = vals[Math.floor(99 * n / 100)];
  const spread = Math.max(p99 - p1, 0.5);
  return { lo: p1 - 12 * spread, hi: p99 + 12 * spread };
}

function excursions(S) {
  const out = [];
  let i = 1;
  while (i < S.length - 1) {
    let hit = null;
    for (let L = 1; L <= 3 && i + L < S.length; L++) {
      const j = i + L - 1, a = S[i - 1], c = S[j + 1];
      let minOut = Infinity;
      for (let k = i; k <= j; k++) minOut = Math.min(minOut, d3(a, S[k]), d3(S[k], c));
      if (minOut > DMIN && d3(a, c) < RATIO * minOut) { hit = { i, j }; break; }
    }
    if (hit) { out.push(hit); i = hit.j + 2; } else i++;
  }
  return out;
}

const files = fs.readdirSync(DIR).filter((f) => /^[0-9a-f]{8}\.json$/.test(f)).sort();
const T = { docs: 0, lives: 0, openings: 0, points: 0, vehLives: 0, vehSamples: 0, singleLives: 0, singleOOB: 0 };
const cat = { T1: [], T2: [], T3: [], T4: [], V1: [], V2: [], V3: [] };
const docsTouched = new Set();
for (const f of files) {
  const id = f.slice(0, 8);
  let doc = JSON.parse(fs.readFileSync(path.join(DIR, f), 'utf8'));
  T.docs++;
  const fi = doc.frameIntervalMs || 100;
  const sec = (fr) => fr * fi / 1000;
  const off = (doc.t0FilmMs ?? 0) - (doc.originMs ?? 0);
  const shown = (t) => { const s = Math.max(0, t * fi - off) / 1000; return `${Math.floor(s / 60)}:${(s % 60).toFixed(1).padStart(4, '0')}`; };
  // garde : points des traces (comme boundsOf)
  const xs = [], ys = [], zs = [];
  for (const tr of doc.tracks) for (const p of tr.points) { xs.push(p.x); ys.push(p.y); zs.push(p.z); }
  const g = [guardOf(xs), guardOf(ys), guardOf(zs)];
  const hors = (p) => p.x < g[0].lo || p.x > g[0].hi || p.y < g[1].lo || p.y > g[1].hi || p.z < g[2].lo || p.z > g[2].hi;
  const m = maps[id]?.map;
  const push = (k, r) => { cat[k].push({ id, map: m, ...r }); docsTouched.add(id); };
  for (const tr of doc.tracks) {
    const S = tr.points;
    T.lives++; T.openings++; T.points += S.length;
    if (S.length === 1) {
      T.singleLives++;
      if (hors(S[0])) { T.singleOOB++; push('T2', { slot: tr.slot, t: S[0].t, shown: shown(S[0].t), p: S[0], visS: 0.1 + DEATH_S, hors: true }); }
      continue;
    }
    const d01 = d3(S[0], S[1]), v01 = d01 / Math.max(sec(S[1].t - S[0].t), 1e-3);
    if (d01 > DMIN && v01 > VMAX) push('T1', { slot: tr.slot, t: S[0].t, shown: shown(S[0].t), p: S[0], visS: sec(S[1].t - S[0].t), hors: hors(S[0]), d: d01 });
    const n = S.length;
    const dl = d3(S[n - 2], S[n - 1]), vl = dl / Math.max(sec(S[n - 1].t - S[n - 2].t), 1e-3);
    if (n >= 3 && dl > DMIN && vl > VMAX) push('T4', { slot: tr.slot, t: S[n - 1].t, shown: shown(S[n - 1].t), p: S[n - 1], visS: sec(S[n - 1].t - S[n - 2].t) + DEATH_S, hors: hors(S[n - 1]), d: dl });
    for (const e of excursions(S)) push('T3', { slot: tr.slot, t: S[e.i].t, shown: shown(S[e.i].t), p: S[e.i], run: e.j - e.i + 1, visS: sec(S[e.j + 1].t - S[e.i - 1].t), hors: S.slice(e.i, e.j + 1).some(hors) });
  }
  for (const v of doc.vehicles || []) {
    const S = v.samples || [];
    T.vehLives++; T.vehSamples += S.length;
    const lastFrame = v.t1max ?? v.t1;
    // V2 : vie FANTOME — aucun echantillon dans la garde (et pas de naissance, ou naissance hors garde)
    if (S.length > 0 && S.every(hors)) {
      push('V2', { slot: v.slot, family: v.family, t: S[0].t, shown: shown(S[0].t), p: S[0], n: S.length, visS: sec(lastFrame - S[0].t), hors: true, spawn: !!v.spawn });
      continue;
    }
    for (const e of excursions(S)) push('V1', { slot: v.slot, family: v.family, t: S[e.i].t, shown: shown(S[e.i].t), p: S[e.i], run: e.j - e.i + 1, visS: sec(S[e.j + 1].t - S[e.i - 1].t), hors: S.slice(e.i, e.j + 1).some(hors) });
    // V3 : NAISSANCE hors garde alors que la trajectoire, elle, est dans la garde
    if (v.spawn && hors(v.spawn) && S.length) push('V3', { slot: v.slot, family: v.family, t: v.t0, shown: shown(v.t0), p: v.spawn, visS: sec(S[0].t - v.t0), hors: true, first: S[0] });
  }
  doc = null;
}
const fmt = (p) => `(${p.x},${p.y},${p.z})`;
for (const [k, rows] of Object.entries(cat)) {
  const vis = rows.map((r) => r.visS).sort((a, b) => a - b);
  const sum = vis.reduce((a, b) => a + b, 0);
  const docs = new Set(rows.map((r) => r.id));
  const horsN = rows.filter((r) => r.hors).length;
  console.log(`## ${k} : ${rows.length} cas, ${docs.size} documents, hors garde 12 etendues ${horsN}/${rows.length}, duree d'affichage fautive totale ${sum.toFixed(1)} s, mediane ${vis.length ? vis[Math.floor(vis.length / 2)].toFixed(1) : 0} s, max ${vis.length ? vis[vis.length - 1].toFixed(1) : 0} s`);
  for (const r of rows) console.log(`   ${r.id} ${r.map} slot=${r.slot} ${r.family ?? ''} t=${r.t} (${r.shown}) ${fmt(r.p)} ${r.run ? 'run=' + r.run : ''} ${r.n ? 'n=' + r.n : ''} ${r.d ? 'd=' + r.d.toFixed(0) : ''} vis=${r.visS.toFixed(1)}s hors=${r.hors}${r.first ? ' 1er=' + fmt(r.first) : ''}${r.spawn !== undefined ? ' spawn=' + r.spawn : ''}`);
}
console.log('# TOTAUX', JSON.stringify(T), 'documents touches', docsTouched.size);
