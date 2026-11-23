// table.mjs - lecture et validation de la table de correspondance (HORS depot).
//
// Format : un fichier texte, une ligne par entree, colonnes separees par une tabulation ;
// lignes vides et lignes commencant par `#` ignorees.
//
//   type <TAB> original <TAB> remplacement [<TAB> drapeaux]
//
// Types :
//   pseudo  un pseudo de joueur. Remplace par MOT ENTIER, sensible a la casse.
//   chemin  un chemin du poste (`C:\Users\<nom>`). Toutes les graphies de separateur et de lecteur
//           sont reconnues, sans tenir compte de la casse ; se termine sur une limite de mot.
//   profil  le NOM d'un profil seul, derriere `Users/` ou `home/` (`Alice` dans `C:\Users\Alice\x`) :
//           le reste du chemin reste intact et valide, ce qui convient au code et aux tests, ou le
//           chemin doit rester un chemin absolu.
//   texte   une chaine litterale quelconque (nom de machine, adresse). Sensible a la casse.
//
// Drapeaux (separes par des virgules) :
//   ambigu       le pseudo est aussi un mot courant : remplace seulement dans un contexte reconnu
//                (voir contextes.mjs) ; les autres occurrences sont ecartees et rapportees.
//   variantes    pseudo ou texte : reconnait aussi la forme en minuscules, en majuscules et, si le
//                nom porte des espaces, les graphies d'URL (`_`, `-`, `%20`, `+`) ; chaque variante
//                recoit le remplacement dans la meme casse.
//   variantes_ctx  comme `variantes`, mais les formes en minuscules, en majuscules et d'URL ne sont
//                remplacees qu'en contexte reconnu (le nom ressemble a un mot courant : sa forme
//                minuscule serait un mot de la langue) ; la forme exacte est remplacee partout.
//   docs, code   l'entree ne s'applique qu'aux documents (md, txt, tsv, html, journaux, dossiers .ai
//                et docs) ou qu'au code (le reste : go, ts, sh, lua, toml, yaml...) ; sans l'un de
//                ces drapeaux, aux deux. Permet `<poste>` dans les documents et un nom de profil
//                neutre dans le code.
//   mot          texte : exige des limites de mot (par defaut : sous-chaine).
//
// La table contient de VRAIS pseudos : elle ne doit jamais etre versionnee (l'outil refuse
// `--appliquer` si elle est sous la racine du depot).

import fs from 'node:fs';

export const TYPES = ['pseudo', 'chemin', 'profil', 'texte'];
export const DRAPEAUX = ['ambigu', 'variantes', 'variantes_ctx', 'mot', 'docs', 'code'];

/** Lit et valide un texte de table. Rend la liste des entrees ; leve une Error nommant la ligne. */
export function lireTableTexte(texte) {
  const entrees = [];
  const lignes = texte.split(/\r?\n/);
  lignes.forEach((l, i) => {
    if (l.trim() === '' || l.startsWith('#')) return;
    const c = l.split('\t');
    const no = i + 1;
    if (c.length < 3 || c.length > 4) {
      throw new Error(`table ligne ${no} : attendu type<TAB>original<TAB>remplacement[<TAB>drapeaux]`);
    }
    const [type, original, remplacement] = c;
    if (!TYPES.includes(type)) throw new Error(`table ligne ${no} : type inconnu "${type}"`);
    const drapeaux = new Set((c[3] ?? '').split(',').map((s) => s.trim()).filter(Boolean));
    for (const d of drapeaux) {
      if (!DRAPEAUX.includes(d)) throw new Error(`table ligne ${no} : drapeau inconnu "${d}"`);
    }
    if (original.length < 3) throw new Error(`table ligne ${no} : original de moins de 3 caracteres`);
    if (remplacement === '') throw new Error(`table ligne ${no} : remplacement vide`);
    entrees.push({ type, original, remplacement, drapeaux, ligne: no });
  });
  verifierCoherence(entrees);
  return entrees;
}

export function lireTableFichier(chemin) {
  return lireTableTexte(fs.readFileSync(chemin, 'utf8'));
}

/** Refuse les doublons ; l'idempotence (un remplacement qui redevient une cible) se juge dans
 *  `verifierIdempotence` de remplacer.mjs, qui connait le moteur. */
function verifierCoherence(entrees) {
  const vus = new Map();
  for (const e of entrees) {
    const cle = `${e.type}\u0000${e.type === 'chemin' ? e.original.toLowerCase() : e.original}`;
    if (vus.has(cle)) {
      throw new Error(`table ligne ${e.ligne} : original deja defini ligne ${vus.get(cle)}`);
    }
    vus.set(cle, e.ligne);
  }
}
