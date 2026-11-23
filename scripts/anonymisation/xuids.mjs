// xuids.mjs - les identifiants de joueur (xuids, 16 chiffres decimaux) d'un depot, et leurs fictifs.
//
// Un xuid reel se reconnait mal par sa forme : un fictif a la meme (16 chiffres, meme plage). On le
// reconnait donc par EXCLUSION : ne sont pas reels les xuids des registres de fictifs versionnes
// (colonne decimale de `IDENTITES_APPS.tsv`) ni les valeurs evidentes (zeros de remplissage, chiffre
// repete, suite 1234567...). Les autres sont "probables reels".
//
// Le fictif d'un xuid est le meme que celui des fixtures de film : la fonction de
// `apps/go-api/internal/identitefictive` (SHA-256 d'une graine publique et du xuid sur 8 octets
// gros-boutiste, 48 bits, plage `[0x0009000000000000, 0x000A000000000000[`). Un xuid qui figure a la
// fois dans un document et dans une fixture recoit donc le meme fictif partout.

import { createHash } from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { decoder } from './fichiers.mjs';

const GRAINE = 'levelup/identifiants-de-test-fictifs/v1';
const PLAGE_BAS = 0x0009000000000000n;
const MASQUE_48 = (1n << 48n) - 1n;

/** Le xuid fictif (chaine decimale de 16 chiffres) d'un xuid reel (chaine decimale). */
export function xuidFictif(reel) {
  const entree = Buffer.alloc(8);
  entree.writeBigUInt64BE(BigInt(reel));
  const h = createHash('sha256').update(GRAINE).update(entree).digest();
  return String(PLAGE_BAS + (h.readBigUInt64BE(0) & MASQUE_48));
}

/** Valeur evidemment inventee : zeros de remplissage (`2533274000000101`), chiffre repete, suite croissante. */
export function xuidEvident(x) {
  return /^(?:2533274[89]0{6,}\d{0,2}|25\d{2}0{8,}\d*|25\d{5}0{5,}\d{1,5})$/.test(x) || /(\d)\1{6,}/.test(x) || /1234567/.test(x);
}

/** Les xuids decimaux d'un registre de fictifs (`xuid<TAB>hex` par ligne, `#` ignore). */
export function lireRegistre(chemin) {
  const out = new Set();
  for (const l of fs.readFileSync(chemin, 'utf8').split(/\r?\n/)) {
    const m = /^(\d{16})\t/.exec(l);
    if (m) out.add(m[1]);
  }
  return out;
}

/** Parcourt les fichiers texte : Map xuid -> { fichiers:Set, occurrences, classe }. */
export function inventorierXuids(racine, fichiers, registre, gardes = new Map()) {
  const par = new Map();
  for (const f of fichiers) {
    const d = decoder(fs.readFileSync(path.join(racine, f)));
    if (d.ignore) continue;
    for (const m of d.texte.matchAll(/(?<![0-9])(?<![0-9][.])(25[0-9]{14})(?![0-9])/g)) {
      const e = par.get(m[1]) ?? { fichiers: new Set(), occurrences: 0 };
      e.fichiers.add(f);
      e.occurrences++;
      par.set(m[1], e);
    }
  }
  for (const [x, e] of par) e.classe = gardes.get(x) ?? classeXuid(x, registre);
  return par;
}

function classeXuid(x, registre) {
  if (registre.has(x)) return 'fictif-registre';
  if (xuidEvident(x)) return 'fictif-evident';
  return /^25(?:33|35)/.test(x) ? 'probable-reel' : 'autre';
}

/** Lignes de table (`texte`, limites de mot) pour les xuids a remplacer. Les `autre` (prefixe hors
 *  2533 et 2535) sont ecrits en commentaire : a relire. Un xuid qui est le fictif d'un autre candidat
 *  est la sortie d'une anonymisation anterieure : il n'est pas remplace, ce qui garde la table
 *  idempotente. Refuse deux originaux de meme fictif. */
export function lignesTableXuids(par) {
  const candidats = [...par].filter(([, e]) => e.classe === 'probable-reel' || e.classe === 'autre');
  const fictifs = new Set(candidats.map(([x]) => xuidFictif(x)));
  const aRemplacer = candidats.filter(([x]) => !fictifs.has(x)).sort((p, q) => p[0].localeCompare(q[0]));
  const sorties = new Set();
  const out = [];
  for (const [x, e] of aRemplacer) {
    const fictif = xuidFictif(x);
    if (sorties.has(fictif)) throw new Error('deux xuids reels ont le meme fictif : changer la graine');
    sorties.add(fictif);
    const ligne = ['texte', x, fictif, 'mot'].join('\t');
    out.push(e.classe === 'autre' ? `# xuid hors 2533/2535, a relire : ${ligne}` : ligne);
  }
  return out;
}
