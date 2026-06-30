/**
 * MedalDigest — résumé narratif médailles par joueur sur les matchs partagés.
 *
 * Rendu : une carte par joueur avec top 5 médailles en chips + stats footer +
 * grille déroulante de toutes les médailles.
 *
 * Avatar joueur : emblème Spartan fourni directement par le backend dans
 * entry.emblem_url (chargé en parallèle côté Go via career_progression).
 * Fallback : initiale dans un cercle à la couleur du joueur.
 *
 * Grille adaptative : N joueurs → N colonnes égales (repeat(N, 1fr)).
 */
import { useState } from 'react'
import { tokenCssVar } from '@/lib/accessibility'
import { dropShadowForDifficulty, boxShadowForDifficulty } from '@/lib/medalDifficulty'
import { MedalIcon } from '@/components/ui/MedalIcon'
import type { MedalDigestEntry, MedalDigestItem } from '@/lib/api/types'
import {
  SQUAD_MAIN_PLAYER_TOKEN,
  SQUAD_TEAMMATE_COLOR_TOKENS,
} from './colors'
import type { SquadText } from './i18n'

interface MedalDigestProps {
  entries: MedalDigestEntry[]
  mainPlayer: string
  t: SquadText['medals']
}

function playerColorVar(mainPlayer: string, player: string, allPlayers: string[]): string {
  if (player === mainPlayer) return tokenCssVar(SQUAD_MAIN_PLAYER_TOKEN)
  const idx = allPlayers.filter((p) => p !== mainPlayer).indexOf(player)
  const token = SQUAD_TEAMMATE_COLOR_TOKENS[idx] ?? SQUAD_TEAMMATE_COLOR_TOKENS[0]
  return tokenCssVar(token)
}

function medalTooltip(item: MedalDigestItem): string {
  const name = item.label || `#${item.medal_id}`
  return item.description ? `${name}: ${item.description}` : name
}

/** Emblème rond du joueur — utilise l'URL fournie par le backend (career_progression). */
function PlayerAvatar({ gamertag, color, emblemUrl }: { gamertag: string; color: string; emblemUrl?: string }) {
  if (emblemUrl) {
    return (
      <img
        src={emblemUrl}
        alt={gamertag}
        className="h-8 w-8 rounded-full object-cover flex-shrink-0"
        style={{ boxShadow: `0 0 0 2px ${color}` }}
      />
    )
  }

  return (
    <span
      className="h-8 w-8 rounded-full flex items-center justify-center flex-shrink-0 text-xs font-bold"
      style={{ background: color, color: '#fff' }} // color-allow: blanc structurel sur fond joueur
    >
      {gamertag.charAt(0).toUpperCase()}
    </span>
  )
}

function MedalChip({ item }: { item: MedalDigestItem }) {
  const tip = medalTooltip(item)
  return (
    <span
      className="relative inline-flex items-center gap-1 rounded-full border border-border bg-muted px-2 py-0.5 text-xs"
      title={tip}
    >
      {item.image_url || item.sprite_sheet ? (
        <MedalIcon
          imageUrl={item.image_url}
          spriteSheet={item.sprite_sheet}
          spriteLeft={item.sprite_left}
          spriteTop={item.sprite_top}
          spriteWidth={item.sprite_width}
          spriteHeight={item.sprite_height}
          label={item.label || String(item.medal_id)}
          size={20}
          className="object-contain"
        />
      ) : (
        <span className="h-5 w-5 flex items-center justify-center rounded-full bg-muted-foreground/20 text-2xs font-bold uppercase">
          {(item.label ?? '?').charAt(0)}
        </span>
      )}
      <span className="text-foreground/80 max-w-[7rem] truncate">{item.label || `#${item.medal_id}`}</span>
      <span
        className="rounded-full px-1 text-2xs font-bold leading-tight"
        style={{
          background: 'var(--muted)', // color-allow: structurel — badge count neutre
          color: 'var(--foreground)',
          border: '1.5px solid var(--background)', // color-allow: structurel — séparation badge/icône
          minWidth: '1.1rem',
          textAlign: 'center',
        }}
      >
        ×{item.total_count}
      </span>
    </span>
  )
}

function MedalIconTile({ item }: { item: MedalDigestItem }) {
  const tip = medalTooltip(item)
  const dropGlow = dropShadowForDifficulty(item.difficulty)
  const boxGlow = boxShadowForDifficulty(item.difficulty)
  return (
    <span className="relative flex flex-col items-center gap-0.5" title={tip}>
      {item.image_url || item.sprite_sheet ? (
        <MedalIcon
          imageUrl={item.image_url}
          spriteSheet={item.sprite_sheet}
          spriteLeft={item.sprite_left}
          spriteTop={item.sprite_top}
          spriteWidth={item.sprite_width}
          spriteHeight={item.sprite_height}
          label={item.label || String(item.medal_id)}
          size={40}
          className="object-contain"
          style={dropGlow ? { filter: dropGlow } : undefined}
        />
      ) : (
        <span
          className="h-10 w-10 flex items-center justify-center rounded-full bg-muted text-xs font-bold uppercase"
          style={boxGlow ? { boxShadow: boxGlow } : undefined}
        >
          {(item.label ?? '?').charAt(0)}
        </span>
      )}
      <span
        className="rounded-full px-1 text-2xs font-bold leading-tight"
        style={{
          background: 'var(--muted)', // color-allow: structurel — badge count neutre
          color: 'var(--foreground)',
          border: '1.5px solid var(--background)', // color-allow: structurel — séparation badge/icône
          minWidth: '1.1rem',
          textAlign: 'center',
        }}
      >
        ×{item.total_count}
      </span>
    </span>
  )
}

const CATEGORY_ORDER = ['multikill', 'spree', 'skill', 'style', 'mode', 'proficiency']

function MedalExpandedGrid({
  medals,
  categoryLabels,
}: {
  medals: MedalDigestItem[]
  categoryLabels: SquadText['medals']['categoryLabels']
}) {
  const groups = new Map<string, MedalDigestItem[]>()
  for (const m of medals) {
    const key = m.category || 'other'
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(m)
  }
  const orderedKeys = [
    ...CATEGORY_ORDER.filter((k) => groups.has(k)),
    ...[...groups.keys()].filter((k) => !CATEGORY_ORDER.includes(k)),
  ]
  return (
    <div className="pt-1 space-y-3">
      {orderedKeys.map((cat) => (
        <div key={cat}>
          <p className="text-2xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
            {categoryLabels[cat as keyof typeof categoryLabels] ?? cat}
          </p>
          <div className="flex flex-wrap gap-3">
            {groups.get(cat)!.map((m) => (
              <MedalIconTile key={m.medal_id} item={m} />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

// Fallback quand personal_score est absent : poids par difficulté.
const DIFFICULTY_WEIGHT: Record<string, number> = { Normal: 1, Heroic: 3, Legendary: 5, Mythic: 10 }

function dominantCategoryFor(medals: MedalDigestItem[]): string | null {
  const totals = new Map<string, number>()
  for (const m of medals) {
    const cat = m.category || 'other'
    const w = m.personal_score && m.personal_score > 0
      ? m.personal_score
      : (DIFFICULTY_WEIGHT[m.difficulty ?? 'Normal'] ?? 1)
    totals.set(cat, (totals.get(cat) ?? 0) + m.total_count * w)
  }
  let best: string | null = null
  let bestVal = 0
  for (const [cat, val] of totals) {
    if (val > bestVal) { bestVal = val; best = cat }
  }
  return best
}

function PlayerMedalCard({
  entry,
  color,
  t,
  expanded,
  onToggle,
}: {
  entry: MedalDigestEntry
  color: string
  t: SquadText['medals']
  /** État déplié PARTAGÉ : un clic déplie/replie toutes les cartes joueur. */
  expanded: boolean
  onToggle: () => void
}) {
  // Le contrat OpenAPI expose top_medals / all_medals en `T[] | null` (le Go
  // peut renvoyer null) → vues non-null réutilisées dans le rendu.
  const allMedals = entry.all_medals ?? []
  const topMedals = entry.top_medals ?? []
  const domCat = dominantCategoryFor(allMedals)
  const domCatLabel = domCat
    ? (t.categoryLabels[domCat as keyof typeof t.categoryLabels] ?? domCat)
    : null

  return (
    <div
      className="flex flex-col gap-3 rounded-lg border border-border bg-card p-4"
      style={{ borderLeft: `4px solid ${color}` }}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <PlayerAvatar gamertag={entry.player} color={color} emblemUrl={entry.emblem_url} />
          <span className="font-semibold text-sm text-foreground truncate">{entry.player}</span>
        </div>
        {domCatLabel && (
          <div className="flex flex-col items-end shrink-0">
            <span className="text-[9px] uppercase tracking-wider text-muted-foreground leading-none">
              {t.dominantCategory}
            </span>
            <span
              className="text-xs font-semibold mt-0.5 rounded px-1.5 py-0.5"
              style={{ background: `${color}22`, color }}
            >
              {domCatLabel}
            </span>
          </div>
        )}
      </div>

      {topMedals.length > 0 && (
        <div>
          <p className="text-[9px] uppercase tracking-wider text-muted-foreground mb-1.5">
            {t.topMedals}
          </p>
          <div className="flex flex-wrap gap-1.5">
            {topMedals.map((m) => (
              <MedalChip key={m.medal_id} item={m} />
            ))}
          </div>
        </div>
      )}

      <div className="flex gap-4 text-xs text-muted-foreground border-t border-border pt-2">
        <span>
          <span className="font-medium text-foreground">{entry.distinct_types}</span>{' '}
          {t.statsDistinct}
        </span>
        <span>
          <span className="font-medium text-foreground">{entry.avg_per_match.toFixed(1)}</span>{' '}
          {t.statsAvg}
        </span>
        <span>
          <span className="font-medium text-foreground">{entry.peak_in_match}</span> {t.statsPeak}
        </span>
      </div>

      {allMedals.length > topMedals.length && (
        <>
          <button
            type="button"
            className="text-xs text-muted-foreground hover:text-foreground underline-offset-2 hover:underline text-left"
            onClick={onToggle}
          >
            {expanded ? t.collapseLabel : `${t.expandLabel} (${allMedals.length})`}
          </button>
          {expanded && (
            <MedalExpandedGrid medals={allMedals} categoryLabels={t.categoryLabels} />
          )}
        </>
      )}
    </div>
  )
}

export function MedalDigest({ entries, mainPlayer, t }: MedalDigestProps) {
  // État « voir toutes les médailles » PARTAGÉ entre toutes les cartes joueur :
  // un clic sur n'importe quel bouton déplie/replie tout le monde (demande user).
  const [expanded, setExpanded] = useState(false)

  if (!entries || entries.length === 0) {
    // Bloc vide : cadre type carte joueur (rounded-lg border bg-card) + message,
    // au lieu de faire disparaître la section.
    return (
      <div className="flex min-h-[120px] items-center justify-center rounded-lg border border-border bg-card text-sm text-muted-foreground">
        {t.noMedals}
      </div>
    )
  }

  const allPlayers = entries.map((e) => e.player)

  // Toujours N colonnes égales — squads 2-4 joueurs, largeur fixe.
  const gridCols = `repeat(${entries.length}, 1fr)`

  // Bloc « sorti » : grille seule, sans Card ni titre interne. Le titre
  // « Médailles — Résumé de l'escouade » est porté par un titre de section
  // dans SquadSynergiesPage.
  return (
    <div style={{ display: 'grid', gridTemplateColumns: gridCols, gap: '1rem' }}>
      {entries.map((entry) => (
        <PlayerMedalCard
          key={entry.player}
          entry={entry}
          color={playerColorVar(mainPlayer, entry.player, allPlayers)}
          t={t}
          expanded={expanded}
          onToggle={() => setExpanded((e) => !e)}
        />
      ))}
    </div>
  )
}
