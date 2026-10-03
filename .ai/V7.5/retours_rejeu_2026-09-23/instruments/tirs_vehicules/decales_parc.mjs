// Tirs PUBLIES dont l'identifiant d'arme est lu DECALE (record 105/36 hors cadrage canonique).
import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const PERSO = 0x42C9679Fn, M32 = 0xFFFFFFFFn;
let tot = 0, dec = 0, docs = new Set(), parD = {}, btbJoueurs = 0, btbSansTir = 0, btbDocs = 0;
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  for (const s of doc.shots) {
    tot++;
    if (!s.w) continue;
    const W = BigInt(s.w);
    if ((W & M32) === PERSO || (W & M32) === 0n) continue;
    let d = 0; for (let k = 1; k <= 20; k++) if (((W >> BigInt(k)) & M32) === PERSO) { d = -k; break; }
    if (!d) continue;
    dec++; docs.add(f.slice(0,8)); parD[d] = (parD[d]||0)+1;
  }
  const big = (doc.roster||[]).filter(r => r.filmIndex >= 16);
  if (big.length) {
    btbDocs++; btbJoueurs += big.length;
    const xs = new Set(big.map(r => r.xuid));
    const slots = new Set(doc.tracks.filter(t => xs.has(t.xuid)).map(t => t.slot));
    // un joueur >=16 a-t-il un seul tir a pied publie ?
    for (const r of big) { const sl = new Set(doc.tracks.filter(t => t.xuid === r.xuid).map(t => t.slot)); if (!doc.shots.some(s => s.v === undefined && sl.has(s.slot))) btbSansTir++; }
  }
}
console.log('tirs publies', tot, 'dont identifiant DECALE', dec, JSON.stringify(parD), 'docs', docs.size);
console.log('docs a index >= 16 :', btbDocs, 'joueurs d index >= 16 :', btbJoueurs, 'dont SANS AUCUN tir a pied publie :', btbSansTir);
