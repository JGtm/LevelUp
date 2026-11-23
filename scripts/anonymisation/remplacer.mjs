// remplacer.mjs - le moteur de remplacement : table -> expressions -> texte remplace + rapport.
//
// Regles :
//   - pseudo : mot entier (ni lettre, ni chiffre, ni `_` de part et d'autre), sensible a la casse ;
//     un pseudo `ambigu` n'est remplace que dans un contexte reconnu (contextes.mjs) ;
//   - chemin : toutes les graphies de separateur et de lecteur, insensible a la casse ;
//   - profil : le nom d'un profil derriere `Users/` ou `home/`, le reste du chemin intact ;
//   - texte  : sous-chaine litterale (limites de mot avec le drapeau `mot`) ;
//   - fins de ligne, BOM et tout octet hors remplacement sont preserves ;
//   - idempotent : `verifierIdempotence` refuse une table dont un remplacement serait lui-meme une cible.

import { creerSuivi } from './contextes.mjs';

const LIMITE_G = '(?<![\\p{L}\\p{N}_])';
const LIMITE_D = '(?![\\p{L}\\p{N}_])';

function echapper(s) {
  return s.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&');
}

/** Les graphies d'un pseudo ou d'un texte : l'original, puis (drapeau `variantes`) casses et URL. */
function formes(entree) {
  const out = new Map([[entree.original, entree.remplacement]]);
  if (!entree.drapeaux.has('variantes') && !entree.drapeaux.has('variantes_ctx')) return out;
  const { original: o, remplacement: r } = entree;
  out.set(o.toLowerCase(), r.toLowerCase());
  out.set(o.toUpperCase(), r.toUpperCase());
  if (o.includes(' ')) {
    for (const sep of ['_', '-', '%20', '+']) {
      out.set(o.split(' ').join(sep), r);
      out.set(o.toLowerCase().split(' ').join(sep), r.toLowerCase());
    }
  }
  return out;
}

function expressionChemin(original) {
  let reste = original;
  let tete = '';
  const lecteur = /^(?:([A-Za-z]):|\/(?:mnt\/)?([A-Za-z])(?=[\\/]))/.exec(reste);
  const sep = '[\\\\/]{1,4}';
  if (lecteur) {
    const l = (lecteur[1] ?? lecteur[2]).toLowerCase();
    tete = `(?:(?:${l}:|/${l}|/mnt/${l})${sep})?`;
    reste = reste.slice(lecteur[0].length);
  } else if (/^[\\/]/.test(reste)) {
    tete = '[\\\\/]{1,4}';
  }
  const segments = reste.split(/[\\/]+/).filter(Boolean).map(echapper);
  return new RegExp(`${LIMITE_G}${tete}${segments.join(sep)}${LIMITE_D}`, 'giu');
}

/** Les entrees `texte` : deux expressions combinees (avec et sans limites de mot), une alternative par
 *  forme, les plus longues d'abord. Une table de milliers de xuids reste ainsi un seul passage. */
function compilerTextes(entrees) {
  const groupes = [false, true].map((mot) => {
    const table = new Map();
    for (const e of entrees.filter((x) => x.drapeaux.has('mot') === mot)) {
      for (const [forme, rempl] of formes(e)) if (!table.has(forme)) table.set(forme, rempl);
    }
    const alt = [...table.keys()].sort((p, q) => q.length - p.length).map(echapper);
    const re = alt.length > 0 ? new RegExp((mot ? LIMITE_G : '') + '(?:' + alt.join('|') + ')' + (mot ? LIMITE_D : ''), 'gu') : null;
    return { re, table };
  });
  return groupes.filter((g) => g.re);
}

function expressionProfil(nom) {
  return new RegExp(`(?<=(?:Users|home)[\\\\/]{1,4})${echapper(nom)}${LIMITE_D}`, 'giu');
}

/** Construit le moteur : { chemins:[{re,entree}], textes:[{re,table}], pseudos:{re,table} }. Les chemins (en
 *  premier) et les profils partagent la liste `chemins`. */
export function compiler(entrees) {
  const rang = (e) => (e.type === 'chemin' ? 0 : 1);
  const chemins = entrees
    .filter((e) => e.type === 'chemin' || e.type === 'profil')
    .sort((a, b) => rang(a) - rang(b) || b.original.length - a.original.length)
    .map((entree) => ({
      entree,
      re: entree.type === 'profil' ? expressionProfil(entree.original) : expressionChemin(entree.original),
    }));
  const textes = compilerTextes(entrees.filter((e) => e.type === 'texte'));
  const table = new Map();
  for (const e of entrees.filter((x) => x.type === 'pseudo')) {
    for (const [forme, rempl] of formes(e)) {
      const ctx = e.drapeaux.has('variantes_ctx') && forme !== e.original;
      if (!table.has(forme)) table.set(forme, { rempl, entree: e, ctx });
    }
  }
  const alternatives = [...table.keys()].sort((a, b) => b.length - a.length).map(echapper);
  const re = alternatives.length > 0 ? new RegExp(`${LIMITE_G}(?:${alternatives.join('|')})${LIMITE_D}`, 'gu') : null;
  return { chemins, textes, pseudos: { re, table } };
}

/** Remplace dans une ligne. `contexte(debut, fin)` dit si une occurrence ambigue est reconnue.
 *  `noter(type, label, statut, original)` recoit chaque occurrence. */
function remplacerLigne(ligne, moteur, contexte, noter) {
  let s = ligne;
  for (const { re, entree } of moteur.chemins) {
    s = s.replace(re, (m) => {
      noter('chemin', entree.remplacement, 'remplace', m);
      return entree.remplacement;
    });
  }
  for (const { re, table } of moteur.textes) {
    s = s.replace(re, (m) => {
      const rempl = table.get(m);
      noter('texte', rempl, 'remplace', m);
      return rempl;
    });
  }
  const { re, table } = moteur.pseudos;
  if (!re) return s;
  const base = s;
  let out = '';
  let dernier = 0;
  re.lastIndex = 0;
  for (let m = re.exec(base); m; m = re.exec(base)) {
    const { rempl, entree, ctx } = table.get(m[0]);
    const fin = m.index + m[0].length;
    const ok = !(entree.drapeaux.has('ambigu') || ctx) || contexte(base, m.index, fin);
    noter('pseudo', rempl, ok ? 'remplace' : 'ecarte', m[0]);
    if (ok) {
      out += base.slice(dernier, m.index) + rempl;
      dernier = fin;
    }
  }
  return out + base.slice(dernier);
}

/** Remplace dans un texte (un fichier decode). Rend { texte, occurrences:[{ligne,type,label,statut,extrait}] }.
 *  Les fins de ligne sont conservees telles quelles. */
export function remplacerTexte(texte, moteur, cheminFichier = '') {
  const suivi = creerSuivi(cheminFichier);
  const occurrences = [];
  const morceaux = texte.split(/(\r\n|\n|\r)/);
  for (let i = 0; i < morceaux.length; i += 2) {
    const ligne = morceaux[i];
    if (ligne === '') continue;
    suivi.ligne(ligne);
    const noter = (type, label, statut, original) => {
      occurrences.push({ ligne: i / 2 + 1, type, label, statut, original, extrait: ligne.trim().slice(0, 240) });
    };
    morceaux[i] = remplacerLigne(ligne, moteur, (l, d, f) => suivi.reconnu(l, d, f), noter);
  }
  return { texte: morceaux.join(''), occurrences };
}

/** Refuse une table dont un remplacement redeviendrait une cible (le second passage changerait
 *  quelque chose) ou dont deux pseudos partageraient un meme remplacement. */
export function verifierIdempotence(entrees, moteur) {
  const vus = new Map();
  for (const e of entrees) {
    if (e.type === 'pseudo') {
      const autre = vus.get(e.remplacement);
      if (autre !== undefined) throw new Error(`table ligne ${e.ligne} : meme remplacement que la ligne ${autre}`);
      vus.set(e.remplacement, e.ligne);
    }
    const sondes = new Set([e.remplacement, ...[...formes(e).values()]]);
    for (const s of sondes) {
      const { texte } = remplacerTexte(s, moteur);
      const force = remplacerTexte(` ${s} `, moteur).texte;
      if (texte !== s || force !== ` ${s} `) {
        throw new Error(`table ligne ${e.ligne} : le remplacement "${s}" est lui-meme remplace (non idempotent)`);
      }
    }
  }
}
