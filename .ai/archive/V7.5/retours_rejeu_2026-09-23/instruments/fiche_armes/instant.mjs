// Vies SANS loadout : a partir de quand un canal a l'instant (tir, prise, ramassage) dit une arme ?
import fs from 'fs';
const DIR = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(DIR).filter(f => /^[0-9a-f]{8}\.json$/.test(f)).sort();
let S = { lives:0, sec:0, secCovShot:0, secCovChange:0, secCovAny:0, livesAny:0, firstEvidenceDelays:[] };
for (const f of files) {
  const doc = JSON.parse(fs.readFileSync(DIR + f, 'utf8'));
  const fi = doc.frameIntervalMs || 100;
  const lo = doc.loadouts || [];
  for (const t of doc.tracks) {
    const s = t.startFrame ?? 0, e = t.endFrame ?? t.points[t.points.length-1].t;
    if (lo.some(l => l.slot === t.slot && l.t >= s && l.t <= e)) continue;
    S.lives++; const dur = (e - s) * fi / 1000; S.sec += dur;
    const shots = (doc.shots||[]).filter(x => x.slot === t.slot && x.t >= s && x.t <= e).map(x => x.t);
    const chg = [...(doc.weaponChanges||[]).filter(x => x.slot === t.slot && x.t >= s && x.t <= e && x.w).map(x=>x.t),
                 ...(doc.pickups||[]).filter(x => x.slot === t.slot && x.kind === 'weapon' && x.t >= s && x.t <= e).map(x=>x.t)];
    const fS = shots.length ? Math.min(...shots) : null, fC = chg.length ? Math.min(...chg) : null;
    if (fS !== null) S.secCovShot += (e - fS) * fi / 1000;
    if (fC !== null) S.secCovChange += (e - fC) * fi / 1000;
    const fA = [fS, fC].filter(x => x !== null);
    if (fA.length) { const a = Math.min(...fA); S.secCovAny += (e - a) * fi / 1000; S.livesAny++; S.firstEvidenceDelays.push((a - s) * fi / 1000); }
  }
}
const d = S.firstEvidenceDelays.sort((a,b)=>a-b); const med = d[Math.floor(d.length/2)];
console.log(`vies sans loadout ${S.lives} (${S.sec.toFixed(0)} s) ; couvertes a partir du 1er tir : ${S.secCovShot.toFixed(0)} s ; du 1er changement/ramassage : ${S.secCovChange.toFixed(0)} s ; de l un ou l autre : ${S.secCovAny.toFixed(0)} s (${(100*S.secCovAny/S.sec).toFixed(1)} %), ${S.livesAny} vies ; delai median naissance -> 1re preuve ${med?.toFixed(1)} s`);
