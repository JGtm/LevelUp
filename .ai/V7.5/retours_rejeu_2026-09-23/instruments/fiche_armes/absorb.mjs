// Records SAUTES mais PRESENTS : un loadout a >= 3 armes dont les armes « en trop » sont celles
// d'un bipede vivant ABSENT de la meme image-cle (lues a l'image-cle d'avant ou d'apres).
import fs from 'fs';
const DIR = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(DIR).filter(f => /^[0-9a-f]{8}\.json$/.test(f)).sort();
let tot = { big:0, explained:0, explainedByMissingAbove:0, holedKf:0, holedKfWithAbsorb:0 }; const ex = [];
for (const f of files) {
  const doc = JSON.parse(fs.readFileSync(DIR + f, 'utf8'));
  const lo = doc.loadouts || [], inv = doc.inventory || [];
  const lives = doc.tracks.map(t => ({ slot: t.slot, s: t.startFrame ?? 0, e: t.endFrame ?? t.points[t.points.length-1].t }));
  const kts = [...new Set(inv.map(i => i.t))].sort((a,b)=>a-b);
  const present = new Map(); for (const i of inv) { if (!present.has(i.t)) present.set(i.t, new Set()); present.get(i.t).add(i.slot); }
  const loAt = (slot, t) => lo.find(l => l.slot === slot && l.t === t);
  for (let k = 0; k < kts.length; k++) {
    const t = kts[k];
    const missing = lives.filter(l => l.s < t && l.e > t && !present.get(t).has(l.slot));
    if (missing.length) tot.holedKf++;
    let absorbedHere = false;
    for (const l of lo.filter(x => x.t === t && x.w.length >= 3)) {
      tot.big++;
      // familles des manquants, lues a l'image-cle voisine
      const fam = new Set(); const famAbove = new Set();
      for (const m of missing) for (const tt of [kts[k-1], kts[k+1]]) { const r = tt !== undefined && loAt(m.slot, tt); if (r) for (const w of r.w) { fam.add(w); if (m.slot > l.slot) famAbove.add(w); } }
      const own = new Set([...(loAt(l.slot, kts[k-1])?.w || []), ...(loAt(l.slot, kts[k+1])?.w || [])]);
      const extra = l.w.filter(w => !own.has(w));
      if (extra.length && extra.every(w => fam.has(w))) { tot.explained++; absorbedHere = true; if (extra.every(w => famAbove.has(w))) tot.explainedByMissingAbove++; if (ex.length < 8) ex.push(`${f.slice(0,8)} t=${t} slot ${l.slot} : ${l.w.map(w=>doc.weaponLabels[w]?.en).join('+')} | manquants ${missing.map(m=>m.slot).join(',')}`); }
    }
    if (missing.length && absorbedHere) tot.holedKfWithAbsorb++;
  }
}
console.log(JSON.stringify(tot)); for (const e of ex) console.log('  ' + e);
