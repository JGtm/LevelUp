import fs from 'fs';
const id = process.argv[2];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const cases = JSON.parse(process.argv[3]); // [[slot, frame, famHex]]
for (const [slot, f, fam] of cases) {
  const tr = doc.tracks.find(t => t.slot === slot);
  const los = doc.loadouts.filter(l => l.slot === slot);
  const past = los.filter(l => l.t <= f), fut = los.filter(l => l.t > f);
  const lab = doc.weaponLabels['0x' + fam.toUpperCase()]?.en ?? fam;
  const pk = doc.pickups.filter(p => p.slot === slot && Math.abs(p.t - f) <= 20).map(p => `${p.t}:${p.family}`);
  const shots = doc.shots.filter(s => s.slot === slot && s.w.startsWith('0x' + fam.toUpperCase())).map(s => s.t);
  console.log(`slot ${slot} vie [${tr?.startFrame}..${tr?.endFrame}] emission ${f} ${lab} ; loadouts passes ${past.map(l=>l.t+':'+l.w.map(w=>doc.weaponLabels[w]?.en).join('+')).join(' ')||'AUCUN'} ; futurs ${fut.slice(0,1).map(l=>l.t+':'+l.w.map(w=>doc.weaponLabels[w]?.en).join('+')).join(' ')} ; ramassages +-2s ${pk.join(' ')||'-'} ; tirs de cette arme ${shots.length ? Math.min(...shots)+'..'+Math.max(...shots) : '-'}`);
}
