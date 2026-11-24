// candidats.mjs - l'inventaire des PSEUDOS candidats d'un depot, sans base de donnees.
//
// Un pseudo est une chaine quelconque : on les trouve par recoupement de contextes structures.
//   tsv         cellule sous une colonne gamertag / joueur / pseudo / player / victime...
//   md-table    meme chose dans un tableau Markdown
//   json        cles "gamertag", "player_name"... dans les JSON et fixtures texte
//   url         segment /players/<pseudo>/
//   mot-cite    `joueur "X"`, `GT : X`, `pseudo « X »`
//   mot-nu      `gamertag X`, `GT X` (nom capitalise)
//   fil         `X a elimine Y`, `X killed Y`
//   forme       un jeton de forme gamertag : Majuscule + lettres + 2 a 6 chiffres (`NomPropre1234`) ; la
//               forme des gamertags que Xbox genere, la plus frequente dans la prose
// Un candidat porte le nombre de fichiers, de lignes et les contextes ou il a ete vu ; le classement
// (proprietaire, fictif, reel...) est dans classes.mjs.

import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { decoder, genreDeFichier } from './fichiers.mjs';

const COLONNE = /^(?:gamertag|gt|joueur|pseudo|player|player_name|victime|killer|tueur|nom_joueur|gamertags?)$/i;
const EXTENSIONS_MEDIAS = /\.(webp|png|jpe?g|gif|ico|ogg|wav|woff2?|gz|bin|duckdb|wal|sig|zst|zip|blob|mvar)$/i;

const MOTIFS = [
  ['json', /"(?:gamertag|Gamertag|GamerTag|player_name|PlayerName|playerName)"\s*:\s*"([^"\\]{3,32})"/g],
  ['url', /\/players\/([^/\s"'`)?#]{3,32})(?=[/\s"'`)?#]|$)/g],
  ['mot-cite', /\b(?:joueur|gamertag|GT|pseudo|player)s?\s*[:=]?\s*[`"'«]\s*([^`"'»\n]{3,24}?)\s*[`"'»]/gi],
  ['mot-nu', /\b(?:gamertag|GT|pseudo)\s+(?:de\s+|d')?([A-Z][A-Za-z0-9_]{2,20})\b/g],
  ['forme', /(?<![A-Za-z0-9_./-])([A-Z][A-Za-z]{2,}[0-9]{2,6}|[A-Z]{3,}[0-9]{2,6})(?![A-Za-z0-9_./-])/g],
  ['fil', /\b([A-Za-z0-9_]{3,20}) (?:a éliminé|éliminé|killed|a tué|a battu|eliminated) ([A-Za-z0-9_]{3,20})\b/g],
];

/** Les cellules d'une ligne de tableau (TSV ou Markdown) et, si c'est un en-tete, les colonnes de pseudo. */
function cellules(ligne, tsv) {
  const brut = ligne.split(tsv ? '\t' : '|');
  return tsv ? brut.map((c) => c.trim()) : brut.slice(1, ligne.trim().endsWith('|') ? -1 : undefined).map((c) => c.trim());
}

/** Ajoute les cellules de pseudo d'une ligne de donnees de tableau. */
function donnees(cellulesNettoyees, etat, source, ajouter) {
  if (!etat.colonnes) return;
  for (const i of etat.colonnes) if (cellulesNettoyees[i]) ajouter(cellulesNettoyees[i], source);
}

/** Une ligne de tableau. En Markdown, une ligne est RETENUE jusqu'a la suivante : si c'est le
 *  separateur `|---|`, la ligne retenue etait un en-tete (jamais une donnee). */
function ligneDeTableau(ligne, tsv, etat, ajouter) {
  const source = tsv ? 'tsv' : 'md-table';
  const nettoyees = cellules(ligne, tsv).map((c) => c.replace(/[`*]/g, ''));
  const separateur = !tsv && /^[\s:|-]+$/.test(ligne);
  if (etat.attente && !separateur) donnees(etat.attente, etat, source, ajouter);
  if (separateur && etat.attente) etat.colonnes = null;
  etat.attente = null;
  if (separateur) return;
  if (nettoyees.some((c) => COLONNE.test(c))) {
    etat.colonnes = nettoyees.map((c, i) => (COLONNE.test(c) ? i : -1)).filter((i) => i >= 0);
  } else if (tsv) {
    donnees(nettoyees, etat, source, ajouter);
  } else {
    etat.attente = nettoyees;
  }
}

function parcourirLigne(ligne, tsv, etat, ajouter) {
  if (tsv || /^\s*\|/.test(ligne)) {
    ligneDeTableau(ligne, tsv, etat, ajouter);
  } else {
    if (etat.attente) donnees(etat.attente, etat, 'md-table', ajouter);
    etat.attente = null;
    etat.colonnes = null;
  }
  for (const [src, re] of MOTIFS) {
    re.lastIndex = 0;
    for (let m = re.exec(ligne); m; m = re.exec(ligne)) {
      ajouter(m[1], src);
      if (src === 'fil') ajouter(m[2], src);
    }
  }
}

/** Parcourt tous les fichiers texte suivis. Rend Map nom -> { sources:Set, fichiers:Set, lignes, genres:Set, docStructure }
 *  (docStructure : vu dans un contexte structure d'un DOCUMENT, la preuve la plus forte). */
export function chercherCandidats(racine) {
  const suivis = execFileSync('git', ['ls-files', '-z'], { cwd: racine, maxBuffer: 1 << 30 })
    .toString('utf8').split('\0').filter(Boolean);
  const candidats = new Map();
  for (const f of suivis) {
    if (EXTENSIONS_MEDIAS.test(f) || f.startsWith('scripts/anonymisation/')) continue;
    const d = decoder(fs.readFileSync(path.join(racine, f)));
    if (d.ignore) continue;
    const genre = genreDeFichier(f);
    const ajouter = (nom, src) => {
      const n = nom.trim();
      if (n.length < 3 || n.length > 32) return;
      const e = candidats.get(n) ?? { sources: new Set(), fichiers: new Set(), lignes: 0, genres: new Set(), docStructure: false };
      e.sources.add(src);
      if (genre === 'docs' && SOURCES_STRUCTUREES.has(src)) e.docStructure = true;
      e.fichiers.add(f);
      e.genres.add(genre);
      e.lignes++;
      candidats.set(n, e);
    };
    const etat = { colonnes: null, attente: null };
    const tsv = /\.tsv$/i.test(f);
    for (const ligne of d.texte.split('\n')) parcourirLigne(ligne, tsv, etat, ajouter);
  }
  return candidats;
}

export const SOURCES_STRUCTUREES = new Set(['tsv', 'md-table', 'json', 'url']);
