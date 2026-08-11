/**
 * MatchKillFeed — le kill feed de la carte « Dominance », rendu en DOM.
 *
 * POURQUOI EN DOM ET PLUS EN ECHARTS. Le feed était deux séries `scatter` sur des lanes
 * du graphe. Un symbole image ECharts (`symbol: 'image://…'`) est dessiné TEL QUEL : il
 * ne se teinte pas. Or l'icône d'arme extraite du jeu est un MASQUE blanc porté par
 * l'alpha — servie telle quelle elle disparaît en thème clair, et elle ne peut pas
 * prendre la couleur de l'équipe du tueur. Le DOM, lui, teinte un masque en une ligne de
 * CSS (cf. `WeaponIcon`), et donne en prime le survol accessible et le texte
 * sélectionnable. Le graphe garde ce qui est vraiment de son ressort : l'histogramme.
 *
 * L'ALIGNEMENT SUR L'AXE DU GRAPHE est reproduit à l'identique, pas approché. L'axe X du
 * graphe est un axe de CATÉGORIES (une par tranche) avec `boundaryGap`, tracé dans une
 * grille encartée de `GRID_INSET_PX` à gauche comme à droite. ECharts place la valeur
 * `binIdx - 0.5 + frac` au pixel `gauche + (binIdx + frac) × largeurDeBande`. Les lanes
 * ci-dessous appliquent la MÊME formule en pourcentage de la zone tracée, ce qui la rend
 * exacte à toute largeur, sans ResizeObserver ni recalcul au redimensionnement.
 *
 * LA COULEUR est celle de l'IDENTITÉ de l'équipe, résolue par la cascade EXACTE de
 * l'en-tête du scoreboard (`MatchScoreboard.tsx`) : couleur fournie par le backend
 * (`team_color`, Halo 5) d'abord, sinon la map officielle par team_id (Halo Infinite :
 * Eagle bleu, Cobra rouge…), sinon le token sémantique allié/ennemi — surchargeable par
 * les réglages d'accessibilité. Aucun hex n'est écrit ici : ils vivent dans `lib/halo/`,
 * référentiel de couleurs de jeu au même titre que `rarity.ts`.
 */
import { useMemo } from 'react'
import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility'
import { displayPlayerName } from '@/lib/players/displayName'
import { parseTeamSideID, resolveTeamColorFromID } from '@/lib/halo/teamNames'
import type { MatchScoreboardRow, MatchTugOfWarBin } from '@/lib/api/types'
import type { MomentumKill } from './_momentum'
import type { MatchViewText } from './i18n'

/** Encart horizontal de la grille ECharts, en pixels (cf. `grid.left` / `grid.right`). */
const GRID_INSET_PX = 14
/** Gabarit d'une icône d'arme dans une lane : le format bandeau de l'atlas kill feed. */
const ICON_W = 22
const ICON_H = 9
/** Diamètre du repère servi quand l'arme est inconnue — le repli, identique à l'ancien point. */
const DOT_PX = 7

export interface KillFeedLaneKill extends MomentumKill {
  /** Nom d'affichage déjà résolu (bots compris). */
  gamertag: string
  /** Couleur d'identité de l'équipe du tueur, prête à être posée en CSS. */
  color: string
}

interface Props {
  bins: MatchTugOfWarBin[]
  kills: MomentumKill[]
  scoreboard: MatchScoreboardRow[] | null | undefined
  /** xuid → nom d'affichage et appartenance, déjà résolus par le parent. */
  xuidMeta: ReadonlyMap<string, { gamertag: string; ally: boolean }>
  t: MatchViewText
}

function formatMmSs(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.max(0, Math.floor(seconds % 60))
  return `${m}:${s.toString().padStart(2, '0')}`
}

/**
 * teamColorResolver — la cascade de l'en-tête du scoreboard, en une fonction.
 *
 * Réutilisée telle quelle plutôt que réinventée : deux cascades divergentes, ce sont
 * deux couleurs pour la même équipe sur la même page.
 */
function teamColorResolver(
  scoreboard: MatchScoreboardRow[] | null | undefined,
): (teamID: number | null, ally: boolean) => string {
  const backendByTeamID = new Map<number, string>()
  for (const r of scoreboard ?? []) {
    const id = parseTeamSideID(r.team_side)
    if (id != null && r.team_color && !backendByTeamID.has(id)) {
      backendByTeamID.set(id, r.team_color)
    }
  }
  return (teamID, ally) => {
    if (teamID != null) {
      const backend = backendByTeamID.get(teamID)
      if (backend) return backend
      const official = resolveTeamColorFromID(teamID)
      if (official) return official
    }
    return tokenCssVar(ally ? 'team-ally' : 'team-enemy')
  }
}

/** Position horizontale d'un kill dans la zone tracée, en CSS. Voir l'en-tête. */
function laneLeft(k: MomentumKill, nbBins: number): string {
  const ratio = nbBins > 0 ? (k.binIdx + k.fracInBin) / nbBins : 0
  return `calc(${GRID_INSET_PX}px + (100% - ${GRID_INSET_PX * 2}px) * ${ratio})`
}

/** Libellé de survol : qui, avec quoi, quand. L'arme n'y figure que si elle est connue. */
function killTitle(k: KillFeedLaneKill, unknownWeapon: string): string {
  const weapon = k.weaponImageUrl ? k.weaponLabel || '' : unknownWeapon
  const parts = [k.gamertag, weapon, formatMmSs(k.tMs / 1000)].filter((p) => p !== '')
  return parts.join(' — ')
}

/** Une lane : la ligne de temps d'une équipe, et ses kills posés dessus. */
function KillLane({
  kills,
  nbBins,
  label,
  unknownWeapon,
}: {
  kills: KillFeedLaneKill[]
  nbBins: number
  label: string
  unknownWeapon: string
}) {
  return (
    <div className="relative h-6" role="list" aria-label={label}>
      <div
        aria-hidden
        className="absolute top-1/2 border-t border-border"
        style={{ left: GRID_INSET_PX, right: GRID_INSET_PX }}
      />
      {kills.map((k) => (
        <span
          key={`${k.xuid}-${k.tMs}`}
          role="listitem"
          title={killTitle(k, unknownWeapon)}
          className="group absolute top-1/2 flex -translate-x-1/2 -translate-y-1/2 items-center justify-center hover:z-10"
          style={{ left: laneLeft(k, nbBins), color: k.color }}
        >
          {k.weaponImageUrl ? (
            <WeaponIcon
              imageUrl={k.weaponImageUrl}
              tinted={k.weaponTinted}
              label={k.weaponLabel || unknownWeapon}
              width={ICON_W}
              height={ICON_H}
            />
          ) : (
            /* Repli assumé : la source du dégât n'est pas identifiable sans ambiguïté.
               Un repère neutre, jamais l'icône d'une autre arme. */
            <span
              aria-hidden
              className="rounded-full"
              style={{ width: DOT_PX, height: DOT_PX, backgroundColor: 'currentColor', opacity: 0.7 }}
            />
          )}
          {/* Le NOM DU TUEUR, dans la couleur de son équipe — même lecture que l'en-tête
              du scoreboard. Au survol seulement : N noms superposés sur une lane de
              80 kills seraient illisibles, et le nom compte au moment où on interroge
              un kill précis. `title` couvre le même besoin pour le clavier et le lecteur
              d'écran. */}
          <span
            className="pointer-events-none absolute bottom-full left-1/2 mb-0.5 hidden -translate-x-1/2
                       whitespace-nowrap rounded bg-popover px-1 py-px text-3xs font-medium
                       shadow-sm group-hover:block"
            style={{ color: k.color }}
          >
            {k.gamertag}
          </span>
        </span>
      ))}
    </div>
  )
}

export function MatchKillFeed({ bins, kills, scoreboard, xuidMeta, t }: Props) {
  const decorated = useMemo<KillFeedLaneKill[]>(() => {
    const colorOf = teamColorResolver(scoreboard)
    return kills
      .map((k) => ({
        ...k,
        gamertag: displayPlayerName(xuidMeta.get(k.xuid)?.gamertag, k.xuid),
        color: colorOf(k.teamID, k.ally),
      }))
      .sort((a, b) => a.tMs - b.tMs)
  }, [kills, scoreboard, xuidMeta])

  if (decorated.length === 0) return null

  const ally = decorated.filter((k) => k.ally)
  const enemy = decorated.filter((k) => !k.ally)
  const avecIcone = decorated.filter((k) => k.weaponImageUrl !== '').length

  return (
    <div className="flex flex-col gap-1 px-2 pb-1">
      <div className="flex items-baseline justify-between px-3">
        <span className="text-3xs uppercase tracking-wider text-muted-foreground">
          {t.combatKillFeedTitle}
        </span>
        {/* La couverture est AFFICHÉE, pas cachée : elle ne dépend pas du rendu mais de
            l'avancement du décodage de film, qui bouge dans le temps. Sans ce compteur,
            un feed sans icône ressemblerait à une panne. */}
        <span className="text-3xs text-muted-foreground">
          {t.combatKillFeedCoverage(avecIcone, decorated.length)}
        </span>
      </div>
      <KillLane
        kills={ally}
        nbBins={bins.length}
        label={t.combatTeamLabel}
        unknownWeapon={t.combatKillFeedUnknownWeapon}
      />
      <KillLane
        kills={enemy}
        nbBins={bins.length}
        label={t.combatEnemyLabel}
        unknownWeapon={t.combatKillFeedUnknownWeapon}
      />
    </div>
  )
}
