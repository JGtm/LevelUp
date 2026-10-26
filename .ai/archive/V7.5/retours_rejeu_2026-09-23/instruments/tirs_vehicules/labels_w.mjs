import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const want = process.argv.slice(2).map(s => s.toUpperCase().replace('0X','0x'));
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const seen = {};
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  for (const [k, v] of Object.entries(doc.weaponLabels || {})) {
    if (want.some(w => k.toUpperCase().includes(w.slice(2)))) { const key = k + ' ' + JSON.stringify(v).slice(0,200); seen[key] = (seen[key]||[]); if (seen[key].length < 5) seen[key].push(f.slice(0,8)); }
  }
}
for (const [k, v] of Object.entries(seen)) console.log(k, v.join(','));
