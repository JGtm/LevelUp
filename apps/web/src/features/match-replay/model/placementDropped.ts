/**
 * placementDropped.ts — CE QU'UN JOUEUR LAISSE AU SOL EN MOURANT, et le peu qui mérite d'être vu.
 *
 * LA RÈGLE R2/W4 DISAIT « RIEN », ET ELLE AVAIT RAISON POUR L'ESSENTIEL. Près de neuf poses
 * sur dix du corpus sont des LÂCHERS À LA MORT (`origin: dropped`, mesure du 2026-08-18,
 * 11 films, 3 661 poses à poseur mesuré) : les dessiner toutes noierait le mur et le capteur
 * sous des centaines de points qui ne disent rien du terrain. C'est pour cela que le calque
 * ne montre que les objets DÉPLOYÉS.
 *
 * L'UTILISATEUR A AMENDÉ CETTE RÈGLE LE 2026-08-18, et l'amendement est étroit :
 * « hors Fiesta c'est tactiquement interessant (power-ups, armes de puissance, equipements
 * laches a la mort) ». Un surbouclier au sol change ce que vaut le prochain échange ; une
 * grenade à fragmentation au sol, non. Ce fichier porte la LISTE de ce qui vaut le coup, et
 * rien d'autre.
 *
 * LA RESTRICTION « HORS FIESTA » A ÉTÉ LEVÉE LE 2026-08-20 : elle masquait 26 lâchers bien
 * réels sur le témoin Fiesta `000d5950` (11 murs, 15 capteurs), et l'utilisateur veut les
 * voir. Le réglage « Objets lâchés au sol » gouverne seul, dans tous les modes ; plus aucune
 * garde de mode ne le croise.
 *
 * LES ARMES LÂCHÉES N'Y SONT PAS, ET CE N'EST PAS UN OUBLI. `equipmentPlacements` est un canal
 * d'ÉQUIPEMENT (tag `eqip`) ; `weaponPads` est un canal de SOCLES (position où une arme
 * REPARAÎT), pas de lâchers. Aucun champ du document ne dit « telle arme est au sol en (x, y)
 * depuis telle image ». Les armes de puissance lâchées demandent donc une publication de
 * données supplémentaire, côté serveur — c'est un report écrit au registre, pas un manque ici.
 *
 * UN LÂCHER N'EST PAS L'OBJET ACTIF, et la mesure le prouve : sur le témoin `000d5950`, les
 * 11 poses `wall/dropped` portent l'identifiant de l'APPAREIL (`0x8e2dc574`), jamais celui des
 * PANNEAUX (`0x528fce46`) que le déploiement produit. Un mur au sol est l'appareil que son
 * porteur n'a pas eu le temps d'utiliser : il n'a ni arc, ni orientation, ni portée. D'où une
 * forme unique et discrète pour tous les lâchers (cf. `drawDroppedObject`), jamais la forme
 * de la famille.
 */
import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import { PAD_EQUIPMENT_FAMILIES } from './weaponPadFamilies'

/**
 * L'origine « lâché à la mort ». Identifiant STABLE du document (schéma 10), jamais un libellé.
 *
 * Le vocabulaire publié est `deployed` / `dropped` / `unknown`. `unknown` NE COMPTE PAS comme
 * un lâcher : ce sont les poses sans poseur mesuré et les artefacts antérieurs au schéma 10,
 * encore sur disque et en production. Les traiter en lâchers ferait apparaître, sur tout ce
 * parc, des objets dont rien n'établit qu'ils sont tombés.
 */
export const PLACEMENT_ORIGIN_DROPPED = 'dropped'

/**
 * LES ÉQUIPEMENTS DÉPLOYABLES du manifeste — ceux dont le lâcher se dessine.
 *
 * Ce sont exactement les familles auxquelles `PLACEMENT_RENDER` donne une forme d'objet ACTIF,
 * `other` excepté : si un objet mérite d'être vu quand il est posé, il mérite d'être vu quand
 * il traîne au sol, ramassable. `other` en est écarté parce que sa bascule est un outil de
 * diagnostic (« un objet est ici, sa nature n'est pas établie ») : promouvoir en objet de
 * puissance ce qu'on ne sait pas nommer affirmerait un enjeu que rien n'établit — même
 * raisonnement que le défaut `classic` des socles.
 *
 * LA LISTE EST ÉCRITE ET NON DÉDUITE de `PLACEMENT_RENDER`, pour une raison de dépendances :
 * ce module est lu PAR le calque, le déduire créerait un cycle. Un garde-rail rejoue donc la
 * correspondance dans les deux sens (`placementDropped.guard.test.ts`) — c'est le patron
 * « helper + garde-rail » de CLAUDE.md n°6, pas une seconde vérité.
 */
export const DROPPED_EQUIPMENT_FAMILIES: readonly string[] = [
  'wall',
  'sensor',
  'translocator_beacon',
  // L ECRAN OCCULTANT entre ici le 2026-08-27, en meme temps qu il gagne sa forme posee. Le
  // garde-rail l exige et il a raison : une famille qui se dessine DEPLOYEE doit se dessiner
  // LACHEE, sinon elle disparait de la carte a la mort de son porteur.
  'shroud_screen',
  'threat_seeker',
  'repair_field',
]

/**
 * TOUTES LES FAMILLES DE PUISSANCE dont le lâcher se dessine : les équipements déployables,
 * plus les POWER-UPS.
 *
 * LES POWER-UPS VIENNENT DE `PAD_EQUIPMENT_FAMILIES`, ils ne sont PAS recopiés. C'est la liste
 * EXPLICITE que l'utilisateur a demandée le 2026-08-18 pour les socles (« liste EXPLICITE »),
 * et elle vit déjà à deux endroits de `weaponPadFamilies.ts` (la table des vignettes et
 * `POWER_PAD_KEYS`). Une troisième copie serait la copie de trop (CLAUDE.md n°6) — et la
 * première à diverger le jour où le titre ajoutera un power-up.
 *
 * NE SONT PAS LÀ, ET C'EST LA MOITIÉ DE LA RÈGLE : les quatre grenades (59 à 63 % des poses
 * du corpus) et les trois capacités (`grapple`, `thruster`, `repulsor`). Les unes et les
 * autres sont à `null` dans `PLACEMENT_RENDER` — la décision de ne pas les dessiner est déjà
 * prise, mesurée et écrite ; ce lot ne la rouvre pas.
 */
export const PLACEMENT_DROPPED_FAMILIES: readonly string[] = [
  ...DROPPED_EQUIPMENT_FAMILIES,
  ...Object.keys(PAD_EQUIPMENT_FAMILIES),
]

/**
 * placementIsDroppedPower — cette pose est-elle un objet de PUISSANCE tombé au sol ?
 *
 * DEUX CONDITIONS, ET AUCUNE N'EST DÉDUITE DE L'AUTRE : l'origine mesurée est un lâcher, et
 * la famille figure dans la liste ci-dessus. Une pose d'origine inconnue n'en est jamais une
 * (cf. `PLACEMENT_ORIGIN_DROPPED`), et une grenade lâchée non plus, quelle que soit sa netteté.
 *
 * L'IDENTIFIANT N'EST PAS TESTÉ, contrairement au mur déployé (`WALL_PANEL_IDS`). C'est
 * délibéré et mesuré : le déploiement d'un mur publie DEUX poses (l'appareil qui vole et ses
 * panneaux), d'où le filtre côté déployé ; un lâcher n'en publie qu'une, celle de l'appareil.
 * Il n'y a rien à dédoublonner.
 */
export function placementIsDroppedPower(p: ReplayEquipmentPlacement): boolean {
  if (p.origin !== PLACEMENT_ORIGIN_DROPPED) return false
  return PLACEMENT_DROPPED_FAMILIES.includes(p.family)
}
