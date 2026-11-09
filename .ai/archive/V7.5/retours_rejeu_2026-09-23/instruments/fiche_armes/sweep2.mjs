// Gate parc : images-cles trouees (bipedes vivants absents) sur une GRILLE complete, et vies sans arme.
import fs from 'fs';
const DIR = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(DIR).filter(f => /^[0-9a-f]{8}\.json$/.test(f)).sort();
const q = (a, p) => { const s = [...a].sort((x,y)=>x-y); return s[Math.min(s.length-1, Math.floor(p*s.length))]; };
let G = { docs:0, gridKf:0, kfNoBiped:0, kfHoled:0, kfClean:0, aliveInst:0, missInst:0, lives:0, lifeSec:0,
  livesNoLo:0, livesNoLoHole:0, livesNoLoShort:0, secNoLoHole:0, secNoLoShort:0, livesAheadOnly:0, secAhead:0,
  livesNoLoButShot:0, livesNoLoButChange:0, docsWithHole:0 };
for (const f of files) {
  const doc = JSON.parse(fs.readFileSync(DIR + f, 'utf8'));
  const fi = doc.frameIntervalMs || 100;
  const lives = doc.tracks.map(t => ({ slot: t.slot, start: t.startFrame ?? 0, end: t.endFrame ?? t.points[t.points.length-1].t }));
  const inv = doc.inventory || [], lo = doc.loadouts || [];
  const kts = [...new Set([...inv.map(e=>e.t), ...lo.map(e=>e.t)])].sort((a,b)=>a-b);
  if (kts.length < 2) continue;
  // grille : pas de 200 (10 Hz, 20 s), comblee entre deux images-cles lues, prolongee aux bords
  const grid = new Set(kts);
  for (let i = 1; i < kts.length; i++) { let d = kts[i]-kts[i-1]; if (d > 300) { const n = Math.round(d/200); for (let k = 1; k < n; k++) grid.add(Math.round(kts[i-1] + k*d/n)); } }
  for (let t = kts[0]-200; t >= 0; t -= 200) grid.add(t);
  for (let t = kts[kts.length-1]+200; t < doc.frameCount; t += 200) grid.add(t);
  const g = [...grid].sort((a,b)=>a-b);
  const invBy = new Map(); for (const e of inv) { if (!invBy.has(e.t)) invBy.set(e.t, new Set()); invBy.get(e.t).add(e.slot); }
  let docHole = false;
  for (const t of g) {
    const alive = lives.filter(l => l.start < t && l.end > t);
    if (!alive.length) continue;
    G.gridKf++;
    const pres = invBy.get(t);
    const miss = alive.filter(l => !pres || !pres.has(l.slot)).length;
    G.aliveInst += alive.length; G.missInst += miss;
    if (!pres) { G.kfNoBiped++; docHole = true; } else if (miss) { G.kfHoled++; docHole = true; } else G.kfClean++;
  }
  if (docHole) G.docsWithHole++;
  for (const l of lives) {
    const dur = (l.end - l.start) * fi / 1000; G.lives++; G.lifeSec += dur;
    const mine = lo.filter(e => e.slot === l.slot && e.t >= l.start && e.t <= l.end).map(e=>e.t);
    if (mine.length) { const first = Math.min(...mine); if (first > l.start) { G.livesAheadOnly++; G.secAhead += (first - l.start) * fi / 1000; } continue; }
    G.livesNoLo++;
    const kfIn = g.some(t => t > l.start && t < l.end);
    if (kfIn) { G.livesNoLoHole++; G.secNoLoHole += dur; } else { G.livesNoLoShort++; G.secNoLoShort += dur; }
    if ((doc.shots||[]).some(s => s.slot === l.slot && s.t >= l.start && s.t <= l.end)) G.livesNoLoButShot++;
    if ((doc.weaponChanges||[]).some(s => s.slot === l.slot && s.t >= l.start && s.t <= l.end) || (doc.pickups||[]).some(p => p.slot === l.slot && p.kind === 'weapon' && p.t >= l.start && p.t <= l.end)) G.livesNoLoButChange++;
  }
  G.docs++;
}
console.log(JSON.stringify(G, null, 1));
console.log(`images-cles de la grille avec vivants : ${G.gridKf} ; propres ${G.kfClean} ; trouees ${G.kfHoled} ; sans AUCUN bipede ${G.kfNoBiped} -> ${(100*(G.kfHoled+G.kfNoBiped)/G.gridKf).toFixed(1)} % touchees ; instants-vie manquants ${G.missInst}/${G.aliveInst} = ${(100*G.missInst/G.aliveInst).toFixed(2)} %`);
console.log(`vies ${G.lives} (${(G.lifeSec/3600).toFixed(1)} h de vie) ; SANS AUCUN loadout ${G.livesNoLo} (${(100*G.livesNoLo/G.lives).toFixed(1)} %) : dont image-cle pendant la vie ${G.livesNoLoHole} (${G.secNoLoHole.toFixed(0)} s) et vie finie avant toute image-cle ${G.livesNoLoShort} (${G.secNoLoShort.toFixed(0)} s) ; secondes de fiche sans arme ${(G.secNoLoHole+G.secNoLoShort).toFixed(0)} = ${(100*(G.secNoLoHole+G.secNoLoShort)/G.lifeSec).toFixed(1)} % du temps de vie`);
console.log(`vies a lecture « a venir » en tete : ${G.livesAheadOnly} (${G.secAhead.toFixed(0)} s = ${(100*G.secAhead/G.lifeSec).toFixed(1)} % du temps de vie) ; vies sans loadout mais avec au moins un tir ${G.livesNoLoButShot}, avec un changement/ramassage d arme ${G.livesNoLoButChange}`);
