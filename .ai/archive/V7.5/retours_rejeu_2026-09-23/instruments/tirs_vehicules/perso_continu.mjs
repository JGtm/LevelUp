// Armes personnelles a tir CONTINU (rayon de sentinelle, etc.) : portees (loadouts/pickups) mais tirees ?
import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const tir = {}, port = {}, nom = {};
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  for (const [k, v] of Object.entries(doc.weaponLabels || {})) if (k.length === 10) nom[k] = v.en;
  for (const s of doc.shots) { const k = (s.w||'').slice(0, 10); tir[k] = (tir[k]||0)+1; }
  for (const l of (doc.loadouts||[])) for (const w of (Array.isArray(l.w) ? l.w : [l.w])) { const k = ('0x' + String(w).replace(/^0x/i,'').toUpperCase()).slice(0,10); port[k] = (port[k]||0)+1; }
  for (const p of (doc.pickups||[])) { if (!p.w) continue; const k = ('0x' + String(p.w).replace(/^0x/i,'').toUpperCase()).slice(0,10); port[k] = (port[k]||0)+1; }
}
const keys = new Set([...Object.keys(nom)]);
const rows = [...keys].map(k => [k, nom[k], port[k]||0, tir[k]||0]).sort((a,b)=>a[3]-b[3]);
for (const r of rows) console.log(r.join('\t'));
