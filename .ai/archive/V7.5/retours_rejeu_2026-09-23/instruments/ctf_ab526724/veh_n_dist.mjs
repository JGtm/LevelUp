// Distribution du nombre d'echantillons de position par vie de vehicule (familles pilotables), parc entier.
import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const skip = new Set(['falcon','pelican','phantom','skiff','tourelle_auto_bannie']);
const hist = {}; const low = [];
for (const f of files) {
  let d; try { d = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  const b = d.bounds;
  for (const v of d.vehicles || []) {
    if (!v.family || skip.has(v.family)) continue;
    const n = (v.samples || []).length; const rides = (v.rides || []).length;
    const k = n === 0 ? 'n=0' : n === 1 ? 'n=1' : n <= 5 ? 'n=2..5' : 'n>5';
    const kk = `${k} rides=${rides > 0 ? '>0' : '0'}`;
    hist[kk] = (hist[kk] || 0) + 1;
    if (n <= 1) {
      const p = v.spawn || (v.samples||[])[0];
      let where = '?';
      if (p && b) { const dx = Math.max(b.minX - p.x, 0, p.x - b.maxX), dy = Math.max(b.minY - p.y, 0, p.y - b.maxY); const dz = Math.max(b.minZ - p.z, 0, p.z - b.maxZ); where = `horsXY=${Math.hypot(dx,dy).toFixed(1)} horsZ=${dz.toFixed(1)}`; }
      const span = (v.t1 ?? 0) - (v.t0 ?? 0);
      low.push(`${f.slice(0,8)} ${v.slot} ${v.family} n=${n} rides=${rides} t0=${v.t0} t1=${v.t1} vie=${(span/10).toFixed(0)}s frames=${d.frameCount} ${where}`);
    }
  }
  d = null;
}
console.log(JSON.stringify(hist));
console.log(low.join('\n'));
