/**
 * MatchViewTabArsenal — contenu de l'onglet « Armes et terrain » de la page match.
 *
 * Né « Contrôle » le 2026-09-19 (plan `.ai/V7.5/PLAN_AJUSTEMENTS_PRE_V75_2026-09-19.md`,
 * lot 2) avec les trois blocs qui disent QUI A TENU QUOI sur le terrain, sortis de
 * « Chronologie » où ils s'intercalaient entre deux courbes temporelles sans en être une.
 *
 * RENOMMÉ ET ÉLARGI LE 2026-09-22 : il reprend de « Général » la répartition des frags et
 * la distance des frags, qui répondent à la même question que les trois autres — AVEC QUOI
 * et OÙ — alors que « Général » doit se lire comme le bilan du match. Un onglet, un axe de
 * lecture : « Contrôle » ne nommait plus son contenu une fois les armes arrivées.
 *
 * DEUX SECTIONS TITRÉES, chacune sur au moins deux blocs :
 *   1. « Frags et armes » — la répartition des frags (`MatchFragCard` : sunburst
 *      classe→rôle + détail par arme) et la distance des frags par arme
 *      (`MatchKillDistanceSection`) ;
 *   2. « Équipement et terrain » — le bilan des gestes d'équipement
 *      (`MatchEquipmentUsageSection`), le ramasseur de chaque socle d'arme
 *      (`MatchPadControlSection`) et les positions du film sur le plan
 *      (`MatchPositionsHeatmap`).
 * Les deux premiers blocs de la seconde section lisent le MÊME artefact de rejeu (une seule
 * clé de cache) ; le troisième lit les positions keyframe, tirées par la page UNIQUEMENT
 * quand cet onglet est actif. Chaque bloc se masque lui-même sans donnée : l'onglet ne pose
 * aucun cadre vide.
 */
import { MatchEquipmentUsageSection } from '@/features/match-replay/MatchEquipmentUsageSection'
import { MatchPadControlSection } from '@/features/match-replay/MatchPadControlSection'
import { DetailSection } from '@/components/ui/detail-section'
import type {
  FragDistribution,
  MatchKillDistancePlayer,
  MatchPlayerPosition,
  MatchRosterRow,
  MatchScoreboardRow,
  MatchWeaponKill,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { MatchFragCard } from './MatchFragCard'
import { MatchKillDistanceSection } from './MatchKillDistanceSection'
import { MatchPositionsHeatmap } from './MatchPositionsHeatmap'
import type { MatchViewText } from './i18n'

interface Props {
  playerSlug: string
  matchId: string
  replayAvailable: boolean
  scoreboard: MatchScoreboardRow[]
  roster: MatchRosterRow[]
  /** Répartition des frags du viewer (classe -> rôle) — absente sur un match sans frags. */
  fragDistribution: FragDistribution | null | undefined
  /** Frags par arme du viewer, source du « Détails des frags ». */
  weaponKills: MatchWeaponKill[]
  /** Distance des frags par arme et par joueur — absente sans positions de film. */
  killDistance: MatchKillDistancePlayer[] | null | undefined
  meXUID: string | null
  friendGamertags: readonly string[]
  matchPositions: MatchPlayerPosition[] | undefined
  locale: Locale
  t: MatchViewText
}

export function MatchViewTabArsenal({
  playerSlug,
  matchId,
  replayAvailable,
  scoreboard,
  roster,
  fragDistribution,
  weaponKills,
  killDistance,
  meXUID,
  friendGamertags,
  matchPositions,
  locale,
  t,
}: Props) {
  return (
    <div className="space-y-6">
      <DetailSection title={t.sectionKillsWeapons}>
        {/* Répartition des frags v2 : sunburst (classe→rôle, 2/3 de largeur) + breakdown
            par arme (1/3). MatchFragCard porte sa propre grille (breakdown pleine largeur
            si le sunburst n'a pas de données) et rend null sans aucune donnée — pas de
            wrapper ici, sinon un gap fantôme resterait quand la carte est absente. Non
            gaté : Infinite = classes sans Spartan ; Halo 5 = avec (capability
            native_kill_mechanics côté backend). */}
        <MatchFragCard distribution={fragDistribution} weapons={weaponKills} />
        {/* Distance par arme, par joueur (LOT G.3, 2026-08-30) — juste après les stats
            d'armes du viewer. Scoreboard passé pour gamertag + total de kills (le DTO
            backend ne porte que le xuid). DEUX PORTES, portées par la section elle-même :
            elle rend null si le TITRE ne déclare pas `film.kill_positions` (rien à espérer,
            jamais), et affiche un état vide explicite si le titre les produit mais pas pour
            CE match. Pas de wrapper ici : un gap fantôme resterait quand elle est absente. */}
        <MatchKillDistanceSection
          players={killDistance}
          scoreboard={scoreboard}
          roster={roster}
          meXUID={meXUID}
          friendGamertags={friendGamertags}
          t={t}
        />
      </DetailSection>

      <DetailSection title={t.sectionEquipmentTerrain}>
        <MatchEquipmentUsageSection
          playerSlug={playerSlug}
          matchId={matchId}
          replayAvailable={replayAvailable}
          scoreboard={scoreboard}
          locale={locale}
        />
        <MatchPadControlSection
          playerSlug={playerSlug}
          matchId={matchId}
          replayAvailable={replayAvailable}
          scoreboard={scoreboard}
          locale={locale}
        />
        <MatchPositionsHeatmap
          playerSlug={playerSlug}
          matchId={matchId}
          positions={matchPositions}
          locale={locale}
        />
      </DetailSection>
    </div>
  )
}
