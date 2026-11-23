// contextes.mjs - les contextes ou un pseudo AMBIGU (aussi un mot courant) est reconnu comme pseudo.
//
// Un pseudo marque `ambigu` dans la table n'est remplace que si son occurrence se trouve dans l'un
// de ces contextes ; ailleurs elle est ecartee et rapportee pour relecture.
//
//   1. cle JSON de pseudo : "gamertag": "X", "player_name": "X"...
//   2. mot qui annonce un pseudo, juste avant : joueur X, gamertag : X, GT `X`, pseudo « X »
//   3. segment d'URL ou de chemin : /players/X/, /joueur/X
//   4. ligne de fil : « X a elimine Y », « Y killed X », « elimine par X »
//   5. cellule entiere d'un tableau (TSV ou Markdown) dont l'en-tete de colonne est une colonne de pseudo

const CLES_JSON = '(?:gamertag|Gamertag|GamerTag|gamer_tag|player_name|playerName|PlayerName|display_name|displayName|killer_gamertag|victim_gamertag)';
const AVANT_JSON = new RegExp(`"${CLES_JSON}"\\s*:\\s*"$`);
const AVANT_MOT = /(?:\b(?:joueur|joueuse|gamertag|gamertags|pseudo|pseudos|GT|player|players|coequipier|adversaire|equipier|coéquipier|escouade de)\s*(?:[:=]|de |d')?\s*[`"'«(]?)$/i;
const AVANT_URL = /\/(?:players|joueur|joueurs|player)\/$/;
const APRES_URL = /^(?:\/|\?|#|$)/;
const VERBES = '(?:a éliminé|a elimine|éliminé|elimine|killed|a tué|a tue|eliminated|a battu|beat)';
const APRES_FIL = new RegExp(`^["'\`»)]?\\s+${VERBES}(?![\\p{L}])`, 'iu');
const AVANT_FIL = new RegExp(`${VERBES}(?:\\s+par|\\s+by)?\\s+["'\`«(]?$`, 'iu');
const COLONNE_PSEUDO = /^(?:gamertag|gt|joueur|pseudo|player|player_name|victime|killer|tueur|nom_joueur|gamertags?)$/i;

function nettoieCellule(c) {
  return c.replace(/[`*_]/g, '').trim();
}

/** Un suivi par fichier : appeler `ligne(texte)` pour chaque ligne, puis `reconnu(...)`. */
export function creerSuivi(cheminFichier) {
  const tsv = /\.tsv$/i.test(cheminFichier);
  let colonnes = null;
  return {
    /** Met a jour l'etat de tableau avec la ligne courante (a appeler avant `reconnu`). */
    ligne(texte) {
      const estTableau = tsv || /^\s*\|/.test(texte);
      if (!estTableau) {
        colonnes = null;
        return;
      }
      const cellules = texte.split(tsv ? '\t' : '|');
      const idx = [];
      cellules.forEach((c, i) => {
        if (COLONNE_PSEUDO.test(nettoieCellule(c))) idx.push(i);
      });
      if (idx.length > 0) colonnes = idx;
    },
    /** Vrai si l'occurrence texte[debut, fin[ de la ligne est dans un contexte de pseudo. */
    reconnu(texte, debut, fin) {
      const avant = texte.slice(0, debut);
      const apres = texte.slice(fin);
      if (AVANT_JSON.test(avant)) return true;
      if (AVANT_MOT.test(avant)) return true;
      if (AVANT_URL.test(avant) && APRES_URL.test(apres)) return true;
      if (APRES_FIL.test(apres) || AVANT_FIL.test(avant)) return true;
      return celluleDePseudo(texte, debut, fin, tsv, colonnes);
    },
  };
}

function celluleDePseudo(texte, debut, fin, tsv, colonnes) {
  if (!colonnes) return false;
  const sep = tsv ? '\t' : '|';
  const gauche = texte.lastIndexOf(sep, debut - 1);
  const droite = texte.indexOf(sep, fin);
  const entre = (a, b) => texte.slice(a, b);
  const avantCellule = gauche < 0 ? entre(0, debut) : entre(gauche + 1, debut);
  const apresCellule = droite < 0 ? entre(fin, texte.length) : entre(fin, droite);
  if (nettoieCellule(avantCellule) !== '' || nettoieCellule(apresCellule) !== '') return false;
  const index = texte.slice(0, debut).split(sep).length - 1;
  return colonnes.includes(index);
}
