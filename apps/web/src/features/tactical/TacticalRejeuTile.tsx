/**
 * TacticalRejeuTile — la mini-tuile « Rejeu » d'une contribution de la zone sélectionnée, sur la
 * grammaire de la tuile de match de l'app : bande d'issue de 3 px, puis une GRILLE à deux colonnes
 * `minmax(0, 1fr) auto` — le texte, et les actions.
 *
 * LE TEXTE tient sur trois lignes, et rien n'y est coupé net : ce qui ne tient pas tronque avec une
 * ellipse DANS sa colonne.
 *   - ligne 1 : mode (tronque), score, issue en mot | date · heure à droite ;
 *   - ligne 2 : pastille d'instant, « Tué par <gamertag> » (seul le gamertag tronque) ;
 *   - ligne 3 : arme (tronque), distance | badge de placement à droite (jamais coupé : l'arme cède).
 *
 * LES ACTIONS ont une largeur RÉSERVÉE, identique sur toutes les tuiles que le rejeu existe ou non
 * (les chevrons s'alignent d'une tuile à l'autre) : le bouton de rejeu, sans bordure, puis le
 * chevron, centrés verticalement. Le rejeu n'est posé que si l'artefact existe et que le titre le
 * sert (`MatchReplayLink`). L'infobulle porte le texte complet.
 *
 * Ce que la tuile dit vient de `modeleDeTuile` (`zone.logic.ts`) ; le mot et la couleur de l'issue
 * viennent des correspondances du titre (`useOutcomeMapping`, jeton `outcome-*`).
 */
import { useEffect, useRef } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { TacticalContribution } from '@/lib/api/types'
import { useOutcomeMapping } from '@/lib/i18n/fieldMappings'
import type { Locale } from '@/lib/i18n/locale'
import { MatchReplayLink } from '@/lib/match-nav/MatchReplayLink'
import { outcomeTokenFromCanonical } from '@/lib/outcome-color'
import { useAppShellStore } from '@/stores/appShellStore'

import type { TacticalText } from './i18n'
import { modeleDeTuile } from './zone.logic'

interface TacticalRejeuTileProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  contribution: TacticalContribution
  /** La tuile est celle du point survolé sur le plan : elle ressort. */
  actif?: boolean
  /** Le pointeur entre dans la tuile (`true`) ou la quitte (`false`) : le plan met son point en avant. */
  onSurvol?: (survole: boolean) => void
  /** La tuile est la contribution CHOISIE : bordée de l'accent, sur fond teinté. */
  choisi?: boolean
  /** Clic (ou Entrée / Espace) sur la tuile : la vue bascule le choix. Le bouton de rejeu garde son propre clic. */
  onChoisir?: () => void
}

export function TacticalRejeuTile({ t, locale, playerSlug, contribution: c, actif = false, onSurvol, choisi = false, onChoisir }: TacticalRejeuTileProps) {
  const timezone = useAppShellStore((s) => s.userTimezone)
  const issue = useOutcomeMapping(c.resultat ?? '')
  const jeton = outcomeTokenFromCanonical(c.resultat ?? '')
  const m = modeleDeTuile(t, c, locale, timezone, issue?.label ?? null)
  const ligne = useRef<HTMLLIElement>(null)
  // Le point du plan survolé amène sa tuile dans la liste à défilement interne.
  useEffect(() => {
    if (actif) ligne.current?.scrollIntoView?.({ block: 'nearest' })
  }, [actif])
  const aLigne3 = Boolean(m.arme || m.distance || m.badge)
  return (
    <li
      ref={ligne}
      className={`group relative grid min-h-14 flex-none cursor-pointer grid-cols-[minmax(0,1fr)_auto] items-center gap-x-1 overflow-hidden rounded-md border py-1.5 pl-2.5 pr-1 text-[12.5px] tabular-nums transition-colors hover:bg-accent focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring${choisi ? ' border-primary bg-primary/10' : ` border-border${actif ? ' bg-accent' : ' bg-card'}`}`}
      title={m.titre}
      role="option"
      tabIndex={0}
      aria-selected={choisi}
      onClick={(event) => {
        // Le lien de rejeu de la tuile garde son propre clic : il n'est pas un choix.
        if (!(event.target as HTMLElement).closest('a,button')) onChoisir?.()
      }}
      onKeyDown={(event) => {
        if (event.target === event.currentTarget && (event.key === 'Enter' || event.key === ' ')) {
          event.preventDefault()
          onChoisir?.()
        }
      }}
      data-actif={actif || undefined}
      onPointerEnter={() => onSurvol?.(true)}
      onPointerLeave={() => onSurvol?.(false)}
      data-testid="tactical-rejeu-tile"
    >
      <span
        aria-hidden
        className={`absolute inset-y-0 left-0 w-[3px]${jeton ? '' : ' bg-muted-foreground'}`}
        style={jeton ? { backgroundColor: tokenCssVar(jeton) } : undefined}
      />
      <div className="flex min-w-0 flex-col gap-1" data-testid="tactical-rejeu-texte">
        <div className="flex min-w-0 items-baseline gap-1 whitespace-nowrap leading-4 text-foreground" data-testid="tactical-rejeu-l1">
          {m.mode && <span className="min-w-0 truncate font-medium">{m.mode}</span>}
          {m.score && <span className="flex-none font-bold">{m.score}</span>}
          {issue && (
            <span className="flex-none font-medium" style={jeton ? { color: tokenCssVar(jeton) } : undefined}>
              {issue.label}
            </span>
          )}
          {m.date && <span className="ml-auto flex-none pl-1 text-[10.5px] text-muted-foreground">{m.date}</span>}
        </div>
        <div
          className="flex min-w-0 items-center gap-[5px] whitespace-nowrap text-[11.5px] leading-4 text-muted-foreground"
          data-testid="tactical-rejeu-l2"
        >
          {m.instant && (
            <span className="flex-none rounded bg-muted px-[5px] font-mono text-[11px] leading-4 text-foreground">{m.instant}</span>
          )}
          {m.fait && (
            <span className="flex min-w-0" data-testid="tactical-rejeu-fait">
              {m.faitDecoupe.avant && <span className="flex-none whitespace-pre">{m.faitDecoupe.avant}</span>}
              {m.faitDecoupe.gamertag && <span className="min-w-0 truncate">{m.faitDecoupe.gamertag}</span>}
              {m.faitDecoupe.apres && <span className="flex-none whitespace-pre">{m.faitDecoupe.apres}</span>}
            </span>
          )}
        </div>
        {aLigne3 && (
          <div
            className="flex min-w-0 items-center gap-[5px] whitespace-nowrap text-[11.5px] leading-4 text-muted-foreground"
            data-testid="tactical-rejeu-l3"
          >
            {m.arme && <span className="min-w-0 truncate">{m.arme}</span>}
            {m.distance && <span className="flex-none" data-testid="tactical-rejeu-distance">{m.distance}</span>}
            {m.badge && <span className="ml-auto flex-none rounded-full bg-muted px-1.5 text-[10.5px] leading-4">{m.badge}</span>}
          </div>
        )}
      </div>
      {/* COLONNE DES ACTIONS : largeur RÉSERVÉE (bouton 32 px + chevron 16 px), la même avec ou sans
          rejeu — les chevrons s'alignent d'une tuile à l'autre ; le contenu se range à droite. */}
      <div className="flex w-[50px] flex-none items-center justify-end gap-0.5" data-testid="tactical-rejeu-actions">
        <MatchReplayLink
          available={m.rejeu.disponible}
          matchId={c.match_id}
          playerSlug={playerSlug}
          label={m.rejeu.libelle}
          variant="large"
          search={m.rejeu.search}
        />
        <svg
          viewBox="0 0 16 16"
          aria-hidden
          className={`size-4 flex-none transition-colors ${choisi ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground'}`}
          fill="none"
          stroke="currentColor"
          strokeWidth="1.75"
          data-testid="tactical-rejeu-chevron"
        >
          <path d="M6 3.5 10.5 8 6 12.5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </div>
    </li>
  )
}
