// classes.mjs - le classement des pseudos candidats.
//
//   proprietaire  le proprietaire du depot (public)                              GARDER
//   auteur        l'auteur d'un projet public credite                            GARDER
//   bot           un nom de bot fourni par le jeu                                GARDER
//   fictif        un pseudo deja fictif (fixtures anonymisees)                   GARDER
//   invente       un nom invente par un test ou une maquette                     GARDER
//   bruit         pas un pseudo (mot-cle, colonne, identifiant de code)          IGNORER
//   reel          un joueur reel : preuve structuree dans un document            REMPLACER
//   reel-prose    jeton de forme gamertag, seulement dans des documents, jamais dans le code REMPLACER
//   reel-doute    indice faible (prose, ou code seul) : classe reel, relecture   REMPLACER
//
// En cas de doute, la classe est reel-doute (jamais invente). Les listes de noms connus (proprietaire,
// auteur, bot, fictif, invente) viennent d'un fichier `--gardes` : `classe<TAB>nom` ou
// `classe<TAB>~expression`. Le fichier, comme la table, se range hors du depot.

import fs from 'node:fs';

export const CLASSES_GARDEES = new Set(['proprietaire', 'auteur', 'bot', 'fictif', 'invente']);

/** Lit un fichier de gardes. Rend { noms: Map nom -> classe, motifs: [{re, classe}] }. */
export function lireGardes(texte) {
  const noms = new Map();
  const motifs = [];
  texte.split(/\r?\n/).forEach((l, i) => {
    if (l.trim() === '' || l.startsWith('#')) return;
    const [classe, valeur] = l.split('\t');
    if (!CLASSES_GARDEES.has(classe) && classe !== 'bruit') throw new Error(`gardes ligne ${i + 1} : classe inconnue "${classe}"`);
    if (!valeur) throw new Error(`gardes ligne ${i + 1} : valeur manquante`);
    if (valeur.startsWith('~')) motifs.push({ re: new RegExp(valeur.slice(1)), classe });
    else noms.set(valeur, classe);
  });
  return { noms, motifs };
}

export function lireGardesFichier(chemin) {
  return lireGardes(fs.readFileSync(chemin, 'utf8'));
}

const MOTS_DE_CODE = new Set([
  'VARCHAR', 'FROM', 'NULL', 'NOT', 'LIKE', 'ILIKE', 'ORDER', 'ASC', 'DESC', 'TEXT', 'UTF', 'DOIT', 'TOUS',
  'SEUL', 'SANS', 'JAMAIS', 'REQUIS', 'COMPLET', 'CODE', 'VIDE', 'NON', 'EST', 'TEL', 'DEMO', 'Xbox', 'Halo',
  'with', 'and', 'the', 'who', 'whom', 'qui', 'est', 'par', 'has', 'ait', 'you', 'read', 'get', 'off', 'pas',
  'count', 'total', 'Total', 'Rang', 'Score', 'Kills', 'Morts', 'Ratio', 'Arme', 'Perf', 'Matchs', 'Matches',
  'unknown', 'Unknown', 'inconnu', 'Inconnu', 'main', 'Main', 'null', 'nil', 'string', 'name', 'Nom', 'field',
]);

function estBruit(nom) {
  return (
    /[{}$<>:()[\]=;,/\\]/.test(nom) ||
    nom.includes('...') ||
    /^[a-z0-9]+(?:_[a-z0-9]+)+$/.test(nom) ||
    /\.(?:ts|tsx|go|json|duckdb|md|toml|yaml|tsv|sh)$/.test(nom) ||
    /^[0-9a-f]{8}$/.test(nom) ||
    /^\d+$/.test(nom) ||
    MOTS_DE_CODE.has(nom)
  );
}

/** Classe un candidat : rend { classe, raison }. */
export function classer(nom, info, gardes) {
  const explicite = gardes.noms.get(nom) ?? gardes.motifs.find((m) => m.re.test(nom))?.classe;
  if (explicite) return { classe: explicite, raison: 'gardes' };
  if (estBruit(nom)) return { classe: 'bruit', raison: 'forme' };
  if (info.docStructure) return { classe: 'reel', raison: 'structure dans un document' };
  if (formeSeule(info)) {
    return estTechnique(nom) ? { classe: 'bruit', raison: 'jeton technique (largeur de bits, outil)' } : { classe: 'reel-prose', raison: 'forme de gamertag dans un document, absente du code' };
  }
  if (info.sources.size === 1 && info.sources.has('forme')) return { classe: 'bruit', raison: 'forme seule, vue dans le code : identifiant' };
  if (info.genres.has('code') && /[A-Z].*[A-Z]/.test(nom)) return { classe: 'bruit', raison: 'identifiant de code (majuscules internes, vu dans le code)' };
  if (!plausible(nom)) return { classe: 'bruit', raison: 'prose : pas un nom isole' };
  if (info.genres.has('docs')) return { classe: 'reel-doute', raison: 'prose dans un document' };
  return { classe: 'reel-doute', raison: 'code ou test seul' };
}

/** Le seul indice est la forme du jeton, et il ne se voit que dans des documents. */
function formeSeule(info) {
  return info.sources.size === 1 && info.sources.has('forme') && info.genres.size === 1 && info.genres.has('docs');
}

/** Un jeton de forme gamertag qui est en fait technique : largeur de bits, code d'erreur, outil. */
function estTechnique(nom) {
  return /[A-Za-z](?:8|16|32|64|128|256|512)$/.test(nom) || /^(?:AADSTS|ADR|CONCAT|REPLAY|UTF|SHA|UINT|INT|FLOAT|BSWAP|HIRC|CVE|GHSA|DEFAULT|FFFF|MPP|PLR|SIM|HTTP|TYPE|TOP|TEAM|TUG|TRL|STATETI|FIELD|MISSINGRUNTIME|QUESTION|QUANT|FORWARDVECTOR|COMBATYIELD|ASSISTPCT|METATECHNICAL|REFUSNI|REAPPARITION|FILM|REV|BIT|WORLD|FIXE|AXISW|TESTVALUE)/i.test(nom) || /^[A-F0-9]+$/.test(nom);
}

/** Un indice faible n'est retenu que pour un mot capitalise isole (un nom propre possible). */
export function plausible(nom) {
  return /^[A-Z][A-Za-z0-9_]{2,}$/.test(nom);
}

/** Pour un pseudo reel a suffixe numerique, la forme courte (sans les chiffres), ambigue par nature. */
export function formeCourte(nom) {
  const m = /^(.{4,}?)\d{2,}$/.exec(nom);
  return m ? m[1].trim() : null;
}

const SEUIL_COURANT = 100;
const SEUIL_LIBELLE = 20;

/** Un nom qui n'est qu'un pseudo reel suivi d'un mot (`<pseudo> Slayer`) n'est pas un pseudo a part. */
export function deriveDUnAutre(nom, reels) {
  const i = nom.indexOf(' ');
  return i >= 4 && reels.has(nom.slice(0, i));
}

/** Affine un `reel` avec la frequence des mots en minuscules (`mots`) : un libelle d'interface ou un
 *  nom de colonne n'est pas un pseudo. Un mot courant n'est ecarte que s'il se repand dans le depot
 *  (`largeDiffusion`) : un pseudo tout en minuscules a une frequence elevee par ses seules occurrences. Rend la classe d'origine ou { classe, raison }. */
export function affiner(nom, c, mots, largeDiffusion = true, lignesExact = 0) {
  if (c.classe !== 'reel' && c.classe !== 'reel-doute') return c;
  const freq = (m) => mots.get(m.toLowerCase()) ?? 0;
  const bruit = (raison) => ({ ...c, classe: 'bruit', raison });
  if (/^[-\s]+$/.test(nom) || /^\d/.test(nom) && !/[A-Za-z]{3,}/.test(nom.slice(2))) return bruit('forme');
  if (/[À-ÿ°%+]/.test(nom)) return bruit('libelle');
  if (/[_.':()/-]/.test(nom) || /^0x/i.test(nom)) return bruit('caractere interdit dans un gamertag');
  const mots_ = nom.split(/\s+/).filter((m) => m.length >= 3);
  if (nom.includes(' ') && mots_.length > 0 && mots_.every((m) => freq(m) >= SEUIL_LIBELLE)) return bruit('libelle');
  if (/^[A-Z][a-z]+$/.test(nom) && freq(nom) >= SEUIL_LIBELLE / 4 && freq(nom) > 2 * lignesExact) {
    return { ...c, classe: 'mot-courant', raison: 'sa forme minuscule est un mot courant' };
  }
  if (/^[A-Z]{4,}$/.test(nom) && freq(nom) >= SEUIL_LIBELLE / 4) {
    return { ...c, classe: 'mot-courant', raison: 'mot en capitales qui est aussi un mot courant' };
  }
  if (!nom.includes(' ') && freq(nom) >= SEUIL_COURANT && largeDiffusion) return { ...c, classe: 'mot-courant', raison: 'mot tres courant' };
  return c;
}
