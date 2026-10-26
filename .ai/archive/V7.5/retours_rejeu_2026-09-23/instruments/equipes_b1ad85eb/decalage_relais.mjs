// decalage_relais.mjs — L'HEURE API D'UN RELAIS PAR UN BOT EST-ELLE EN RETARD SUR LE FILM ?
// Pour chaque ligne API `bid(...)` arrivee EN COURS, la vie de bot (nommee au nom du bot, ou
// anonyme `index_hors_table`) qui nait le plus pres AVANT l'heure API, et celle qui nait le plus pres
// APRES. Un retard systematique de l'API se lit comme un pic d'ecarts negatifs.
import fs from 'fs';
import { REPO, loadDoc, participants, registry, trackWindow } from './panneaux_lib.mjs';

const ids = fs.readdirSync(`${REPO}/data/cache/replays/halo_infinite`)
  .filter((f) => /^[0-9a-f]{8}\.json$/.test(f)).map((f) => f.slice(0, 8)).sort();
const ecarts = [];
for (const id8 of ids) {
  const parts = participants(id8).filter((p) => p.xuid.startsWith('bid(') && p.joined_in_progress === 'true' && p.fj != null);
  if (!parts.length) continue;
  const doc = loadDoc(id8);
  const reg = registry(id8);
  const toF = (ms) => (Number(ms) - Number(reg.st) - doc.originMs) / doc.frameIntervalMs;
  const horsTable = new Set((doc.identity?.bipedSlots ?? []).filter((b) => b.link?.method === 'index_hors_table').map((b) => `${b.slot}@${b.link.from}`));
  for (const p of parts) {
    const fApi = toF(p.fj);
    const nom = `${p.gamertag} [bot]`;
    const candidates = (doc.tracks ?? []).filter((t) => t.bot === nom || (!t.xuid && !t.bot && horsTable.has(`${t.slot}@${t.startFrame ?? 0}`)))
      .map((t) => ({ d: (trackWindow(t).start - fApi) / 10, nom: t.bot ? 'nommee' : 'anonyme', slot: t.slot }));
    const avant = candidates.filter((c) => c.d <= 0).sort((a, b) => b.d - a.d)[0];
    const apres = candidates.filter((c) => c.d > 0).sort((a, b) => a.d - b.d)[0];
    ecarts.push({ id8, bot: p.gamertag, fApi: fApi.toFixed(0), avant, apres });
  }
}
for (const e of ecarts) {
  console.log(`${e.id8} ${e.bot.padEnd(18)} api f${e.fApi.padStart(5)} | avant: ${e.avant ? `${e.avant.d.toFixed(1)} s (${e.avant.nom}, slot ${e.avant.slot})` : '-'} | apres: ${e.apres ? `+${e.apres.d.toFixed(1)} s (${e.apres.nom})` : '-'}`);
}
