#!/usr/bin/env node
// anonymiser.mjs - remplace pseudos de joueurs et chemins du poste dans les DOCUMENTS suivis du depot.
//
// Usage (depuis la racine du depot ; Node, aucune dependance) :
//
//   node scripts/anonymisation/anonymiser.mjs --table <table.tsv> --rapport <dossier>   (simulation)
//   node scripts/anonymisation/anonymiser.mjs --table <table.tsv> --rapport <dossier> --appliquer
//
// Options :
//   --table <f>      table de correspondance (format : table.mjs). OBLIGATOIRE. Doit etre HORS du depot.
//   --rapport <d>    dossier de sortie du rapport. Doit etre HORS du depot (il cite les vrais pseudos
//                    dans ecartes.tsv). Sans lui, seul un resume sans pseudo s'affiche.
//   --simulation     (defaut) n'ecrit aucun fichier du depot.
//   --appliquer      reecrit les fichiers. Refuse si la table ou le rapport est sous la racine.
//   --racine <d>     racine du depot (defaut : dossier courant).
//   --inclure <glob> ne garde que ces chemins (repetable ; `**`, `*`, `?`).
//   --exclure <glob> ecarte ces chemins en plus de la liste par defaut (repetable).
//   --chemins <f>    fichier de lignes `+glob` (inclusion) et `-glob` (exclusion).
//   --profondeur <n> profondeur des dossiers du resume (defaut 2).
//
// Garanties : fichiers suivis seulement ; jamais de binaire, de fichier genere ni d'UTF-8 invalide ;
// fins de ligne et BOM conserves ; idempotent (un second passage ne change rien).

import fs from 'node:fs';
import path from 'node:path';
import { parseArgs } from 'node:util';
import { pathToFileURL } from 'node:url';
import { lireTableFichier } from './table.mjs';
import { compiler, remplacerTexte, verifierIdempotence } from './remplacer.mjs';
import { choisirFichiers, decoder, encoder, estSous, genreDeFichier, lireListeChemins, lireOctets } from './fichiers.mjs';

const OPTIONS = {
  table: { type: 'string' },
  rapport: { type: 'string' },
  simulation: { type: 'boolean', default: false },
  appliquer: { type: 'boolean', default: false },
  racine: { type: 'string', default: '.' },
  inclure: { type: 'string', multiple: true, default: [] },
  exclure: { type: 'string', multiple: true, default: [] },
  chemins: { type: 'string' },
  profondeur: { type: 'string', default: '2' },
  aide: { type: 'boolean', default: false },
};

function dossierDe(fichier, profondeur) {
  const parties = fichier.split('/');
  if (parties.length === 1) return '.';
  return parties.slice(0, Math.min(profondeur, parties.length - 1)).join('/');
}

/** Parcourt les fichiers choisis ; rend le bilan par fichier. N'ecrit rien. */
export function analyser(racine, fichiers, moteurs) {
  const bilans = [];
  const ignores = [];
  for (const f of fichiers) {
    const d = decoder(lireOctets(racine, f));
    if (d.ignore) {
      ignores.push({ fichier: f, raison: d.ignore });
      continue;
    }
    const genre = genreDeFichier(f);
    const r = remplacerTexte(d.texte, moteurs[genre], f);
    if (r.occurrences.length > 0) bilans.push({ fichier: f, genre, ...r, bom: d.bom });
  }
  return { bilans, ignores };
}

/** Le type d'une occurrence pour le rapport : un xuid est un `texte` dont le remplacement a 16 chiffres. */
function typeAffiche(o) {
  return o.type === 'texte' && /^[0-9]{16}$/.test(o.label) ? 'xuid' : o.type;
}

function resumerParType(bilans) {
  const par = new Map();
  for (const b of bilans) {
    for (const t of new Set(b.occurrences.map(typeAffiche))) {
      const occ = b.occurrences.filter((o) => typeAffiche(o) === t);
      const e = par.get(t) ?? { fichiers: 0, lignes: 0, occurrences: 0, ecartes: 0 };
      const rempl = occ.filter((o) => o.statut === 'remplace');
      if (rempl.length > 0) e.fichiers++;
      e.lignes += new Set(rempl.map((o) => o.ligne)).size;
      e.occurrences += rempl.length;
      e.ecartes += occ.length - rempl.length;
      par.set(t, e);
    }
  }
  return [...par].map(([t, e]) => [t, e.fichiers, e.lignes, e.occurrences, e.ecartes].join('\t'));
}

function lignesDistinctes(occurrences, statut) {
  return new Set(occurrences.filter((o) => o.statut === statut).map((o) => o.ligne)).size;
}

function resumerParDossier(bilans, profondeur) {
  const par = new Map();
  for (const b of bilans) {
    const k = dossierDe(b.fichier, profondeur);
    const e = par.get(k) ?? { fichiers: 0, lignes: 0, occurrences: 0, ecartes: 0 };
    const rempl = lignesDistinctes(b.occurrences, 'remplace');
    if (rempl > 0) e.fichiers++;
    e.lignes += rempl;
    e.occurrences += b.occurrences.filter((o) => o.statut === 'remplace').length;
    e.ecartes += b.occurrences.filter((o) => o.statut === 'ecarte').length;
    par.set(k, e);
  }
  return [...par.entries()].sort((a, b) => b[1].lignes - a[1].lignes);
}

function ecrireRapport(dossier, { bilans, ignores, nomsDeFichiers }, profondeur) {
  fs.mkdirSync(dossier, { recursive: true });
  const w = (nom, lignes) => fs.writeFileSync(path.join(dossier, nom), lignes.join('\n') + '\n');
  const dossiers = resumerParDossier(bilans, profondeur);
  w('par_dossier.tsv', ['dossier\tfichiers\tlignes\toccurrences\tecartes', ...dossiers.map(([k, e]) => `${k}\t${e.fichiers}\t${e.lignes}\t${e.occurrences}\t${e.ecartes}`)]);
  w('fichiers.tsv', [
    'fichier\tgenre\tlignes\toccurrences\tecartes\ttypes',
    ...bilans.map((b) => {
      const rempl = b.occurrences.filter((o) => o.statut === 'remplace');
      const ecart = b.occurrences.filter((o) => o.statut === 'ecarte').length;
      return [b.fichier, b.genre, lignesDistinctes(b.occurrences, 'remplace'), rempl.length, ecart, [...new Set(b.occurrences.map(typeAffiche))].join(',')].join('\t');
    }),
  ]);
  w('par_type.tsv', ['type\tfichiers\tlignes\toccurrences\tecartes', ...resumerParType(bilans)]);
  const ecartes = bilans.flatMap((b) => b.occurrences.filter((o) => o.statut === 'ecarte').map((o) => `${b.fichier}\t${o.ligne}\t${o.label}\t${o.extrait.replace(/\t/g, ' ')}`));
  w('ecartes.tsv', ['fichier\tligne\tlabel\textrait', ...ecartes]);
  const code = bilans.filter((b) => b.genre === 'code').flatMap((b) => b.occurrences.filter((o) => o.statut === 'remplace').map((o) => `${b.fichier}\t${o.ligne}\t${typeAffiche(o)}\t${o.label}\t${o.original}\t${o.extrait.replace(/\t/g, ' ')}`));
  w('code.tsv', ['fichier\tligne\ttype\tremplacement\toriginal\textrait', ...code]);
  const parLabel = new Map();
  for (const b of bilans) {
    for (const o of b.occurrences.filter((x) => x.statut === 'remplace')) {
      const e = parLabel.get(o.label) ?? { fichiers: new Set(), lignes: new Set(), n: 0 };
      e.fichiers.add(b.fichier);
      e.lignes.add(`${b.fichier}:${o.ligne}`);
      e.n++;
      parLabel.set(o.label, e);
    }
  }
  w('par_remplacement.tsv', ['remplacement\tfichiers\tlignes\toccurrences', ...[...parLabel.entries()].sort((a, b) => b[1].lignes.size - a[1].lignes.size).map(([l, e]) => `${l}\t${e.fichiers.size}\t${e.lignes.size}\t${e.n}`)]);
  w('ignores.tsv', ['fichier\traison', ...ignores.map((i) => `${i.fichier}\t${i.raison}`)]);
  w('noms_de_fichiers.tsv', ['fichier_dont_le_chemin_porte_une_cible', ...nomsDeFichiers]);
  return dossiers;
}

function nomsDeFichiersCibles(racine, moteurs) {
  const { retenus } = choisirFichiers(racine);
  return retenus.filter((f) => remplacerTexte(f, moteurs[genreDeFichier(f)], '').occurrences.some((o) => o.statut === 'remplace'));
}

function ecrireFichiers(racine, bilans, moteurs) {
  let n = 0;
  for (const b of bilans) {
    if (!b.occurrences.some((o) => o.statut === 'remplace')) continue;
    const octets = encoder(b.texte, b.bom);
    const second = remplacerTexte(b.texte, moteurs[b.genre], b.fichier);
    if (second.occurrences.some((o) => o.statut === 'remplace')) {
      throw new Error(`${b.fichier} : le second passage change encore le texte (non idempotent), rien d'autre n'est ecrit`);
    }
    fs.writeFileSync(path.join(racine, b.fichier), octets);
    n++;
  }
  return n;
}

export function executer(argv) {
  const { values: v } = parseArgs({ args: argv, options: OPTIONS });
  if (v.aide || !v.table) {
    console.log(fs.readFileSync(new URL(import.meta.url), 'utf8').split('\n').slice(1, 24).map((l) => l.replace(/^\/\/ ?/, '')).join('\n'));
    return v.aide ? 0 : 2;
  }
  const racine = path.resolve(v.racine);
  if (v.simulation && v.appliquer) throw new Error('--simulation et --appliquer s\'excluent');
  if (estSous(racine, v.table)) throw new Error('la table contient de vrais pseudos : elle doit etre hors du depot');
  if (v.rapport && estSous(racine, v.rapport)) throw new Error('le rapport cite de vrais pseudos : il doit etre hors du depot');
  const entrees = lireTableFichier(v.table);
  verifierIdempotence(entrees, compiler(entrees));
  const moteurs = {
    docs: compiler(entrees.filter((e) => !e.drapeaux.has('code'))),
    code: compiler(entrees.filter((e) => !e.drapeaux.has('docs'))),
  };
  const liste = v.chemins ? lireListeChemins(fs.readFileSync(v.chemins, 'utf8')) : { inclure: [], exclure: [] };
  const { retenus, ignores: exclus } = choisirFichiers(racine, {
    inclure: [...v.inclure, ...liste.inclure],
    exclure: [...v.exclure, ...liste.exclure],
  });
  const { bilans, ignores } = analyser(racine, retenus, moteurs);
  const profondeur = Number.parseInt(v.profondeur, 10);
  const dossiers = v.rapport
    ? ecrireRapport(v.rapport, { bilans, ignores, nomsDeFichiers: nomsDeFichiersCibles(racine, moteurs) }, profondeur)
    : resumerParDossier(bilans, profondeur);
  const totalLignes = dossiers.reduce((s, [, e]) => s + e.lignes, 0);
  const totalFichiers = dossiers.reduce((s, [, e]) => s + e.fichiers, 0);
  const ecartes = dossiers.reduce((s, [, e]) => s + e.ecartes, 0);
  console.log(`${retenus.length} fichiers lus, ${exclus.exclusion} exclus, ${exclus.binaire} binaires, ${ignores.length} ignores (binaire/UTF-16/non UTF-8)`);
  for (const [k, e] of dossiers) console.log(`${String(e.lignes).padStart(8)} lignes ${String(e.fichiers).padStart(5)} fichiers  ${k}`);
  console.log(`TOTAL ${totalLignes} lignes dans ${totalFichiers} fichiers ; ${ecartes} occurrences ambigues ecartees`);
  if (v.appliquer) {
    console.log(`${ecrireFichiers(racine, bilans, moteurs)} fichiers reecrits`);
  } else {
    console.log('simulation : aucun fichier modifie');
  }
  return 0;
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try {
    process.exitCode = executer(process.argv.slice(2));
  } catch (e) {
    console.error(`erreur : ${e.message}`);
    process.exitCode = 1;
  }
}
