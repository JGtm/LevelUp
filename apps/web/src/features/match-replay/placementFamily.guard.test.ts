/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : LES FAMILLES DE POSE SONT ÉNUMÉRÉES EN QUATRE ENDROITS.
 *
 * Une famille d'objet d'équipement existe à quatre endroits, et il n'y a pas de compilateur
 * entre eux :
 *   1. le VALIDEUR du serveur — `equipmentFamilies` (loader_replay_labels_equipment.go),
 *      liste FERMÉE des familles qu'un manifeste de titre a le droit de nommer ;
 *   2. la TABLE du titre      — `[[equipment_objects]]` de replay_labels.toml (`family = …`) ;
 *   3. le RENDU du client     — `PLACEMENT_RENDER` (equipmentPlacementsLayer.ts) ;
 *   4. les LIBELLÉS du client — `placementFamily` (i18n.ts), FR et EN.
 *
 * CE QUI CASSE SANS CE TEST, ET SILENCIEUSEMENT. Une famille ajoutée au manifeste et acceptée
 * par le valideur, mais absente de la table de rendu, ne dessine RIEN : la pose est publiée,
 * décodée, transmise, et l'écran reste vide sans qu'aucune erreur ne soit levée. Une règle de
 * rendu sans libellé rendrait l'infobulle muette. Les deux dérives sont invisibles à
 * l'exécution — c'est exactement ce qu'un garde-rail doit attraper. Modèle : fxInk.guard.test.ts.
 *
 * LES DEUX NIVEAUX NE SE CONFONDENT PAS, et c'est le point délicat de ce fichier :
 * `PLACEMENT_RENDER` est indexée par FAMILLE et rend une RÈGLE DE RENDU (`PlacementKind`) —
 * deux familles peuvent partager un tracé — tandis que `placementFamily` (i18n) est indexée
 * par RÈGLE. Le test relie donc famille → règle → libellé, jamais famille → libellé.
 *
 * LES ABSENCES SONT DES DÉCISIONS, et elles sont NOMMÉES ici. `null` dans la table = famille
 * connue qu'on ne dessine pas (objets portés). Absente de la table = décision assumée, et la
 * seule aujourd'hui porte sur les power-ups de carte (registre des reports, 2026-08-18 : un
 * seul membre mesuré chacun, tous deux `dropped`). Un ajout au valideur qui ne serait ni
 * dessiné ni listé ici fait ROUGIR ce test — c'est voulu.
 *
 * CETTE TABLE NE DIT RIEN DES LÂCHERS depuis le 2026-08-19 : un power-up LÂCHÉ se dessine, et
 * il reste pourtant hors table. La règle est ORTHOGONALE (une ORIGINE croisée avec une liste
 * de familles) et son garde-rail est `placementDropped.guard.test.ts` — ne pas « corriger »
 * l'absence des power-ups ci-dessous en les ajoutant à `PLACEMENT_RENDER`.
 */
import { describe, expect, it } from 'vitest'

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { PLACEMENT_RENDER, type PlacementKind } from './equipmentPlacementsLayer'
import { REPLAY_TEXT } from './i18n'

const REPO = resolve(__dirname, '..', '..', '..', '..', '..')
const GO_LOADER = resolve(
  REPO,
  'apps/go-api/internal/games/mappings/loader_replay_labels_equipment.go',
)
const TOML = resolve(REPO, 'config/titles/halo_infinite/mappings/replay_labels.toml')

/** La famille par défaut du serveur : admise et dessinée, jamais nommée. */
const DEFAULT_FAMILY = 'other'

/** La règle de rendu du défaut : dessinée, sans libellé de famille. */
const DEFAULT_KIND = 'unnamed'

/**
 * Les familles que le valideur admet MAIS que le rendu laisse dehors, par décision écrite.
 * Toute autre absence est un oubli, et ce test la signale.
 */
const HORS_TABLE_ASSUME = ['powerup_camo', 'powerup_overshield']

/** Les familles admises par le VALIDEUR Go (la liste fermée d'`equipmentFamilies`). */
function goFamilies(): string[] {
  const src = readFileSync(GO_LOADER, 'utf8')
  const start = src.indexOf('var equipmentFamilies = map[string]bool{')
  expect(start, 'equipmentFamilies introuvable dans le loader Go').toBeGreaterThan(-1)
  const body = src.slice(start, src.indexOf('\n}', start))
  const noms = [...body.matchAll(/"(\w+)":\s*true/g)].map((m) => m[1])
  // `equipFamilyOther` est servi par une CONSTANTE Go, pas par un littéral : la regex ne peut
  // pas l'attraper, et l'oublier ferait croire que le défaut n'est pas admis.
  if (body.includes('equipFamilyOther:')) noms.push(DEFAULT_FAMILY)
  return [...new Set(noms)].sort()
}

/**
 * Les familles que la table du titre emploie réellement (`family = "…"`).
 *
 * LE BLOC EST BORNÉ PAR LA TABLE SUIVANTE, et ce n'est pas une précaution théorique : le
 * manifeste porte depuis le 2026-08-18 une table `[[objective_objects]]` qui emploie AUSSI une
 * clé `family` — d'un autre vocabulaire (le drapeau de CTF, archétype `ti=42`), validé par une
 * autre liste fermée. Sans cette borne, la lecture allait jusqu'à la fin du fichier et
 * réclamait au valideur des poses une famille qui ne lui appartient pas.
 */
function tomlFamilies(): string[] {
  const src = readFileSync(TOML, 'utf8')
  const start = src.indexOf('[[equipment_objects]]')
  expect(start, '[[equipment_objects]] introuvable dans le manifeste du titre').toBeGreaterThan(-1)
  const block = blocDeTable(src.slice(start), '[[equipment_objects]]')
  return [...new Set([...block.matchAll(/^family\s*=\s*"(\w+)"/gm)].map((m) => m[1]))].sort()
}

/** Rend le préfixe de `src` qui n'appartient qu'à la table `entete` (répétée ou non). */
function blocDeTable(src: string, entete: string): string {
  const lignes = src.split('\n')
  const fin = lignes.findIndex((l) => l.trimStart().startsWith('[[') && l.trim() !== entete)
  return fin < 0 ? src : lignes.slice(0, fin).join('\n')
}

/** Les règles de rendu réellement employées par la table, sans le défaut. */
function kindsNommes(): string[] {
  const kinds = Object.values(PLACEMENT_RENDER).filter(
    (k): k is Exclude<PlacementKind, typeof DEFAULT_KIND> => k !== null && k !== DEFAULT_KIND,
  )
  return [...new Set<string>(kinds)].sort()
}

describe('garde-rail : le vocabulaire des familles de pose', () => {
  it('toute famille de la table de RENDU est admise par le valideur Go', () => {
    for (const f of Object.keys(PLACEMENT_RENDER)) {
      expect(goFamilies(), `${f} dessine, mais le valideur Go la refuserait`).toContain(f)
    }
  })

  it('toute famille admise par le Go est dessinée, ou hors table PAR DÉCISION ÉCRITE', () => {
    const dessinees = Object.keys(PLACEMENT_RENDER)
    for (const f of goFamilies()) {
      if (dessinees.includes(f)) continue
      expect(HORS_TABLE_ASSUME, `${f} est admise par le Go et ne dessine rien`).toContain(f)
    }
  })

  it('chaque famille que le titre emploie est admise ET connue du rendu', () => {
    for (const f of tomlFamilies()) {
      expect(goFamilies(), `${f} n'est pas admise par le valideur Go`).toContain(f)
      if (HORS_TABLE_ASSUME.includes(f)) continue
      expect(Object.keys(PLACEMENT_RENDER), `${f} ne dessine rien côté client`).toContain(f)
    }
  })

  it('chaque RÈGLE DE RENDU nommée a son libellé en FR et en EN', () => {
    for (const locale of ['fr', 'en'] as const) {
      const labels = REPLAY_TEXT[locale].placementFamily as Record<string, string>
      expect(Object.keys(labels).sort(), `libellés ${locale} désynchronisés`).toEqual(kindsNommes())
      for (const k of kindsNommes()) expect(labels[k], `${k} sans libellé ${locale}`).toBeTruthy()
    }
  })

  it('la famille par défaut est admise et dessinée, mais jamais nommée', () => {
    expect(goFamilies()).toContain(DEFAULT_FAMILY)
    expect(PLACEMENT_RENDER[DEFAULT_FAMILY]).toBe(DEFAULT_KIND)
    expect(Object.keys(REPLAY_TEXT.fr.placementFamily)).not.toContain(DEFAULT_KIND)
  })
})

/**
 * Garde-rail de TAILLE — le cliquet du registre des reports (2026-08-16 : « prochaine addition
 * sur l'un d'eux : extraire d'abord »). Le canvas du rejeu porte une dette de taille gelée ;
 * le lot des poses l'avait fait passer de 861 à 942 lignes sans extraction préalable, et c'est
 * ce que la revue a relevé. Le plafond n'est pas un idéal, c'est un CLIQUET : il ne remonte
 * jamais. Le franchir se corrige en extrayant, pas en relevant le nombre.
 *
 * 861 -> 858 le 2026-08-18 (lot des SOCLES D'ARME) : le calque, son survol et son infobulle
 * pesaient une soixantaine de lignes, et le canvas était PILE à son plafond. Les huit encres
 * qui y recopiaient huit fois le même corps sont donc parties dans `useReplayInks` AVANT
 * l'ajout — c'est exactement la manœuvre que ce cliquet existe pour imposer, et le nombre
 * descend d'autant.
 *
 * 858 -> 812 le 2026-08-18 (lot R3, item R3.1) : la croix de mort change de taille, d'encre et
 * de duree, et la duree devait ecrire POURQUOI. Plutot que d'allonger le canvas d'un
 * paragraphe, les reglages temporels et leur conversion sont partis dans `useReplayTiming` —
 * quatrieme extraction imposee par ce cliquet, et la plus grosse.
 *
 * 812 -> 808 le 2026-08-19 (lot des objets LACHES) : le canvas etait PILE a son plafond et le
 * lot y ajoutait une bascule, un comptage, un argument de survol et une prop de garde de mode.
 * Les trois morceaux des poses — comptes, axe de temps, survol — sont donc partis dans
 * `useReplayPlacements` AVANT l'ajout, cinquieme extraction imposee par ce cliquet.
 *
 * 808 -> 797 le 2026-08-24 (epure UI + « ! » du tireur) : le canvas etait PILE a son plafond
 * et le lot y ajoutait le calque fireMark. L'icone du bouton de reglages (ajoutee le meme
 * jour) est donc partie dans `SlidersIcon.tsx` AVANT l'ajout, sixieme extraction imposee par
 * ce cliquet ; la suppression du paragraphe de note et du compteur de vies fait le reste.
 *
 * 797 -> 791 le 2026-08-24 (barre de lecture) : la barre (icones lecture/recommencer,
 * vitesse, son, timeline) part dans `ReplayTransport.tsx` — septieme extraction — pendant
 * que le lot ajoute l'option « couleurs distinctes par joueur ».
 *
 * 791 -> 777 le 2026-08-24 (reglages dans la barre) : le bouton du tiroir rejoint la barre
 * de lecture, et la barre du haut du cadre — qui ne portait plus que lui — disparait avec
 * ses imports.
 *
 * 777 -> 742 le 2026-08-25 (fin de rejeu sur l'etat final) : le canvas n'avait plus que deux
 * lignes de marge et le lot y ajoutait la politique de fin de film. La LECTURE — etat
 * lu/pause, boucle rAF, curseur de la frise, arret sur la derniere image — part dans
 * `useReplayPlayback.ts`, huitieme extraction imposee par ce cliquet. Le canvas garde le
 * DESSIN, le hook porte le TEMPS.
 *
 * 742 -> 706 le 2026-08-26 (capture d'image et enregistrement video) : le canvas etait PILE a
 * son plafond, a la ligne pres, et le lot y branche deux commandes de sortie. Le CADRAGE —
 * fond retenu ou ecarte, bornes de scene, largeur de dessin, amplitude verticale, projection
 * partagee et trame d'altitudes — part dans `useReplayView.ts`, neuvieme extraction imposee
 * par ce cliquet. Les noms sortent inchanges, donc pas une ligne du dessin ne bouge.
 *
 * 706 -> 695 le 2026-08-27 (crane d'Oddball libre, schema 21) : le lot branche un calque de
 * plus — l'objet d'objectif la ou il est quand personne ne le porte. DIXIEME extraction imposee
 * par ce cliquet, en deux morceaux : le CABLAGE du calque (encre neutre + decision de peindre)
 * part dans `useReplayObjectiveObjects.ts`, patron de `useReplayFlagCarries` ; les trois
 * REGLAGES CONSTANTS (tokens de serie, reference vide de zones, cadence de publication) partent
 * dans `replayCanvasConfig.ts`. Aucun des deux ne deplace une ligne de logique.
 *
 * LE CABLAGE DE LA CAPTURE TIENT EN QUATRE LIGNES, et il n'en prendra pas une de plus : le
 * hook rend UN objet (`ReplayCapture`, patron de `ReplaySound`) que le canvas repasse tel quel
 * a la barre. L'enregistrement video puis le son de la video se branchent en ETENDANT l'appel
 * existant, pas en l'allongeant — c'est ce qui permet a ce plafond de tenir sur les trois lots.
 *
 * 706 -> 697 le 2026-08-26 (cadrage de la lecture sur le match reel) : le canvas etait PILE a
 * son plafond, a la ligne pres, et le lot y fait entrer la FENETRE DE GAMEPLAY (bornes de la
 * lecture, de la frise et de l'horloge). L'HORLOGE AFFICHEE et la publication bridee de l'image
 * courante partent donc dans `useReplayClock.ts`, dixieme extraction imposee par ce cliquet :
 * elles disent OU ON EN EST, pas ce qu'il faut peindre. Le canvas ne garde qu'un `clockTick`
 * en fin de trace.
 *
 * 697 -> 691 le 2026-08-27 (lot C, sons de fin de partie) : le canvas etait PILE a son plafond
 * et le lot y fait entrer la CONCLUSION SONORE (une prop de plus, relayee au lecteur et a la
 * lecture). Les cinq memos d'EFFETS PRECALCULES — tirs, « ! » du tireur, morts, fins de vol,
 * grappin — partent donc dans `useReplayFx.ts`, onzieme extraction imposee par ce cliquet :
 * ils ne dependent que du FILM, ne lisent ni theme ni cadrage, et ne dessinent rien. Les noms
 * sortent inchanges, comme a la neuvieme.
 *
 * 691 -> 691 le 2026-08-27 (lot VIP COURONNE) : le canvas etait PILE a son plafond et le lot y
 * fait entrer la COURONNE VIP (hook `useReplayVipCrown`, un paint et une prop de tiroir). Le
 * memo `objectivePulses` REJOINT donc `useReplayFx.ts` — il est de la meme nature que les cinq
 * autres (une liste d'evenements en monde, precalculee une fois, qui ne dessine rien) : douzieme
 * extraction imposee par ce cliquet, nom inchange, aucune ligne de tracé deplacee.
 *
 * 691 -> 690 le 2026-08-28 (correctif CI) : le decompte du lot VIP etait faux d'UNE ligne —
 * le fichier a atteint origin a 692 et ce test etait le seul rouge de la branche. L'appel
 * `useReplayVipCrown` est resserre sur une ligne (patron des appels longs voisins du tiroir),
 * le fichier retombe a 690 et le cliquet SUIT le fichier, comme a chaque fois : lui laisser
 * la ligne d'ecart aurait offert une ligne gratuite a la prochaine addition.
 *
 * 690 -> 672 le 2026-08-28 (lot LECTEUR, planche 2a — fusionne avec le correctif ci-dessus,
 * mene en parallele sur la meme base rouge a 692). La refonte de la barre de lecture ajoutait
 * au canvas une prop (`feedEntries`) et un hook. DEUX extractions, donc :
 *   - TREIZIEME, `useReplayTimeline.ts` : la frise et son clavier — echelle des pistes,
 *     reduction du fil aligne, dominance, medias, horloges de l'axe, raccourcis. Ils decrivent
 *     LA FRISE, le canvas garde le DESSIN.
 *   - QUATORZIEME, `useReplayDrawer.ts` : le montage du tiroir de reglages, une cinquantaine de
 *     lignes qui ne decidaient rien — elles RECOPIAIENT trente bascules de `useReplaySettings`
 *     vers le panneau. Le canvas garde desormais l'objet de reglages entier et n'en destructure
 *     que les VALEURS, celles que le trace lit.
 * Au merge, l'appel VipCrown resserre du correctif s'ajoute aux deux extractions : le fichier
 * tombe a 672 et le plafond descend a la mesure du jour, comme a chaque extraction depuis 861.
 *
 * 672 -> 679 le 2026-08-28 (lot PORTEUR ODDBALL, schema 23) : un CALQUE FRERE de plus, le porteur
 * du crane. Sa LOGIQUE est deja extraite — tout vit dans `useReplaySkullCarrier.ts`, quinzieme
 * hook de la meme famille que `useReplayVipCrown` et `useReplayFlagCarries`. Ne reste au canvas que
 * le cablage IRREDUCTIBLE d'un calque : un import, l'appel du hook, une ligne de peinture, une
 * dependance de `draw`, une entree de disponibilite du tiroir. C'est le SEUL cas ou le plafond
 * remonte, et il est justifie : l'extraction a bien eu lieu (le hook), et il n'y avait pas, dans ce
 * lot, d'extraction coincidente pour absorber les cinq lignes de glue — les inventer aurait ete du
 * gonflage inverse. La prochaine addition retombe sous la regle : logique dans un hook, cablage au
 * canvas, et le plafond suit le fichier vers le BAS a la premiere extraction.
 *
 * 679 -> 679 le 2026-08-28 (lot EXPORT HORS TEMPS REEL, etape E3) : le fichier etait PILE a son
 * plafond et le lot devait y brancher une commande de plus (le verdict du match, qui permet de
 * repeindre l'ecran de fin DANS la toile — un encodeur video ne voit pas le DOM). Le plafond ne
 * bouge pas d'une ligne, et c'est le point : l'addition a ete PAYEE, pas absorbee.
 *   - EXTRACTION, `hoverLayers.ts` : la distribution du geste de survol aux trois calques
 *     survolables. La decoupe etait deja ECRITE dans le canvas (« chacun rejoue le survol sur SA
 *     donnee, le canvas passe le geste ») — il recopiait le meme geste trois fois, deux fois de
 *     suite. Neuf lignes. Fonction pure et non hook : mémoïser un tableau reconstruit a chaque
 *     rendu aurait demande une echappatoire de lint pour un gain nul.
 *   - PAS DE SECOND HOOK AU CANVAS : l'export se branche DANS `useReplayCapture`, qui est deja
 *     « ce qui sort du rejeu » (image, video) — l'export en est la troisieme sortie. Le canvas
 *     ne gagne donc qu'une prop, une ligne d'options et un import.
 * La prochaine addition se paie de la meme facon.
 *
 * 679 -> 678 le 2026-08-30 (lot des ARMES AU SOL, schema 27) : un CALQUE FRERE de plus, les
 * armes abandonnees au sol. Sa logique est deja extraite — lecture temporelle dans
 * `groundWeaponTime.ts`, trace dans `groundWeaponsLayer.ts`, cablage dans
 * `useReplayGroundWeapons.ts` — mais le fichier etait PILE a son plafond et les six lignes de
 * glue devaient etre PAYEES.
 *   - SEIZIEME EXTRACTION, `useReplayDrawer.ts` : l'ETAT D'OUVERTURE du tiroir. Le canvas
 *     gardait trois choses qui ne parlent que du tiroir — ouvert ou ferme, le bouton qui
 *     l'ouvre, et la fermeture qui lui rend le focus — alors que le hook du tiroir existe
 *     depuis la quatorzieme. Il rend desormais `{ open, toggle, buttonRef, panel }` ; le canvas
 *     n'en garde que l'usage, et pas une ligne de logique ne se deplace.
 *   - DIX-SEPTIEME EXTRACTION, `useReplayGrenadeRest.ts` : la FIN DE VOL des grenades. Elle
 *     PAIE une addition — la DEFLAGRATION d'Assaut (`useReplayBombBlast`, 2026-08-31), qui
 *     demandait ses lignes de glue alors que le fichier etait PILE a son plafond. Seul l'appel
 *     de trace et son objet d'options se deplacent ; la mesure reste dans `useReplayFx` et
 *     `useReplayTiming`.
 *
 * Le plafond SUIT le fichier vers le bas, comme a chaque extraction depuis 861.
 *
 * 665 -> 665 le 2026-09-05 (lot VEHICULES, arrive du chantier `feat/v75-vehicules-sons`, ses
 * schemas 29-31 fondus dans le 39) : un CALQUE FRERE de plus (sprites teintes, cap dans le sens
 * du deplacement), PLUS une donnee transverse que nul calque precedent ne portait — le PREDICAT
 * EMBARQUE (C7) que `drawTracksLayer` consomme pour supprimer le pion d'un occupant. Toute la
 * logique est dans `vehiclesLayer.ts` (pur) et `useReplayVehicles.ts` (le hook, dix-huitieme de
 * la famille) ; le canvas ne gagne que le cablage IRREDUCTIBLE : un import, l'appel du hook, une
 * ligne de peinture, une dependance de `draw`, une entree de disponibilite du tiroir, et le
 * branchement d'`embarkedAtSlot` sur le style du calque des pions. Le meme lot apporte la SOURCE
 * D'IDENTITE `colorOfXuid` (le pont slot->joueur est MUET pendant un episode d'occupation, le
 * bipede ne repliquant plus, alors que le document nomme son occupant) : tout est dans
 * `rosterLogic.ts` / `useSlotIdentity.ts` / `vehiclesLayer.ts`, le canvas ne gagne QUE deux
 * jetons sur des lignes existantes. Le plafond NE BOUGE PAS D'UNE LIGNE.
 */
describe('garde-rail : la taille du canvas du rejeu ne remonte pas', () => {
  it('ReplayCanvas.tsx reste sous son plafond', () => {
    const src = readFileSync(resolve(__dirname, 'ReplayCanvas.tsx'), 'utf8')
    expect(src.split('\n').length - 1).toBeLessThanOrEqual(665)
  })
})
