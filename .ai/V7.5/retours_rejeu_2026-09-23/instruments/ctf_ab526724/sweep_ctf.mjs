// Balayage SEQUENTIEL (un document a la fois) des documents de rejeu CTF + etat du cache film.
import fs from 'fs';
const ROOT = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/';
const ids = process.argv.slice(2);
const rows = [];
for (const id of ids) {
  const p = `${ROOT}replays/halo_infinite/${id}.json`;
  if (!fs.existsSync(p)) { console.log(id, 'NO DOC'); continue; }
  const st = fs.statSync(p);
  let doc = JSON.parse(fs.readFileSync(p, 'utf8'));
  const c = doc.coverage || {};
  const fc = c.flagCarries || {};
  const ob = c.objectives || {};
  const oo = c.objectiveObjects || {};
  const sc = c.score || {};
  const br = c.bridge || {};
  const sb = doc.identity?.coverage?.statborgSlot || {};
  const ft = doc.identity?.coverage?.filmTable || {};
  const teams = doc.scoreTimeline?.teams || [];
  const teamDesc = teams.map(t => `${t.teamId ?? '?'}:${(t.total||[]).length ? t.total[t.total.length-1].v : '-'}`).join('/');
  // manifeste + fichiers
  let man = null; try { man = JSON.parse(fs.readFileSync(`${ROOT}film_manifests/${id}.json`, 'utf8')); } catch {}
  let files = []; try { files = fs.readdirSync(`${ROOT}film_chunks/${id}`).filter(f => f.startsWith('chunk_')); } catch {}
  const manN = man?.chunks?.length ?? -1;
  const has3 = man?.chunks?.some(ch => ch.chunk_type === 3) ?? false;
  const factsP = `${ROOT}film_facts/halo_infinite/${id}.filmfacts.bin`;
  const facts = fs.existsSync(factsP) ? fs.statSync(factsP).mtime.toISOString().slice(0,16) : '-';
  rows.push({
    id, built: st.mtime.toISOString().slice(0,16), schema: doc.schemaVersion, frames: doc.frameCount,
    flag: (doc.flagCarries||[]).length, obj: (doc.objectives||[]).length, oobj: (doc.objectiveObjects||[]).length,
    fz: doc.flagReturnZone ? 1 : 0,
    fcFilm: fc.flagFilm, open: fc.openings, carr: fc.carries, noBr: fc.noBridge, oow: fc.outOfWindow, noTr: fc.noTrack, spawns: fc.spawns, objL: fc.objectLives,
    obA: ob.available, obAt: ob.attached, obNo: ob.noSlot,
    ooDecl: oo.declared, ooLives: oo.lives,
    teamId: sc.teamIdentity, pts: sc.points, trunc: sc.truncated, teams: teamDesc,
    sbD: sb.deduit, sbNR: sb.non_resolu, ftAcc: ft.accord, ftSil: ft.silence,
    offM: br.deathOffsetMatched, offMs: br.deathOffsetMs, livesNamed: br.livesNamed, livesTot: br.livesTotal,
    verdict: c.verdict?.bridge?.slice(0,20),
    manN, has3, files: files.length, facts,
  });
  doc = null;
}
const keys = Object.keys(rows[0]);
console.log(keys.join('\t'));
for (const r of rows) console.log(keys.map(k => r[k] === undefined ? '' : String(r[k])).join('\t'));
