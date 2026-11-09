// Pour chaque document : la piste SCORE de la frise serait-elle dessinee ? (regle web : >= 2 series d'equipe AVEC teamId)
import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const cat = {};
const rows = [];
for (const f of files) {
  let d; try { d = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  const teams = d.scoreTimeline?.teams ?? [];
  const ident = teams.filter(t => t.teamId != null).length;
  const unres = d.coverage?.score?.teamIdentity;
  const k = `series=${teams.length} identifiees=${ident} teamIdentity=${unres ?? 'absent'}`;
  cat[k] = (cat[k] || 0) + 1;
  if (teams.length === 1) rows.push(`${f.slice(0,8)} ${k} final=${teams[0].total?.slice(-1)[0]?.v}`);
  d = null;
}
console.log(Object.entries(cat).sort((a,b)=>b[1]-a[1]).map(([k,n])=>`${n}\t${k}`).join('\n'));
console.log('--- documents a UNE serie d equipe :');
console.log(rows.join('\n'));
