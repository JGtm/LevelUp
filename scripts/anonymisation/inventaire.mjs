#!/usr/bin/env node
// inventaire.mjs - l'inventaire des pseudos d'un depot, et la table de correspondance qui en decoule.
//
//   node scripts/anonymisation/inventaire.mjs --sortie <dossier> [--gardes <f>] [--table <f>] [--racine <d>]
//
// Rend dans `--sortie` (HORS depot : les fichiers citent de vrais pseudos) :
//   pseudos.tsv   un pseudo par ligne : classe, raison, fichiers, lignes, fichiers de code, contextes,
//                 mot courant, forme courte. Classes : voir classes.mjs.
//   bruit.tsv     les candidats ecartes (pas des pseudos) et la raison : a parcourir pour y reperer un
//                 vrai pseudo mal classe.
//   xuids.tsv     (avec --xuids) un xuid par ligne : classe, fichiers, occurrences ; les xuids probables
//                 reels sont ajoutes a table.tsv avec leur fictif (voir xuids.mjs ; --registre).
//   table.tsv     la table de correspondance (format : table.mjs) pour les classes reel et reel-doute :
//                 `joueur_NNN` ; avec `--table`, une table existante est conservee et seuls les
//                 nouveaux pseudos s'y ajoutent (les libelles restent stables d'un passage a l'autre).
//
// `--gardes` : fichier `classe<TAB>nom` ou `classe<TAB>~expression` (proprietaire, auteur, bot, fictif,
// invente, bruit) ; voir classes.mjs.

import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { parseArgs } from 'node:util';
import { pathToFileURL } from 'node:url';
import { chercherCandidats } from './candidats.mjs';
import { affiner, classer, deriveDUnAutre, formeCourte, lireGardes, lireGardesFichier } from './classes.mjs';
import { compiler, remplacerTexte } from './remplacer.mjs';
import { decoder, estSous, genreDeFichier } from './fichiers.mjs';
import { lireTableFichier } from './table.mjs';
import { inventorierXuids, lignesTableXuids, lireRegistre } from './xuids.mjs';

const SEUIL_MOT_COURANT = 5;
const SEUIL_DIFFUSION = 30;
const SEUIL_JOUEUR_SUIVI = 100;
const SEUIL_SURNOM = 5;
const SEUIL_FICHIERS_SUIVI = 20;

/** Un passage sur tous les fichiers texte : frequence des mots en minuscules (de 3 lettres et plus). */
function frequencesDeMots(racine, fichiers) {
  const freq = new Map();
  for (const f of fichiers) {
    const d = decoder(fs.readFileSync(path.join(racine, f)));
    if (d.ignore) continue;
    for (const m of d.texte.matchAll(/(?<![\p{L}\p{N}_])[a-z]{3,}(?![\p{L}\p{N}_])/gu)) {
      freq.set(m[0], (freq.get(m[0]) ?? 0) + 1);
    }
  }
  return freq;
}

function fichiersTexte(racine) {
  return execFileSync('git', ['ls-files', '-z'], { cwd: racine, maxBuffer: 1 << 30 })
    .toString('utf8').split('\0').filter(Boolean)
    .filter((f) => !f.startsWith('scripts/anonymisation/') && !/\.(webp|png|jpe?g|gif|ico|ogg|wav|woff2?|gz|bin|duckdb|wal|sig|zst|zip|blob|mvar)$/i.test(f));
}

/** Compte, pour chaque nom, les fichiers et lignes ou il apparait comme MOT ENTIER (tout le depot). */
function compterOccurrences(racine, fichiers, noms) {
  const entrees = noms.map((n, i) => ({ type: 'pseudo', original: n, remplacement: `x_${i}`, drapeaux: new Set(), ligne: i }));
  const moteur = compiler(entrees);
  const par = new Map(noms.map((n) => [n, { fichiers: new Set(), lignes: new Set(), code: new Set() }]));
  for (const f of fichiers) {
    const d = decoder(fs.readFileSync(path.join(racine, f)));
    if (d.ignore) continue;
    const { occurrences } = remplacerTexte(d.texte, moteur, f);
    for (const o of occurrences) {
      const e = par.get(noms[Number(o.label.slice(2))]);
      e.fichiers.add(f);
      e.lignes.add(`${f}:${o.ligne}`);
      if (genreDeFichier(f) === 'code') e.code.add(f);
    }
  }
  return par;
}

/** Extrait les libelles deja attribues d'une table existante, et le plus grand numero. */
function numerotation(tableExistante) {
  const connus = new Map();
  let max = 0;
  for (const e of tableExistante) {
    if (e.type !== 'pseudo') continue;
    connus.set(e.original, e);
    const m = /^joueur_(\d+)$/.exec(e.remplacement);
    if (m) max = Math.max(max, Number(m[1]));
  }
  return { connus, max };
}

/** Un nom en lettres seules, capitalise, dont la forme minuscule est un mot courant du depot. */
function ressembleAUnMot(nom, mots) {
  return /^[A-Z][A-Za-z]*$/.test(nom) && (mots.get(nom.toLowerCase()) ?? 0) >= SEUIL_MOT_COURANT;
}

const estReel = (c) => c.classe === 'reel' || c.classe === 'reel-doute' || c.classe === 'reel-prose';

/** Classement en trois temps : preuves de contexte, frequence des mots, puis derives d'un pseudo. */
function classerTout(candidats, gardes, mots, comptes) {
  const classes = new Map();
  for (const [nom, info] of candidats) classes.set(nom, { ...classer(nom, info, gardes), info });
  for (const [nom, c] of classes) {
    if (c.classe === 'reel-doute' && (mots.get(nom.toLowerCase()) ?? 0) >= SEUIL_MOT_COURANT) {
      classes.set(nom, { ...c, classe: 'mot-courant', raison: 'indice faible et mot courant' });
    }
  }
  const largeDiffusion = (nom) => (comptes.get(nom)?.fichiers.size ?? 0) >= SEUIL_DIFFUSION;
  for (const [nom, c] of classes) classes.set(nom, affiner(nom, c, mots, largeDiffusion(nom), comptes.get(nom)?.lignes.size ?? 0));
  const surs = new Set([...classes].filter(([n, c]) => c.classe === 'reel' && !n.includes(' ')).map(([n]) => n));
  for (const [nom, c] of classes) {
    if (c.classe === 'reel' && deriveDUnAutre(nom, surs)) classes.set(nom, { ...c, classe: 'bruit', raison: 'derive d un pseudo reel' });
  }
  return classes;
}

/** Les surnoms des joueurs suivis : un debut de leur pseudo (au moins 4 lettres, 2 de moins que le
 *  pseudo, qui n'est pas un mot courant) cite au moins 5 fois dans le depot, a la casse exacte. Seuls les
 *  joueurs suivis (100 lignes et 20 fichiers) en ont. Map surnom -> pseudo. */
function chercherSurnoms(racine, fichiers, reels, classes, comptes, mots) {
  const suivis = reels.filter((n) => classes.get(n).classe === 'reel' && (comptes.get(n)?.lignes.size ?? 0) >= SEUIL_JOUEUR_SUIVI && (comptes.get(n)?.fichiers.size ?? 0) >= SEUIL_FICHIERS_SUIVI);
  const candidats = new Map();
  for (const n of suivis) {
    for (let k = 4; k <= n.length - 2; k++) {
      const p = n.slice(0, k);
      const deja = classes.get(p)?.classe;
      if (p === formeCourte(n) || /\s$/.test(p) || (mots.get(p.toLowerCase()) ?? 0) >= SEUIL_MOT_COURANT || (deja && deja !== 'mot-courant' && deja !== 'reel-doute')) continue;
      candidats.set(p, n);
    }
  }
  const comptes2 = compterOccurrences(racine, fichiers, [...candidats.keys()]);
  return new Map([...candidats].filter(([p]) => comptes2.get(p).lignes.size >= SEUIL_SURNOM));
}

export function inventorier({ racine, gardes, tableExistante = [] }) {
  const fichiers = fichiersTexte(racine);
  const candidats = chercherCandidats(racine);
  const mots = frequencesDeMots(racine, fichiers);
  const provisoires = [...candidats].filter(([n, i]) => estReel(classer(n, i, gardes))).map(([n]) => n);
  const comptes = compterOccurrences(racine, fichiers, provisoires);
  const classes = classerTout(candidats, gardes, mots, comptes);
  const reels = [...classes].filter(([, c]) => estReel(c)).map(([n]) => n);
  const courtes = new Map();
  for (const n of reels.filter((x) => classes.get(x).classe === 'reel')) {
    const c = formeCourte(n);
    const deja = classes.get(c)?.classe;
    if (c && (deja === undefined || deja === 'mot-courant' || deja === 'reel-doute')) courtes.set(c, n);
  }
  const surnoms = chercherSurnoms(racine, fichiers, reels, classes, comptes, mots);
  for (const [s, n] of surnoms) courtes.set(s, n);
  const nouveaux = [...new Set([...reels, ...courtes.keys()])].filter((n) => !comptes.has(n)).sort();
  for (const [n, v] of compterOccurrences(racine, fichiers, nouveaux)) comptes.set(n, v);
  const noms = [...new Set([...reels, ...courtes.keys()])].sort();
  return { classes, courtes, surnoms, comptes, mots, noms, fichiersTexte: fichiers.length, fichiersListe: fichiers, tableExistante };
}

function lignesPseudos({ classes, courtes, surnoms, comptes, mots }) {
  const out = ['pseudo\tclasse\traison\tfichiers\tlignes\tfichiers_code\tcontextes\tmot_courant\tforme_courte_de'];
  const ligne = (nom, classe, raison, contextes, courteDe) => {
    const c = comptes.get(nom);
    const mc = mots.get(nom.toLowerCase()) ?? 0;
    out.push([nom, classe, raison, c?.fichiers.size ?? '', c?.lignes.size ?? '', c?.code.size ?? '', contextes, mc >= SEUIL_MOT_COURANT ? mc : '', courteDe ?? ''].join('\t'));
  };
  for (const [nom, c] of [...classes].sort((a, b) => a[0].localeCompare(b[0]))) {
    if (c.classe === 'bruit') continue;
    ligne(nom, c.classe, c.raison, [...c.info.sources].join(','), null);
  }
  for (const [court, long] of [...courtes].sort()) ligne(court, surnoms.has(court) ? 'reel-surnom' : 'reel-forme-courte', surnoms.has(court) ? 'debut du pseudo d un joueur suivi' : 'sans suffixe numerique', '', long);
  return out;
}

/** La table : un `joueur_NNN` par pseudo. Un pseudo `reel` (preuve structuree) est remplace partout ;
 *  une forme courte ou un indice de prose en document est `ambigu` (contextes reconnus seulement) ;
 *  un indice vu seulement dans du code ou des tests est ECRIT EN COMMENTAIRE (a decommenter apres
 *  relecture : ce sont presque toujours des noms inventes par un test). */
function lignesTable({ classes, courtes, surnoms, comptes, mots, noms, tableExistante }) {
  const { connus, max } = numerotation(tableExistante);
  let n = max;
  const out = ['# table de correspondance (HORS depot) : type<TAB>original<TAB>remplacement<TAB>drapeaux'];
  for (const e of tableExistante) {
    out.push([e.type, e.original, e.remplacement, [...e.drapeaux].join(',')].join('\t').replace(/\t$/, ''));
  }
  // Numerotation dans l'ordre d'une empreinte : le numero ne revele ni l'initiale ni la frequence.
  const empreinte = (s) => createHash('sha256').update(s).digest('hex');
  const ordre = [...noms].sort((a, b) => empreinte(a).localeCompare(empreinte(b)));
  for (const nom of ordre) {
    if (connus.has(nom)) continue;
    const c = classes.get(nom);
    const court = courtes.has(nom);
    const codeSeul = c?.classe === 'reel-doute' && c.raison === 'code ou test seul';
    const drapeaux = [ressembleAUnMot(nom, mots) ? 'variantes_ctx' : 'variantes'];
    // Forme courte d'un pseudo tres present (joueur suivi) : remplacee partout, variantes de casse comprises.
    const courteSure = court && (comptes.get(courtes.get(nom))?.lignes.size ?? 0) >= SEUIL_JOUEUR_SUIVI;
    if (surnoms.has(nom)) drapeaux[0] = 'variantes_ctx';
    else if (courteSure) drapeaux[0] = 'variantes';
    else if (court || c?.classe === 'reel-doute') drapeaux.push('ambigu');
    n += 1;
    const ligne = ['pseudo', nom, `joueur_${String(n).padStart(3, '0')}`, drapeaux.join(',')].join('\t');
    out.push(codeSeul ? `# doute (code ou test seul) : ${ligne}` : ligne);
  }
  return out;
}

/** Les xuids des gardes (`classe<TAB>2533...`) : Map xuid -> classe ; ils ne sont jamais remplaces. */
function gardesXuids(gardes) {
  return new Map([...gardes.noms].filter(([nom]) => /^25[0-9]{14}$/.test(nom)));
}

/** Inventaire des xuids : ecrit xuids.tsv et rend les lignes de table des xuids a remplacer. */
function ecrireXuids(racine, v, fichiers, gardes) {
  const registre = lireRegistre(path.resolve(racine, v.registre));
  const par = inventorierXuids(racine, fichiers, registre, gardesXuids(gardes));
  const rang = [...par].sort((a, b) => b[1].occurrences - a[1].occurrences);
  const tsv = ['xuid\tclasse\tfichiers\toccurrences', ...rang.map(([x, e]) => [x, e.classe, e.fichiers.size, e.occurrences].join('\t'))];
  fs.writeFileSync(path.join(v.sortie, 'xuids.tsv'), tsv.join('\n') + '\n');
  return lignesTableXuids(par);
}

export function executer(argv) {
  const { values: v } = parseArgs({
    args: argv,
    options: {
      racine: { type: 'string', default: '.' },
      sortie: { type: 'string' },
      gardes: { type: 'string' },
      table: { type: 'string' },
      xuids: { type: 'boolean', default: false },
      registre: { type: 'string', default: 'apps/go-api/cmd/anonymise-gamertags/testdata/IDENTITES_APPS.tsv' },
    },
  });
  if (!v.sortie) throw new Error('--sortie <dossier> est obligatoire');
  const racine = path.resolve(v.racine);
  if (estSous(racine, v.sortie)) throw new Error('la sortie cite de vrais pseudos : elle doit etre hors du depot');
  const gardes = v.gardes ? lireGardesFichier(v.gardes) : lireGardes('');
  const tableExistante = v.table && fs.existsSync(v.table) ? lireTableFichier(v.table) : [];
  const bilan = inventorier({ racine, gardes, tableExistante });
  fs.mkdirSync(v.sortie, { recursive: true });
  fs.writeFileSync(path.join(v.sortie, 'pseudos.tsv'), lignesPseudos(bilan).join('\n') + '\n');
  const bruits = [...bilan.classes].filter(([, c]) => c.classe === 'bruit').map(([n, c]) => `${n}\t${c.raison}`);
  fs.writeFileSync(path.join(v.sortie, 'bruit.tsv'), ['jeton\traison', ...bruits.sort()].join('\n') + '\n');
  const lignes = lignesTable(bilan);
  if (v.xuids) lignes.push(...ecrireXuids(racine, v, bilan.fichiersListe, gardes));
  fs.writeFileSync(path.join(v.sortie, 'table.tsv'), lignes.join('\n') + '\n');
  const par = {};
  for (const [, c] of bilan.classes) par[c.classe] = (par[c.classe] ?? 0) + 1;
  console.log(`${bilan.fichiersTexte} fichiers texte ; candidats par classe :`, par, `; formes courtes : ${bilan.courtes.size}`);
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
