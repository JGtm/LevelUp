/**
 * usageI18n.ts — LE DICTIONNAIRE DES FORMES PARTAGÉES de l'usage : les états vides (D8), la légende
 * de la bande de régularité, et les noms des familles d'équipement (lus par le nommage des objets de
 * l'Emprise, `squad/emprise/objectName.ts`, et par la carte « Équipement pris » des Séries
 * temporelles et de Sessions).
 *
 * PARITÉ FR/EN PAR TYPAGE : `Record<Locale, UsageText>` — une clé ajoutée d'un côté casse la
 * compilation de l'autre. Il vit à part des manifestes TOML des pages : ces formes parlent le
 * vocabulaire du FILM (familles d'équipement), pas celui des KPI de session.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface UsageText {
  /**
   * LES ÉTATS VIDES D'UN BLOC DANS UNE RANGÉE (D8, 2026-09-21) : le bloc RESTE affiché et NOMME sa
   * cause — une rangée amputée d'une carte se lit comme un bug. Une description par cause…
   */
  unavailableLoadFailed: string
  emptyNoFilm: string
  emptyNoObjectives: string
  /**
   * …et un TITRE COURT par cause (2026-09-22) : l'état vide canonique de l'app (`EmptyStateNotice`)
   * se lit sur deux lignes, un titre en gras puis sa description en gris.
   */
  emptyTitleNoFilm: string
  emptyTitleNoObjectives: string
  emptyTitleLoadFailed: string
  /** LA LÉGENDE DE LA BANDE DE RÉGULARITÉ (2026-09-21) : les quatre encres, écrites UNE fois. */
  bandLegendAbove: string
  bandLegendNear: string
  bandLegendBelow: string
  bandLegendUnmeasured: string
  /** Noms des familles d'équipement du bilan (clés du résumé Go). */
  metricCamo: string
  metricOvershield: string
  metricWall: string
  equipSensor: string
  equipShroud: string
  equipSeeker: string
  equipField: string
  equipTranslocator: string
  /** Famille hors catalogue : la clé reste à l'écran — un nom approchant se lirait comme une certitude. */
  metricDeployedFmt: (family: string) => string
}

export const USAGE_TEXT: Record<Locale, UsageText> = {
  fr: {
    unavailableLoadFailed: "La lecture du résumé d'usage a échoué.",
    emptyNoFilm: 'Aucun film décodé sur cette sélection.',
    emptyNoObjectives: 'Aucun objectif dans les modes de cette sélection.',
    emptyTitleNoFilm: 'Aucune mesure',
    emptyTitleNoObjectives: "Aucune mesure d'objectif",
    emptyTitleLoadFailed: 'Lecture impossible',
    bandLegendAbove: 'Au-dessus de la parité',
    bandLegendNear: 'Au niveau de la parité',
    bandLegendBelow: 'Sous la parité',
    bandLegendUnmeasured: 'Non mesuré',
    metricCamo: 'Camouflage',
    metricOvershield: 'Surbouclier',
    metricWall: 'Mur de protection',
    equipSensor: 'Capteur de menaces',
    equipShroud: 'Écran occultant',
    equipSeeker: 'Traqueur de menaces',
    equipField: 'Champ de réparation',
    equipTranslocator: 'Translocateur',
    metricDeployedFmt: (fam) => `Équipement ${fam}`,
  },
  en: {
    unavailableLoadFailed: 'Loading the usage summary failed.',
    emptyNoFilm: 'No decoded film in this selection.',
    emptyNoObjectives: 'No objective in the modes of this selection.',
    emptyTitleNoFilm: 'Nothing measured',
    emptyTitleNoObjectives: 'No objective measured',
    emptyTitleLoadFailed: 'Could not load',
    bandLegendAbove: 'Above parity',
    bandLegendNear: 'At parity',
    bandLegendBelow: 'Below parity',
    bandLegendUnmeasured: 'Not measured',
    metricCamo: 'Camouflage',
    metricOvershield: 'Overshield',
    metricWall: 'Drop wall',
    equipSensor: 'Threat sensor',
    equipShroud: 'Shroud screen',
    equipSeeker: 'Threat seeker',
    equipField: 'Repair field',
    equipTranslocator: 'Translocator',
    metricDeployedFmt: (fam) => `Equipment ${fam}`,
  },
}

/**
 * equipmentFamilyLabel — le libellé d'une famille du BILAN D'ÉQUIPEMENT, la clé étant celle du
 * RÉSUMÉ Go (`replay.EquipmentOutcomeFamilies`, vocabulaire des POSES : `translocator_beacon`,
 * `shroud_screen`, `threat_seeker`, `repair_field`). Une famille NEUVE (manifeste étendu) garde sa
 * clé à l'écran — jamais un nom approchant.
 */
export function equipmentFamilyLabel(family: string, t: UsageText): string {
  switch (family) {
    case 'wall':
      return t.metricWall
    case 'sensor':
      return t.equipSensor
    case 'translocator_beacon':
      return t.equipTranslocator
    case 'shroud_screen':
      return t.equipShroud
    case 'threat_seeker':
      return t.equipSeeker
    case 'repair_field':
      return t.equipField
    case 'powerup_camo':
      return t.metricCamo
    case 'powerup_overshield':
      return t.metricOvershield
    default:
      return t.metricDeployedFmt(family)
  }
}
