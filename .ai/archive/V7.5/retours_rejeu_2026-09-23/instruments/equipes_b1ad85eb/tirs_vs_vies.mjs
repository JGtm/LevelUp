// tirs_vs_vies.mjs — A QUI APPARTIENT L'INDEX DE TIREUR DES EVENEMENTS DE TIR ?
// Pour chaque index de tireur (record type 105, FireEvent.FilmIndex, vide par la sonde Go), la part
// de ses tirs qui tombe DANS une vie publiee de chaque joueur (tolerance 2 frames). Un index qui est
// l'index de PARTICIPANT tombe a ~100 % dans les vies du joueur de cet index ; un index qui est une
// PLACE reprise par un remplacant tombe dans les vies du remplacant.
import fs from 'fs';
import { SPD, loadDoc, trackWindow } from './panneaux_lib.mjs';

for (const id8 of process.argv.slice(2)) {
  const doc = loadDoc(id8);
  const tirs = fs.readFileSync(`${SPD}/tirs_${id8}.tsv`, 'utf8').trim().split('\n').map((l) => l.split('\t').map(Number));
  const nom = new Map((doc.roster ?? []).map((e) => [e.xuid || `bot:${e.name}`, `${e.name ?? '?'}#${e.filmIndex}`]));
  const vies = (doc.tracks ?? []).map((t) => ({ w: trackWindow(t), who: t.xuid || (t.bot ? `bot:${t.bot}` : `anon@${t.slot}`) }));
  const parIndex = new Map();
  for (const [k, f] of tirs) {
    if (!parIndex.has(k)) parIndex.set(k, { n: 0, par: new Map(), aucun: 0 });
    const e = parIndex.get(k);
    e.n++;
    const vivants = new Set(vies.filter((v) => f >= v.w.start - 2 && f <= v.w.end + 2).map((v) => v.who));
    if (vivants.size === 0) e.aucun++;
    for (const w of vivants) e.par.set(w, (e.par.get(w) ?? 0) + 1);
  }
  console.log(`== ${id8}`);
  for (const [k, e] of [...parIndex.entries()].sort((a, b) => a[0] - b[0])) {
    const top = [...e.par.entries()].sort((a, b) => b[1] - a[1]).slice(0, 3)
      .map(([w, n]) => `${nom.get(w) ?? w} ${(100 * n / e.n).toFixed(0)}%`).join(' | ');
    console.log(`  index ${String(k).padStart(2)} n=${String(e.n).padStart(3)} : ${top}${e.aucun ? ` | hors de toute vie ${e.aucun}` : ''}`);
  }
}
