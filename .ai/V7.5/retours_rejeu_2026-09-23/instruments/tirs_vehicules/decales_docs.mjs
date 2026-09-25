import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const PERSO = 0x42C9679Fn, M32 = 0xFFFFFFFFn;
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  let n = 0; for (const s of doc.shots) { if (!s.w) continue; const W = BigInt(s.w); if ((W & M32) === PERSO || (W & M32) === 0n) continue; if (((W >> 5n) & M32) === PERSO || ((W >> 3n) & M32) === PERSO) n++; }
  if (!n) continue;
  const bots = (doc.tracks||[]).filter(t => !t.xuid || /^bid|bot/i.test(String(t.xuid))).length;
  const nomsBots = (doc.roster||[]).filter(r => !r.xuid || /bid/i.test(r.xuid)).length;
  console.log(f.slice(0,8), 'decales', n, 'roster', (doc.roster||[]).length, 'rosterBots', nomsBots, 'tracesSansXuid/bot', bots, 'tirs', doc.shots.length);
}
