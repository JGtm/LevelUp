/**
 * buildSquadFragTools — « Outils de destruction » Escouade = version MULTI-JOUEURS de
 * buildFragDetailBreakdown (source UNIQUE du « Détails des frags »).
 *
 * Par joueur : armes gun (per-arme, depuis weapon_kills) + détail NON-arme (Assassinat /
 * Corps-à-corps / Coup au sol / Charge spartane / Grenade / engins) tiré de la
 * FragDistribution du joueur (frag_classes). Le résidu « Non attribué »/« Spartan » et les
 * classes sans outil identifiable (équipement, environnement, UGC) sont EXCLUS par
 * buildFragDetailBreakdown — NON_WEAPON_FRAG_CLASSES, vrai depuis DEC-1 (2026-08-29).
 *
 * Anti-double-comptage : les sentinels grenade/mêlée de weapon_kills (is_grenade_melee)
 * sont écartés du volet per-arme — leur compte fait DOUBLON avec la FragDistribution
 * (mêmes match_participants.{grenade,melee}_kills). Le détail autoritatif (dont le split
 * Mêlée → Assassinat/Corps-à-corps sur H5) vient de la distribution.
 *
 * Réalimente le chart existant squadWeaponKillsChart (barres groupées) : sortie de MÊME
 * shape que l'entrée (SquadWeaponKills). DEUX plafonds indépendants, l'un et l'autre sans
 * perte silencieuse : les armes gun aux `topGuns` plus gros totaux escouade (reste →
 * « Autres armes »), les lignes de détail non-arme aux `topDetails` plus gros (reste →
 * « Autres frags »). Ordre final ASC par total escouade (comme le chart), les deux agrégats
 * épinglés en bas.
 */
import { buildFragDetailBreakdown, GUN_CLASSES } from '@/components/charts/fragDetailBreakdown'
import type {
  FragClassEntry,
  FragDistribution,
  SquadWeaponKills,
  SynthesisWeaponKillEntry,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

/** Nombre d'armes gun individuelles affichées avant repli en « Autres armes ». */
export const SQUAD_TOOLS_TOP_GUNS = 8

/**
 * Nombre de lignes de DÉTAIL non-arme (mêlée, types de grenade, engins) affichées avant
 * repli en « Autres frags ». Le cap top-N ne portait que sur les armes : le détail, lui,
 * arrivait entier et le tri ASC le faisait remonter EN HAUT du graphe, noyant les 8 armes
 * sous une dizaine de micro-lignes (constat utilisateur du 2026-08-29).
 * 6 = le cas nominal tient sans repli (mêlée + les 5 types de grenade, ou le split H5
 * Assassinat/Corps-à-corps/Coup au sol/Charge + grenade + un engin).
 */
export const SQUAD_TOOLS_TOP_DETAILS = 6

export interface SquadFragToolsOpts {
  /** Libellé localisé d'un rôle (manifeste frags.role.<role>). */
  roleLabel: (role: string) => string
  /** Libellé localisé d'une classe (manifeste frags.class.<class>). */
  classLabel: (className: string) => string
  /** Locale d'affichage courante — choisit label/label_en pour les rôles OBJET (D2). */
  locale: Locale
  /** Libellé de la ligne agrégée « Autres armes » (i18n). */
  otherWeaponsLabel: string
  /** Libellé de la ligne agrégée « Autres frags » (i18n) — détail non-arme replié. */
  otherKillsLabel: string
  /** Cap d'armes gun individuelles ; les autres → « Autres armes ». */
  topGuns: number
  /** Cap de lignes de détail non-arme ; les autres → « Autres frags ». */
  topDetails: number
}

interface MergedBar {
  label: string
  class: string
  killsByPlayer: Record<string, number>
  total: number
}

/** Une ligne est un « tir » (arme gun) si sa classe est vide (arme hors registre) ou
 *  ∈ GUN_CLASSES. Sinon c'est un détail non-arme (mêlée/grenade/capacités). */
function isGunLine(cls: string): boolean {
  return cls === '' || GUN_CLASSES.has(cls)
}

/** « Détails des frags » d'UN joueur : armes gun (sans les sentinels grenade/mêlée,
 *  doublon distribution) + détail non-arme depuis sa FragDistribution. */
function buildPlayerDetail(
  bars: NonNullable<SquadWeaponKills['bars']>,
  player: string,
  fragClassesByPlayer: Record<string, FragClassEntry[]>,
  opts: SquadFragToolsOpts,
): SynthesisWeaponKillEntry[] {
  const weaponsP = bars
    .filter((b) => !b.is_grenade_melee)
    .map((b) => ({ label: b.label, class: b.class, kills: b.kills_by_player[player] ?? 0 }))
    .filter((w) => w.kills > 0)
  // buildFragDetailBreakdown ne lit que `.classes` — total_kills neutre.
  const distributionP: FragDistribution = { total_kills: 0, classes: fragClassesByPlayer[player] ?? [] }
  return buildFragDetailBreakdown(distributionP, weaponsP, {
    roleLabel: opts.roleLabel,
    classLabel: opts.classLabel,
    locale: opts.locale,
  })
}

/** Fusionne le détail d'un joueur dans l'accumulateur multi-joueurs (clé = label). */
function mergeDetail(merged: Map<string, MergedBar>, player: string, detail: SynthesisWeaponKillEntry[]): void {
  for (const line of detail) {
    let m = merged.get(line.label)
    if (!m) {
      m = { label: line.label, class: line.class ?? '', killsByPlayer: {}, total: 0 }
      merged.set(line.label, m)
    }
    if (!m.class && line.class) m.class = line.class // privilégie une classe non vide
    m.killsByPlayer[player] = (m.killsByPlayer[player] ?? 0) + line.kills
    m.total += line.kills
  }
}

/**
 * Replie les lignes au-delà d'un cap en UNE ligne agrégée (« Autres armes » côté gun,
 * « Autres frags » côté détail). La classe de l'agrégat est vide : c'est une ligne
 * synthétique sans classe canonique, et `class` n'est plus lue après cette étape.
 */
function aggregateOverflow(overflow: MergedBar[], label: string): MergedBar {
  const killsByPlayer: Record<string, number> = {}
  let total = 0
  for (const g of overflow) {
    for (const [pl, k] of Object.entries(g.killsByPlayer)) killsByPlayer[pl] = (killsByPlayer[pl] ?? 0) + k
    total += g.total
  }
  return { label, class: '', killsByPlayer, total }
}

export function buildSquadFragTools(
  weaponKills: SquadWeaponKills | null | undefined,
  fragClassesByPlayer: Record<string, FragClassEntry[]>,
  opts: SquadFragToolsOpts,
): SquadWeaponKills | null {
  const players = weaponKills?.players ?? []
  const bars = weaponKills?.bars ?? []
  if (!weaponKills || players.length === 0 || bars.length === 0) return null

  // weapon_id par label (armes gun) — carry pour la sortie (les lignes synthétiques,
  // rôles + « Autres armes », prennent 0 en repli ; le chart affiche le label).
  const weaponIdByLabel = new Map<string, number>()
  for (const b of bars) if (b.label && !weaponIdByLabel.has(b.label)) weaponIdByLabel.set(b.label, b.weapon_id)

  const merged = new Map<string, MergedBar>()
  for (const p of players) mergeDetail(merged, p, buildPlayerDetail(bars, p, fragClassesByPlayer, opts))

  // Deux familles, deux caps INDÉPENDANTS (kills desc avant la coupe des deux côtés) :
  // sans cap sur le détail, les micro-lignes non-arme remontaient en tête du tri ASC et
  // repoussaient les armes hors du champ visible.
  const all = [...merged.values()]
  const byTotalDesc = (a: MergedBar, b: MergedBar) =>
    a.total !== b.total ? b.total - a.total : a.label.localeCompare(b.label)
  const guns = all.filter((m) => isGunLine(m.class)).sort(byTotalDesc)
  const details = all.filter((m) => !isGunLine(m.class)).sort(byTotalDesc)
  const gunCap = Math.max(0, opts.topGuns)
  const detailCap = Math.max(0, opts.topDetails)

  const outRows: MergedBar[] = [...guns.slice(0, gunCap), ...details.slice(0, detailCap)]
  // Tri par usage : ASC par total escouade (tie-break label).
  outRows.sort((a, b) => (a.total !== b.total ? a.total - b.total : a.label.localeCompare(b.label)))

  // Agrégats épinglés TOUT EN BAS, HORS tri par usage (un agrégat ne peut pas être classé
  // « significatif »). Le chart squadWeaponKills n'a PAS d'inverse yAxis → la 1re catégorie
  // (index 0) est rendue EN BAS → on PRÉFIXE les agrégats, « Autres armes » en tout dernier.
  const detailOverflow = details.slice(detailCap)
  if (detailOverflow.length > 0) outRows.unshift(aggregateOverflow(detailOverflow, opts.otherKillsLabel))
  const gunOverflow = guns.slice(gunCap)
  if (gunOverflow.length > 0) outRows.unshift(aggregateOverflow(gunOverflow, opts.otherWeaponsLabel))

  return {
    players,
    bars: outRows.map((m) => ({
      weapon_id: weaponIdByLabel.get(m.label) ?? 0,
      label: m.label,
      class: m.class,
      kills_by_player: m.killsByPlayer,
      total_squad: m.total,
    })),
  }
}
