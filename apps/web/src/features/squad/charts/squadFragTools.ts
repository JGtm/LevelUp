/**
 * buildSquadFragTools — « Outils de destruction » Escouade = version MULTI-JOUEURS de
 * buildFragDetailBreakdown (source UNIQUE du « Détails des frags »).
 *
 * Par joueur : armes gun (per-arme, depuis weapon_kills) + détail NON-arme (Assassinat /
 * Corps-à-corps / Coup au sol / Charge spartane / Grenade) tiré de la FragDistribution du
 * joueur (frag_classes). Le résidu « Non attribué »/« Spartan » (classe unattributed) et
 * les buckets non-combat sont EXCLUS par buildFragDetailBreakdown.
 *
 * Anti-double-comptage : les sentinels grenade/mêlée de weapon_kills (is_grenade_melee)
 * sont écartés du volet per-arme — leur compte fait DOUBLON avec la FragDistribution
 * (mêmes match_participants.{grenade,melee}_kills). Le détail autoritatif (dont le split
 * Mêlée → Assassinat/Corps-à-corps sur H5) vient de la distribution.
 *
 * Réalimente le chart existant squadWeaponKillsChart (barres groupées) : sortie de MÊME
 * shape que l'entrée (SquadWeaponKills). Les armes gun sont plafonnées aux `topGuns` plus
 * gros totaux escouade ; le reste est agrégé en une ligne « Autres armes » (aucune perte
 * silencieuse). Ordre final ASC par total escouade (comme le chart), pour cohérence visuelle.
 */
import { buildFragDetailBreakdown, GUN_CLASSES } from '@/components/charts/fragDetailBreakdown'
import type {
  FragClassEntry,
  FragDistribution,
  SquadWeaponKills,
  SynthesisWeaponKillEntry,
} from '@/lib/api/types'

/** Nombre d'armes gun individuelles affichées avant repli en « Autres armes ». */
export const SQUAD_TOOLS_TOP_GUNS = 8

export interface SquadFragToolsOpts {
  /** Libellé localisé d'un rôle (manifeste frags.role.<role>). */
  roleLabel: (role: string) => string
  /** Libellé localisé d'une classe (manifeste frags.class.<class>). */
  classLabel: (className: string) => string
  /** Libellé de la ligne agrégée « Autres armes » (i18n). */
  otherWeaponsLabel: string
  /** Cap d'armes gun individuelles ; les autres → « Autres armes ». */
  topGuns: number
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

/** Replie les armes gun au-delà du cap en une seule ligne « Autres armes ». */
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

  const all = [...merged.values()]
  const guns = all.filter((m) => isGunLine(m.class)).sort((a, b) => b.total - a.total)
  const details = all.filter((m) => !isGunLine(m.class))
  const cap = Math.max(0, opts.topGuns)

  const outRows: MergedBar[] = [...guns.slice(0, cap), ...details]
  // Tri par usage : ASC par total escouade (tie-break label).
  outRows.sort((a, b) => (a.total !== b.total ? a.total - b.total : a.label.localeCompare(b.label)))

  // « Autres armes » : agrégat épinglé TOUT EN BAS, HORS tri par usage (un agrégat ne peut
  // pas être classé « significatif »). Le chart squadWeaponKills n'a PAS d'inverse yAxis →
  // la 1re catégorie (index 0) est rendue EN BAS → on PRÉFIXE l'agrégat.
  const overflow = guns.slice(cap)
  if (overflow.length > 0) outRows.unshift(aggregateOverflow(overflow, opts.otherWeaponsLabel))

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
