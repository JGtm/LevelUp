/**
 * replayAimCone.ts — LE CÔNE DE VISÉE d'un marqueur : son cap, et son ÉLÉVATION lue dans sa
 * LONGUEUR.
 *
 * EXTRAIT DE `replayMarkers.ts` LE 2026-08-18 (lot R2-V) : le calque des marqueurs avait
 * franchi le seuil de taille du dépôt (CLAUDE.md n°5) et le lot lui ajoutait le double contour
 * du joueur de la page. La découpe tombe sur une frontière nette — le cône ne dessine PAS le
 * joueur, il dessine ce qu'il REGARDE, et il est le seul bloc du fichier à lire deux champs
 * (`h` et `p`) plutôt qu'une position.
 */
import type { MarkerStyle } from "./replayMarkers";
import { freshness, heldReading, type XY } from "./replayLogic";
import type { ReplayTrackReady } from "./replayNormalize";

/** Longueur d'une visée À PLAT — la référence, pas le maximum : l'élévation joue autour d'elle. */
const AIM_LENGTH = 52;
const AIM_HALF_ANGLE = 0.42;
/**
 * LA LUNETTE (schéma 29) : le cône RESSERRE SON OUVERTURE, et ne change PAS de longueur.
 *
 * Les deux mécaniques sont volontairement orthogonales. L'ÉLÉVATION pilote la LONGUEUR (un
 * regard incliné porte moins loin sur le plan) ; la LUNETTE pilote l'ANGLE (un joueur épaulé
 * regarde plus étroit). Elles se lisent donc ensemble sans se confondre : un joueur à la
 * lunette qui vise vers le bas a un cône à la fois plus étroit et plus court.
 *
 * CETTE VALEUR EST UN CHOIX DE RENDU, PAS UNE MESURE, et il faut le dire : le film transmet le
 * PALIER de lunette, jamais le grossissement — celui-ci appartient à l'arme. 0,18 rad vaut
 * environ 2,3 fois moins que l'ouverture à la hanche, assez pour se lire d'un coup d'œil, et
 * assez large pour que le cône ne disparaisse pas sur un marqueur de petite taille.
 */
const AIM_SCOPED_HALF_ANGLE = 0.18;
/**
 * Un cône plus étroit couvre moins de pixels : à opacité égale il se verrait MOINS qu'un cône
 * large, alors qu'il dit quelque chose de plus rare et de plus intéressant. Le supplément
 * compense la surface perdue, il ne crie pas.
 */
const AIM_SCOPED_ALPHA_BOOST = 1.25;
const AIM_CONE_ALPHA = 0.55;
/** Une visée de 5 s ne vaut pas une visée de l'instant : elle perd 62 % de son opacité. */
const AIM_FADE = 0.62;
/**
 * L'AMPLITUDE DE L'ÉLÉVATION : ±55 % de la longueur à plat, atteints à la verticale.
 *
 * Le cône va donc de 23 px (visée pleine plongée) à 81 px (visée pleine montée) autour des
 * 52 px du regard horizontal — un rapport de 3,4 entre les deux extrêmes, assez pour que le
 * sens se lise SANS second repère, ce qui est tout l'objet du changement.
 *
 * Ce que la mesure dit du champ (lot E, 3 films) : médiane −4,7 / −3,4 / −3,6°, 67 à 77 % des
 * visées vers le BAS, extrêmes −85,5 à +82°. La carte ne s'allonge donc pas : les cônes
 * passent leur temps un cheveu SOUS la référence, et ce sont les instants qui comptent — un
 * tir en plongée, un joueur qui couvre une passerelle — qui s'écartent franchement.
 */
const AIM_PITCH_SWING = 0.55;
/** Au-delà, `sin` REDESCEND et inverserait la lecture : la longueur cesserait d'être monotone. */
const AIM_PITCH_CLAMP_DEG = 90;

/**
 * drawAimCone dessine la DIRECTION DU REGARD, décodée du même record que la position.
 *
 * Le cône se dégrade du centre vers le bord — dense à l'origine, où il faut lire QUI vise,
 * transparent au bout, où il ne faut pas masquer le décor. Il pâlit avec l'âge de la mesure et
 * n'est PAS dessiné au-delà du maintien : passé ce délai, on ne sait plus où le joueur regarde,
 * et une direction périmée affirmerait ce qu'on ignore.
 *
 * IL N'Y A PLUS D'AXE (décision D3, 2026-08-16) : les deux traits qui prolongeaient le point
 * — « le bâton » — ont été supprimés à la demande de l'utilisateur. Ce que le cône seul perd
 * en précision d'angle, la carte le regagne en lisibilité : huit bâtons sur un 4v4 se
 * croisaient au-dessus des noms.
 *
 * IL N'Y A PLUS DE TICK D'ÉLÉVATION NON PLUS (2026-08-29, même demande, même raison). Le trait
 * collé à la pointe du cône se lisait comme un défaut de tracé plutôt que comme une mesure, et
 * il n'existait que pour rattraper le COSINUS d'alors, qui étant pair confondait « vise le
 * ciel » et « vise ses pieds ». La longueur porte désormais le signe elle-même (`pitchScale`) :
 * le tick n'avait plus rien à dire, il a donc été supprimé — pas désactivé (CLAUDE.md n°7).
 *
 * SES DIMENSIONS SONT CELLES DE LA PLANCHE, « un peu plus prononcées » (verdict du soir du
 * 2026-08-16, §1bis du plan) : rayon 52 à plat, demi-ouverture 0,42 rad, alpha 0,55. Le cône
 * avait d'abord été rétréci à 30 px / 0,30 — trop timide pour se lire une fois les noms posés.
 *
 * DEPUIS LE SCHÉMA 13, IL DIT LES DEUX AXES DU REGARD. Le cap oriente le secteur ; l'ÉLÉVATION
 * (`p`, degrés, positif = vers le haut) en règle la LONGUEUR — court vers le bas, long vers le
 * haut. Un artefact antérieur au schéma 13 ne porte pas `p` : le cône y garde sa longueur de
 * référence, et c'est le comportement voulu — absent se lit « à plat », jamais « inconnu ».
 */
export function drawAimCone(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  c: XY,
  style: MarkerStyle,
  color: string,
): void {
  const read = heldReading(
    track.points,
    style.frame,
    (p) => p.h,
    style.timing.aimHold,
  );
  if (!read) return;
  const fresh = freshness(read.age, style.timing.aimHold, AIM_FADE);
  // Monde -> canevas : l'axe Y est inversé, donc l'angle l'est aussi.
  const ang = (-read.value * Math.PI) / 180;
  const R =
    AIM_LENGTH * pitchScale(heldPitch(track.points, style, read.age)) * style.k;
  // LA LUNETTE agit sur l'OUVERTURE, jamais sur la longueur : celle-ci est deja portee par
  // l'elevation, juste au-dessus. Les deux mecaniques restent ainsi lisibles ensemble.
  const scoped = heldScoped(track.points, style, read.age);
  const halfAngle = scoped ? AIM_SCOPED_HALF_ANGLE : AIM_HALF_ANGLE;
  const gradient = ctx.createRadialGradient(c.x, c.y, 0, c.x, c.y, R);
  gradient.addColorStop(0, color);
  gradient.addColorStop(1, "transparent");
  ctx.globalAlpha = Math.min(
    1,
    AIM_CONE_ALPHA * fresh * (scoped ? AIM_SCOPED_ALPHA_BOOST : 1),
  );
  ctx.beginPath();
  ctx.moveTo(c.x, c.y);
  ctx.arc(c.x, c.y, R, ang - halfAngle, ang + halfAngle);
  ctx.closePath();
  ctx.fillStyle = gradient;
  ctx.fill();
  ctx.globalAlpha = 1;
}

/**
 * heldPitch rend l'ÉLÉVATION en vigueur, ou 0 (à plat).
 *
 * LA RÈGLE D'ÂGE N'EST PAS UNE PRÉCAUTION DÉCORATIVE. `p` est omis quand la visée s'arrondit
 * à plat (contrat du champ, cf. `Point.P` côté Go), et `heldReading` remonterait alors
 * jusqu'à un point PLUS ANCIEN qui, lui, porte une élévation : le marqueur afficherait une
 * plongée périmée sur une visée à plat actuelle. Les deux angles venant du MÊME
 * enregistrement, une élévation trouvée plus loin dans le passé que le cap appartient
 * forcément à une autre visée — on la refuse, et « absent » redevient ce qu'il doit être.
 */
function heldPitch(
  points: ReplayTrackReady["points"],
  style: MarkerStyle,
  headingAge: number,
): number {
  const read = heldReading(
    points,
    style.frame,
    (p) => p.p,
    style.timing.aimHold,
  );
  return read && read.age <= headingAge ? read.value : 0;
}

/**
 * heldScoped dit si le joueur est A LA LUNETTE au moment lu, et applique LA MEME REGLE D'AGE
 * que l'elevation, pour la meme raison.
 *
 * `s` est omis quand le joueur n'est pas a la lunette (contrat du champ, cf. `Point.S` cote
 * Go) : `heldReading` remonterait donc jusqu'a un point PLUS ANCIEN qui, lui, porte un palier,
 * et le marqueur afficherait un epaulement perime sur une visee a la hanche actuelle. Le champ
 * etant pose par le MEME enregistrement que le cap, un palier trouve plus loin dans le passe
 * que le cap appartient forcement a un autre instant — on le refuse.
 *
 * ABSENT SE LIT « PAS A LA LUNETTE », JAMAIS « INCONNU » : c'est le contrat publie par le
 * document, et il rend correct par defaut le rendu des artefacts anterieurs au schema 29 —
 * ils dessinent l'ouverture large, ce qu'ils ont toujours fait.
 */
function heldScoped(
  points: ReplayTrackReady["points"],
  style: MarkerStyle,
  headingAge: number,
): boolean {
  const read = heldReading(
    points,
    style.frame,
    (p) => p.s,
    style.timing.aimHold,
  );
  return read !== null && read.age <= headingAge && read.value > 0;
}

/**
 * pitchScale : ce que l'élévation fait à la LONGUEUR du cône — `1 + 0,55 × sin(p)`.
 *
 * CE N'EST PLUS UNE PROJECTION, ET C'EST DÉLIBÉRÉ. La version précédente valait `cos(p)`, la
 * part horizontale d'un regard incliné : physiquement juste, mais PAIRE — plonger de 30° et
 * viser 30° en l'air rendaient exactement le même cône, si bien qu'un repère de sens (le tick)
 * devait être ajouté à côté pour lever une ambiguïté que la longueur créait elle-même.
 *
 * Le sinus renverse ce marché : il est IMPAIR et STRICTEMENT CROISSANT sur [−90, +90], donc la
 * longueur seule ordonne les visées — plus le joueur pique, plus le cône est court ; plus il
 * lève la tête, plus il est long. Un seul trait de plume porte l'information, et rien n'est
 * collé au cône pour la compléter.
 *
 * IL EST AUSSI LE PLUS SENSIBLE LÀ OÙ VIVENT LES MESURES : sa pente est maximale à plat, où
 * tombe la médiane du champ (−4,7 / −3,4 / −3,6° sur trois films), et il sature près de la
 * verticale, où le degré exact n'apprend plus rien.
 *
 * LE PLANCHER N'EST PLUS UNE CONSTANTE À PART : à −90° le facteur vaut 0,45, soit 23 px — le
 * marqueur reste lisible sans qu'on ait à border quoi que ce soit. Le BORNAGE, lui, est
 * nécessaire pour une autre raison : `sin` redescend au-delà de ±90°, et la formule publiée de
 * `p` couvre ±180° sous réserve explicite (cf. `AimPitchDeg` côté Go). Une valeur hors de la
 * moitié centrale du champ raccourcirait donc un cône qui monte — on l'écrête plutôt que de
 * laisser la lecture s'inverser.
 */
function pitchScale(pitchDeg: number): number {
  const bounded = Math.max(
    -AIM_PITCH_CLAMP_DEG,
    Math.min(AIM_PITCH_CLAMP_DEG, pitchDeg),
  );
  return 1 + AIM_PITCH_SWING * Math.sin((bounded * Math.PI) / 180);
}
