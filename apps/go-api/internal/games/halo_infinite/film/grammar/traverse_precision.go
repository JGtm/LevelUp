package grammar

// traverse_precision.go — LES DESCRIPTEURS DE QUANTIFICATION DE POSITION.
//
// Sorti de `traverse.go` par deplacement pur au lot 2.7 (scission des fichiers de plus de
// 500 lignes) : aucune ligne de logique n'a change. Ce fichier porte les deux descripteurs
// que le deser d'i0 consulte — le chemin DELTA du bipede (`Lecteur.traversal`) et le chemin
// ABSOLU des objets du monde (`WorldObjectPrecision`) — et le seul point d'installation des
// largeurs de la carte.

// traversal est le descripteur de quantification de position employé quand le
// déser d'i0 (object-position-dynamic-precision-component) se déclenche. Les tables de
// l'exécutable lisent 0 statiquement : les vraies largeurs viennent de la config de
// réplication du film.
//
// LE 6/6/6 HISTORIQUE ÉTAIT FAUX, ET C'ÉTAIT LA FAUTE CENTRALE DU DÉCODEUR (2026-07-26).
// Il donne un i0 de 5 + 6+6+6 = 23 bits. La vérité terrain — 134 767 relevés du curseur
// réel sur le désérialiseur du jeu — donne **47 bits, à 100 %**. Déficit de 24 bits dès le
// PREMIER composant de 100 % des records : tout ce qui suit était lu à côté, ce qui explique
// les comptes de grenades à 255 sur un champ borné à 2.
//
//	47 = 5 (3 spine + 1 useDefault + 1 index de région) + 13 + 13 + 14 + 2 de queue
//
// CE CHAMP N'EST PAS LE LEVIER — piège vérifié le 2026-07-26, à ne pas refaire.
// `AxisW` est la largeur des DELTAS de position (petits écarts image à image : 6 bits
// suffisent). Les 13/13/14 sont les largeurs des positions ABSOLUES, et le déser en tient
// une seconde, distincte, via `absAxisW`/`AbsoluteAxisW`. Y écrire 13/13/14 allonge le
// chemin delta — qui est le chemin DOMINANT — et dégrade au lieu de corriger : mesuré,
// i22 passe de 90,02 % à 92,83 % de comptes impossibles. Le 6/6/6 est donc conservé ici.
//
// Le vrai correctif d'i0 doit distinguer les deux largeurs le long de chaque branche de
// FUN_1406cfe44 (keep-baseline 101 bits · absolu · delta prédit), et non régler une globale.
// `calibratedSkip` court-circuite déjà le problème en sautant 47 ou 101 bits selon
// le premier bit — mais c'est un banc de calibration propre à Cliffhanger, pas un décodeur.
//
// C'ÉTAIT UNE VARIABLE DE PAQUET (`TraversalPrecision`) JUSQU'AU LOT 2.2.a : elle vient
// désormais du PROFIL que le lecteur porte ([Lecteur.poserMouvement]), et son défaut est
// l'invariant de [mouvementDuProfil]. Le seul écrivain de production — la calibration de
// `killsource` — le passe maintenant par `FrameConfig.Mouvement`.
func (b *Lecteur) traversal() PrecisionDescriptor { return b.p.Mouvement.Traversal }

// WorldObjectPrecision est le descripteur du chemin WORLD-OBJECT d'i0
// (`object-position-component`) : projectiles ti=41, armes au sol ti=42, équipement ti=37,
// corps rigides ti=38. À la différence du bipède, ces archétypes n'envoient PAS de delta de
// position : ils envoient la position ABSOLUE quantifiée à chaque image. Les largeurs sont
// donc celles de la CARTE — les mêmes que l'absolu du bipède — et non un 6/6/6 de delta.
//
// 45 bits au total = 1 precHigh + 1 index-sel + 1 index de région + 13 + 13 + 14 + 2 de queue.
// Mesuré sur 000d5950 : le triplet de porte vaut (0,0,0) dans 6 922 des 6 924 records des
// 70 trajectoires de grenade.
//
// PROPRE À LA CARTE, et CE DÉFAUT EST L'ENTRÉE `cliffhanger` DU CATALOGUE : `{13,13,14}` est
// exactement `map_quant_bounds.json` -> `cliffhanger`.`axisWidths` (module `ridgeline`), ce que
// `map_bounds_test.go` vérifie. Le défaut n'est donc pas un repli neutre : c'est UNE carte.
//
// QUI L'INSTALLE, ET DEPUIS QUAND (2026-08-15). `replay.BuildFromFilm` installe les largeurs de
// la carte du match sur le profil de balayage du contexte, pour toute la durée du décodage
// (`replay.installWorldObjectPrecision`). Elles viennent de `MapQuantEntry.AxisWidths`
// — la MÊME entrée de catalogue qui fournit les bornes, jamais un second réglage à armer à part.
//
// AVANT cette date, AUCUN chemin de production ne l'écrasait : toutes les cartes autres que
// Cliffhanger déquantifiaient leurs objets du monde aux largeurs de Cliffhanger. Mesuré sur
// 7 films / 7 cartes, dont 6 autres que Cliffhanger (part d'échantillons de projectile dans l'emprise du nuage des bipèdes du
// même film, coordonnées normalisées de l'AABB) : 0,09 · 0,51 · 28,46 · 31,31 · 65,21 % avec le
// défaut, 98,96 à 99,79 % avec les largeurs du catalogue — et 92,11 % des DEUX côtés sur
// Cliffhanger, où le correctif ne change rien par construction.
//
// `DetectI0Layout` n'est PAS la source de ces largeurs : c'est le CONTRÔLE que réclame le
// commentaire d'`AxisWidths`. Accord catalogue <-> découpage lu dans le film : 7 films sur 7.
//
// C'ÉTAIT UNE VARIABLE DE PAQUET JUSQU'AU LOT 2.2.b, puis l'héritage de processus jusqu'au lot
// 2.3 : elle vient du PROFIL que le lecteur porte (`Movement.WorldObject`), et l'installateur
// de `replay` la pose sur le PROFIL DE BALAYAGE DU CONTEXTE du film — le canal qui atteint les
// quarante balayages de la cuisson du rejeu, chacun construisant ses lecteurs par
// [FilmContext.NouveauLecteur].
func (b *Lecteur) worldObjectPrecision() PrecisionDescriptor { return b.p.Mouvement.WorldObject }
