// sweep.mjs — balayage des 111 documents de rejeu (schema 68) : positions aberrantes.
// Lecture SEQUENTIELLE (un document en memoire a la fois). Aucune ecriture hors du scratchpad.
// Usage : node sweep.mjs > sweep_out.txt
import fs from 'fs';
import path from 'path';

const REPO = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration';
const DIR = `${REPO}/data/cache/replays/halo_infinite`;
const HERE = path.dirname(new URL(import.meta.url).pathname).replace(/^\/([A-Za-z]:)/, '$1');

// carte par match
const maps = {};
for (const line of fs.readFileSync(path.join(HERE, 'maps.tsv'), 'utf8').split('\n').slice(1)) {
  const [id8, map, variant] = line.split('\t');
  if (id8) maps[id8] = { map, variant };
}
const cat = JSON.parse(fs.readFileSync(`${REPO}/data/titles/halo_infinite/reference/map_quant_bounds.json`, 'utf8'));
const norm = (s) => (s || '').toLowerCase().split(/\s+/).filter(Boolean).join(' ').replace(/ - ranked$/, '').replace(/ heavies$/, '');

const d3 = (a, b) => Math.hypot(a.x - b.x, a.y - b.y, (a.z ?? 0) - (b.z ?? 0));
const DMIN = 25; // m : un aller-retour plus court n'est pas compte
const RATIO = 0.2; // retour : d(avant, apres) < RATIO * min(distance aller, retour)

function quanta(entry, p) {
  if (!entry) return null;
  const out = [];
  const v = [p.x, p.y, p.z];
  for (let ax = 0; ax < 3; ax++) {
    const w = entry.axisWidths[ax];
    const step = (entry.max[ax] - entry.min[ax]) / 2 ** w;
    out.push(Math.round((v[ax] - entry.min[ax]) / step - 0.5));
  }
  return out;
}

// excursions : runs [i..j] (1 a 3 echantillons) dont le voisin avant et apres sont proches l'un de l'autre
function excursions(S) {
  const out = [];
  const n = S.length;
  let i = 1;
  while (i < n - 1) {
    let hit = null;
    for (let L = 1; L <= 3 && i + L < n; L++) {
      const j = i + L - 1;
      const a = S[i - 1], c = S[j + 1];
      const dac = d3(a, c);
      let minOut = Infinity;
      for (let k = i; k <= j; k++) minOut = Math.min(minOut, d3(a, S[k]), d3(S[k], c));
      if (minOut > DMIN && dac < RATIO * minOut) { hit = { i, j, before: a, after: c, minOut, dac }; break; }
    }
    if (hit) { out.push(hit); i = hit.j + 2; } else i++;
  }
  return out;
}

const files = fs.readdirSync(DIR).filter((f) => /^[0-9a-f]{8}\.json$/.test(f)).sort();
const rows = [];
const perDoc = [];
let totTracks = 0, totTrackPts = 0, totVeh = 0, totVehSamples = 0;
for (const f of files) {
  const id = f.slice(0, 8);
  let doc = JSON.parse(fs.readFileSync(path.join(DIR, f), 'utf8'));
  const m = maps[id] || {};
  const entry = cat.maps[norm(m.map)];
  const B = doc.bounds;
  const fi = doc.frameIntervalMs || 100;
  const off = (doc.t0FilmMs ?? 0) - (doc.originMs ?? 0);
  const shown = (t) => { const ms = Math.max(0, t * fi - off); const s = ms / 1000; return `${Math.floor(s / 60)}:${(s % 60).toFixed(1).padStart(4, '0')}`; };
  const mx = Math.max(20, 0.25 * (B.maxX - B.minX)), my = Math.max(20, 0.25 * (B.maxY - B.minY));
  const oob = (p) => p.x < B.minX - mx || p.x > B.maxX + mx || p.y < B.minY - my || p.y > B.maxY + my || p.z < B.minZ - 15 || p.z > B.maxZ + 30;
  const stat = { id, map: m.map, variant: m.variant, tracks: doc.tracks.length, vehicles: (doc.vehicles || []).length,
    trFirst: 0, trExc: 0, trLast: 0, trOOB: 0, trSingleOOB: 0, vExc: 0, vOOB: 0, vFirst: 0, vLongGapJump: 0, trLongGapJump: 0 };
  const push = (r) => { rows.push({ id, map: m.map, ...r }); };
  // --- traces joueurs
  for (const tr of doc.tracks) {
    const S = tr.points;
    totTracks++; totTrackPts += S.length;
    for (const p of S) if (oob(p)) stat.trOOB++;
    if (S.length === 1 && oob(S[0])) { stat.trSingleOOB++; push({ kind: 'track-single', slot: tr.slot, xuid: tr.xuid, t: S[0].t, shown: shown(S[0].t), p: S[0], q: quanta(entry, S[0]), interpFrames: 0 }); }
    if (S.length >= 2) {
      const d01 = d3(S[0], S[1]); const dt = S[1].t - S[0].t; const v = d01 / Math.max(dt * fi / 1000, 1e-3);
      if (d01 > DMIN && v > 30) {
        stat.trFirst++;
        push({ kind: 'track-first', slot: tr.slot, xuid: tr.xuid, t: S[0].t, shown: shown(S[0].t), p: S[0], next: S[1], d: d01, v, interpFrames: dt, q: quanta(entry, S[0]), oob: oob(S[0]) });
      }
      const n = S.length;
      const dl = d3(S[n - 2], S[n - 1]); const dtl = S[n - 1].t - S[n - 2].t; const vl = dl / Math.max(dtl * fi / 1000, 1e-3);
      if (n >= 3 && dl > DMIN && vl > 30) {
        stat.trLast++;
        push({ kind: 'track-last', slot: tr.slot, xuid: tr.xuid, t: S[n - 1].t, shown: shown(S[n - 1].t), p: S[n - 1], prev: S[n - 2], d: dl, v: vl, interpFrames: dtl, q: quanta(entry, S[n - 1]), oob: oob(S[n - 1]) });
      }
      for (const e of excursions(S)) {
        stat.trExc++;
        for (let k = e.i; k <= e.j; k++) push({ kind: 'track-exc', slot: tr.slot, xuid: tr.xuid, t: S[k].t, shown: shown(S[k].t), p: S[k], before: e.before, after: e.after, run: e.j - e.i + 1, interpFrames: e.after.t - e.before.t, q: quanta(entry, S[k]), oob: oob(S[k]) });
      }
      for (let k = 1; k < n; k++) {
        const dt2 = S[k].t - S[k - 1].t; const dd = d3(S[k - 1], S[k]);
        if (dt2 > 50 && dd > DMIN) stat.trLongGapJump++;
      }
    }
  }
  // --- vehicules
  for (const v of doc.vehicles || []) {
    const S = v.samples || [];
    totVeh++; totVehSamples += S.length;
    for (const p of S) if (oob(p)) stat.vOOB++;
    if (S.length && v.spawn) {
      const ds = d3(v.spawn, S[0]);
      if (ds > DMIN) { stat.vFirst++; push({ kind: 'veh-first', slot: v.slot, family: v.family, t: S[0].t, shown: shown(S[0].t), p: S[0], spawn: v.spawn, d: ds, q: quanta(entry, S[0]), oob: oob(S[0]), n: S.length }); }
    }
    for (const e of excursions(S)) {
      stat.vExc++;
      for (let k = e.i; k <= e.j; k++) push({ kind: 'veh-exc', slot: v.slot, family: v.family, t: S[k].t, shown: shown(S[k].t), p: S[k], before: e.before, after: e.after, run: e.j - e.i + 1, interpFrames: e.after.t - e.before.t, q: quanta(entry, S[k]), oob: oob(S[k]) });
    }
    for (let k = 1; k < S.length; k++) {
      const dt2 = S[k].t - S[k - 1].t; const dd = d3(S[k - 1], S[k]);
      if (dt2 > 50 && dd > DMIN) stat.vLongGapJump++;
    }
    // echantillons hors bornes qui ne sont ni excursion ni premier
    for (const p of S) if (oob(p)) push({ kind: 'veh-oob', slot: v.slot, family: v.family, t: p.t, shown: shown(p.t), p, q: quanta(entry, p), n: S.length });
  }
  perDoc.push(stat);
  doc = null;
}

// --- sorties
const fmtP = (p) => p ? `(${p.x},${p.y},${p.z})` : '';
console.log(`# documents=${files.length} traces=${totTracks} points=${totTrackPts} vehicules=${totVeh} echantillons=${totVehSamples}`);
console.log('# PAR DOCUMENT (seulement ceux touches)');
for (const s of perDoc) {
  const touched = s.trFirst + s.trExc + s.trLast + s.trOOB + s.vExc + s.vOOB + s.vFirst;
  if (touched) console.log(`${s.id}\t${s.map}\t${s.variant}\ttrFirst=${s.trFirst} trExc=${s.trExc} trLast=${s.trLast} trOOB=${s.trOOB} trSingleOOB=${s.trSingleOOB} vFirst=${s.vFirst} vExc=${s.vExc} vOOB=${s.vOOB} vLongGapJump=${s.vLongGapJump} trLongGapJump=${s.trLongGapJump}`);
}
console.log('# LIGNES');
for (const r of rows) {
  console.log([r.id, r.map, r.kind, r.slot, r.family ?? r.xuid ?? '', `t=${r.t}`, r.shown, fmtP(r.p),
    r.q ? `q=${r.q.join('/')}` : 'q=?', r.before ? `avant=t${r.before.t}${fmtP(r.before)}` : '', r.after ? `apres=t${r.after.t}${fmtP(r.after)}` : '',
    r.next ? `suivant=t${r.next.t}${fmtP(r.next)}` : '', r.prev ? `precedent=t${r.prev.t}${fmtP(r.prev)}` : '', r.spawn ? `spawn=${fmtP(r.spawn)}` : '',
    r.run ? `run=${r.run}` : '', r.d ? `d=${r.d.toFixed(1)}` : '', r.v ? `v=${r.v.toFixed(1)}` : '', r.interpFrames !== undefined ? `interp=${(r.interpFrames / 10).toFixed(1)}s` : '',
    r.oob !== undefined ? `oob=${r.oob}` : '', r.n !== undefined ? `n=${r.n}` : ''].join('\t'));
}
const tot = perDoc.reduce((a, s) => { for (const k of Object.keys(s)) if (typeof s[k] === 'number') a[k] = (a[k] || 0) + s[k]; return a; }, {});
console.log('# TOTAUX', JSON.stringify(tot));
console.log('# DOCUMENTS TOUCHES (hors vOOB seul)', perDoc.filter((s) => s.trFirst + s.trExc + s.trLast + s.vExc + s.vFirst + s.trSingleOOB > 0).length);
