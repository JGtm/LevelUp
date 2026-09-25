// Inventaire du cache film : manifeste (entrees, type 3 present, blob_prefix) vs fichiers de chunks presents.
import fs from 'fs';
const ROOT = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/';
const mans = fs.readdirSync(ROOT + 'film_manifests').filter(f => f.endsWith('.json'));
const docs = new Set(fs.readdirSync(ROOT + 'replays/halo_infinite').filter(f => f.endsWith('.json') && !f.includes('derived')).map(f => f.slice(0, 8)));
let stats = { total: 0, goWriter: 0, legacy: 0, no3: 0, no3Go: 0, filesGtMan: 0, filesGtManGo: 0, suspects: [] };
for (const f of mans) {
  const id = f.slice(0, 8);
  let m; try { m = JSON.parse(fs.readFileSync(ROOT + 'film_manifests/' + f, 'utf8')); } catch { continue; }
  const ch = m.chunks || [];
  const has3 = ch.some(c => c.chunk_type === 3);
  const maxIdx = ch.reduce((a, c) => Math.max(a, c.index), -1);
  let files = []; try { files = fs.readdirSync(ROOT + 'film_chunks/' + id).filter(x => x.startsWith('chunk_')); } catch {}
  const fileIdx = files.map(x => parseInt(x.slice(6, 8), 10)).filter(n => !isNaN(n));
  const maxFile = fileIdx.length ? Math.max(...fileIdx) : -1;
  const go = !('blob_prefix' in m);
  const mt = fs.statSync(ROOT + 'film_manifests/' + f).mtime.toISOString().slice(0, 10);
  stats.total++; if (go) stats.goWriter++; else stats.legacy++;
  if (!has3) { stats.no3++; if (go) stats.no3Go++; }
  const extra = fileIdx.filter(n => !ch.some(c => c.index === n)).length;
  if (extra > 0) { stats.filesGtMan++; if (go) stats.filesGtManGo++; }
  if (go && (!has3 || extra > 0)) stats.suspects.push(`${id} man=${ch.length} has3=${has3} maxIdx=${maxIdx} files=${files.length} maxFile=${maxFile} extraFiles=${extra} mtime=${mt} doc=${docs.has(id)}`);
}
console.log(JSON.stringify({ ...stats, suspects: undefined }));
console.log('Go-writer manifests without type 3 or with chunk files absent from the manifest:');
for (const s of stats.suspects) console.log('  ' + s);
