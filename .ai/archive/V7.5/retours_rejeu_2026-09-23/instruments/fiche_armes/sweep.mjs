// Balayage du parc : images-cles trouees et duree de vie sans arme publiee.
import fs from 'fs';
const DIR = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(DIR).filter(f => /^[0-9a-f]{8}\.json$/.test(f)).sort();
const q = (arr, p) => { if (!arr.length) return NaN; const s = [...arr].sort((a,b)=>a-b); return s[Math.min(s.length-1, Math.floor(p*(s.length)))]; };
let T = { docs:0, kf:0, kfHoled:0, kfEmptyish:0, missSlots:0, aliveSlots:0, gaps:0, lives:0, livesNoLoad:0, livesNoLoadWithKf:0, livesNoLoadShort:0, livesWithLoad:0 };
const waitsAll = [], waitsWithLoad = [], waitsNoLoadKf = [], waitsNoLoadShort = [], holeRatio = [];
const perDoc = [];
const holedDetail = [];
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(DIR + f, 'utf8')); } catch (e) { console.log('ILLISIBLE', f, e.message); continue; }
  const id = f.slice(0,8);
  const fi = doc.frameIntervalMs || 100;
  const lives = doc.tracks.map(t => ({ slot: t.slot, xuid: t.xuid, start: t.startFrame ?? 0, end: t.endFrame ?? (t.points.length ? t.points[t.points.length-1].t : 0) }));
  const inv = doc.inventory || [], lo = doc.loadouts || [];
  const kfSet = new Set([...inv.map(e=>e.t), ...lo.map(e=>e.t)]);
  const kfs = [...kfSet].sort((a,b)=>a-b);
  const invBy = new Map(), loBy = new Map();
  for (const e of inv) { if (!invBy.has(e.t)) invBy.set(e.t, new Set()); invBy.get(e.t).add(e.slot); }
  for (const e of lo) { if (!loBy.has(e.t)) loBy.set(e.t, new Set()); loBy.get(e.t).add(e.slot); }
  let d = { id, kf: kfs.length, holed: 0, miss: 0, alive: 0, gaps: 0, lives: lives.length, noLoad: 0, noLoadKf: 0, fullSlotsAvg: 0, mode: doc.coverage?.zones?.roles ?? '' , trackCount: doc.tracks.length};
  for (let i = 1; i < kfs.length; i++) if (kfs[i] - kfs[i-1] > 300) { d.gaps += Math.round((kfs[i]-kfs[i-1])/200) - 1; }
  for (const t of kfs) {
    const alive = lives.filter(l => l.start < t && l.end > t);
    const present = invBy.get(t) || new Set();
    const miss = alive.filter(l => !present.has(l.slot));
    d.alive += alive.length; d.miss += miss.length;
    if (miss.length > 0) { d.holed++; holedDetail.push({ id, t, alive: alive.length, present: present.size, missing: miss.map(l=>l.slot), presentSlots: [...present].sort((a,b)=>a-b) }); }
    if (alive.length) holeRatio.push(miss.length / alive.length);
  }
  // grille des images-cles : pas median
  const steps = []; for (let i = 1; i < kfs.length; i++) steps.push(kfs[i]-kfs[i-1]);
  const step = steps.length ? q(steps, 0.5) : 200;
  const gridNext = (s) => { // premiere image-cle de la grille >= s (extrapolee)
    for (const t of kfs) if (t >= s) return t;
    if (!kfs.length) return Infinity;
    let t = kfs[kfs.length-1]; while (t < s) t += step; return t; };
  for (const l of lives) {
    const mine = lo.filter(e => e.slot === l.slot && e.t >= l.start && e.t <= l.end).map(e=>e.t);
    const first = mine.length ? Math.min(...mine) : null;
    const dur = l.end - l.start;
    const wait = (first ?? l.end) - l.start;
    waitsAll.push(wait * fi / 1000);
    if (first !== null) { T.livesWithLoad++; waitsWithLoad.push(wait * fi / 1000); }
    else {
      d.noLoad++;
      const g = gridNext(l.start + 1);
      if (g < l.end) { d.noLoadKf++; waitsNoLoadKf.push(dur * fi / 1000); }
      else { T.livesNoLoadShort++; waitsNoLoadShort.push(dur * fi / 1000); }
    }
  }
  T.docs++; T.kf += d.kf; T.kfHoled += d.holed; T.missSlots += d.miss; T.aliveSlots += d.alive; T.gaps += d.gaps; T.lives += d.lives; T.livesNoLoad += d.noLoad; T.livesNoLoadWithKf += d.noLoadKf;
  perDoc.push(d);
}
console.log('TOTAL', JSON.stringify(T));
console.log('images-cles trouees', T.kfHoled, '/', T.kf, (100*T.kfHoled/T.kf).toFixed(1)+'%', '; slots vivants absents', T.missSlots, '/', T.aliveSlots, (100*T.missSlots/T.aliveSlots).toFixed(2)+'%', '; images-cles manquantes (trous de grille)', T.gaps);
console.log('vies', T.lives, 'avec loadout', T.livesWithLoad, 'sans aucun loadout', T.livesNoLoad, '(dont image-cle pendant la vie', T.livesNoLoadWithKf, '; vie finie avant la 1re image-cle', T.livesNoLoadShort, ')');
const fmt = (a) => `n=${a.length} p10=${q(a,.1)?.toFixed(1)} p25=${q(a,.25)?.toFixed(1)} med=${q(a,.5)?.toFixed(1)} p75=${q(a,.75)?.toFixed(1)} p90=${q(a,.9)?.toFixed(1)} max=${q(a,1-1e-9)?.toFixed(1)} moy=${(a.reduce((s,x)=>s+x,0)/a.length).toFixed(2)}`;
console.log('attente naissance -> 1re arme publiee (s), toutes vies :', fmt(waitsAll));
console.log('  vies avec loadout :', fmt(waitsWithLoad));
console.log('  vies sans loadout MALGRE une image-cle (duree de vie) :', fmt(waitsNoLoadKf));
console.log('  vies sans loadout, finies avant la 1re image-cle (duree de vie) :', fmt(waitsNoLoadShort));
const sumAll = waitsAll.reduce((s,x)=>s+x,0);
console.log('secondes de vie sans arme publiee (somme)', sumAll.toFixed(0));
fs.writeFileSync(process.argv[2] || 'sweep_out.json', JSON.stringify({ T, perDoc, holedDetail }, null, 1));
perDoc.sort((a,b)=>b.miss-a.miss);
console.log('pires documents (slots absents) :'); for (const d of perDoc.slice(0,15)) console.log(`  ${d.id} kf=${d.kf} trouees=${d.holed} absents=${d.miss}/${d.alive} trous_grille=${d.gaps} vies=${d.lives} sans_loadout=${d.noLoad} (malgre_kf=${d.noLoadKf}) roles=${d.mode}`);
