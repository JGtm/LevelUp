/**
 * MatchCard — tuile de match (Sprint 56).
 *
 * Affiche :
 *  - Image de la map (h-48, object-cover)
 *  - Titre centré `mode sur carte`
 *  - Playlist en sous-titre
 *  - Section score colorée selon le résultat avec badges narratifs
 */
import type { RecentMatchItem } from '@/lib/api/types'
import { getPerfColor } from '@/lib/perf-color'
import { getMatchCardOutcomeStyle, getMatchNarrativeBadgeMeta } from './match-card-presentation'
import { CitationProgressRing } from './citation-progress-ring'
import { skillDeltaScale, kdScale, mmrDeltaScale } from '@/lib/accessibility/scales'
import { tokenCssVar } from '@/lib/accessibility'

export interface MatchCardProps {
  match: RecentMatchItem
  locale?: 'fr' | 'en'
  timezone?: string
  onClick?: () => void
  onToggleFavorite?: () => void
  favoriteDisabled?: boolean
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function formatMatchDuration(secs: number): string {
  const m = Math.floor(secs / 60)
  const s = secs % 60
  return `${m}m ${s.toString().padStart(2, '0')}s`
}

function formatMatchDateTime(isoDate: string, timezone: string, locale: 'fr' | 'en'): string {
  const date = new Date(isoDate)
  if (isNaN(date.getTime())) return ''
  const intlLocale = locale === 'en' ? 'en-GB' : 'fr-FR'
  return new Intl.DateTimeFormat(intlLocale, {
    timeZone: timezone,
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function normalizeModeLabel(modeLabel: string | null | undefined, mapLabel: string | null | undefined): string | null {
  if (!modeLabel) {
    return null
  }

  let normalized = modeLabel.trim()
  if (!normalized) {
    return null
  }

  const spacedSeparatorIndex = normalized.indexOf(' : ')
  if (spacedSeparatorIndex > 0) {
    normalized = normalized.slice(0, spacedSeparatorIndex).trim()
  } else {
    const separatorIndex = normalized.lastIndexOf(':')
    if (separatorIndex >= 0 && separatorIndex < normalized.length - 1) {
      normalized = normalized.slice(separatorIndex + 1).trim()
    }
  }

  if (mapLabel?.trim()) {
    const escapedMap = escapeRegExp(mapLabel.trim())
    normalized = normalized.replace(new RegExp(`\\s+(?:on|sur)\\s+${escapedMap}$`, 'i'), '')
  } else {
    normalized = normalized.replace(/\s+(?:on|sur)\s+.+$/i, '')
  }

  normalized = normalized.replace(/\s*-\s*(?:Forge|Ranked)\b/gi, '').trim()
  return normalized || modeLabel.trim()
}
function buildMatchHeading(match: RecentMatchItem, locale: 'fr' | 'en'): string {
  const normalizedMode = normalizeModeLabel(match.mode_ui, match.map_ui)
  const connector = locale === 'en' ? 'on' : 'sur'

  if (normalizedMode && match.map_ui) {
    return `${normalizedMode} ${connector} ${match.map_ui}`
  }

  return normalizedMode ?? match.map_ui ?? match.title
}

export function MatchCard({ match: m, locale = 'fr', timezone = 'UTC', onClick, onToggleFavorite, favoriteDisabled }: MatchCardProps) {
  const heading = buildMatchHeading(m, locale)
  const outcomeStyle = getMatchCardOutcomeStyle(m.outcome_tone)
  const scoreLabel = m.score_label?.trim() ?? ''
  const narrativeBadges = m.narrative_badges ?? []

  const perfScore = m.performance_score_relative
  const skillValue = m.skill_rating_value
  const skillType = m.skill_rating_type ?? 'LUSR'
  const skillTierLabel = m.skill_tier_label
  const skillDelta = m.skill_rating_delta
  const skillPlaylist = m.skill_playlist_group
  const skillBadgeURL = m.skill_rank_image_url
  const hasPerfOrSkill = perfScore != null || skillValue != null

  const kills = m.kills ?? 0
  const assists = m.assists ?? 0
  const deaths = m.deaths ?? 0
  const hasKDA = m.kills != null || m.assists != null || m.deaths != null
  const kdaTotal = kills + assists + deaths

  const offConv = m.offensive_conversion ?? 0
  const defRes = m.defensive_resistance ?? 0
  const hasDamageBar = m.offensive_conversion != null || m.defensive_resistance != null
  const damageTotal = offConv + defRes

  const hasAccuracyLine = m.accuracy != null || m.avg_life_secs != null

  const isWithFriends = m.is_with_friends
  const rankInTeam = m.rank_in_team
  const hasMatchMeta = isWithFriends != null || rankInTeam != null

  return (
    <div
      className="rounded-xl overflow-hidden border border-border bg-card flex flex-col h-full transition-colors"
    >
      {/* Image de la map — ratio 16/9, rogné sans déformation */}
      <div className="relative aspect-video bg-muted overflow-hidden flex-shrink-0">
        {m.map_image_url ? (
          <img
            src={m.map_image_url}
            alt={m.map_ui ?? m.title}
            className="w-full h-full object-cover"
            loading="lazy"
            onError={(e) => {
              e.currentTarget.style.display = 'none'
              e.currentTarget.nextElementSibling?.removeAttribute('style')
            }}
          />
        ) : null}
        <div
          className="w-full h-full flex items-center justify-center text-muted-foreground text-xs"
          style={m.map_image_url ? { display: 'none' } : undefined}
        >
          {m.map_ui ?? 'Map inconnue'}
        </div>

        {/* Bouton favori en overlay coin supérieur gauche */}
        {onToggleFavorite && (
          <button
            type="button"
            aria-label={m.is_favorite ? 'Retirer des favoris' : 'Ajouter aux favoris'}
            disabled={favoriteDisabled}
            onClick={(e) => {
              e.stopPropagation()
              onToggleFavorite()
            }}
            className="absolute top-2 left-2 rounded-full p-1 bg-black/40 hover:bg-black/60 transition-colors disabled:opacity-40"
          >
            {m.is_favorite ? (
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="#f59e0b" className="h-4 w-4" aria-hidden="true"> {/* color-allow: amber gold pour étoile favori — CLAUDE.md §20 (warning/amber UI générique) */}
                <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
              </svg>
            ) : (
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="#f59e0b" className="h-4 w-4" aria-hidden="true"> {/* color-allow: amber gold pour étoile favori (outline) — CLAUDE.md §20 */}
                <path strokeLinecap="round" strokeLinejoin="round" d="M11.48 3.499a.562.562 0 011.04 0l2.125 5.111a.563.563 0 00.475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 00-.182.557l1.285 5.385a.562.562 0 01-.84.61l-4.725-2.885a.563.563 0 00-.586 0L6.982 20.54a.562.562 0 01-.84-.61l1.285-5.386a.562.562 0 00-.182-.557l-4.204-3.602a.562.562 0 01.321-.988l5.518-.442a.563.563 0 00.475-.345L11.48 3.5z" />
              </svg>
            )}
          </button>
        )}
      </div>

      {/* Corps */}
      <div className="flex flex-1 flex-col gap-3 px-3 py-3">
        <div className="min-h-[3.5rem] flex flex-col justify-center space-y-1 text-center">
          {onClick ? (
            <button
              type="button"
              onClick={onClick}
              className="group mx-auto inline-flex items-center gap-1 text-sm font-semibold text-foreground leading-tight hover:underline cursor-pointer bg-transparent border-none p-0"
            >
              {heading}
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor" className="h-3 w-3 shrink-0 opacity-40 group-hover:opacity-90 transition-opacity" aria-hidden="true">
                <path d="M6.22 8.72a.75.75 0 0 0 1.06 1.06l5.22-5.22v1.69a.75.75 0 0 0 1.5 0v-3.5a.75.75 0 0 0-.75-.75h-3.5a.75.75 0 0 0 0 1.5h1.69L6.22 8.72Z" />
                <path d="M3.5 6.75c0-.69.56-1.25 1.25-1.25H7A.75.75 0 0 0 7 4H4.75A2.75 2.75 0 0 0 2 6.75v4.5A2.75 2.75 0 0 0 4.75 14h4.5A2.75 2.75 0 0 0 12 11.25V9a.75.75 0 0 0-1.5 0v2.25c0 .69-.56 1.25-1.25 1.25h-4.5c-.69 0-1.25-.56-1.25-1.25v-4.5Z" />
              </svg>
            </button>
          ) : (
            <p className="text-sm font-semibold text-foreground leading-tight">
              {heading}
            </p>
          )}
          {m.playlist_ui && (
            <p className="text-xs text-muted-foreground leading-tight">
              {m.playlist_ui}
            </p>
          )}
          <div
            data-testid="match-card-badges-row"
            className="min-h-[1.625rem] flex flex-wrap items-center justify-center gap-1.5 pt-1"
          >
            {narrativeBadges.map((badgeType) => {
              const badgeMeta = getMatchNarrativeBadgeMeta(badgeType)
              if (!badgeMeta) return null
              return (
                <span
                  key={badgeType}
                  className="rounded px-2 py-1 text-[10px] font-black uppercase tracking-[0.12em]"
                  style={{
                    backgroundColor: badgeMeta.color,
                    color: badgeMeta.textColor,
                  }}
                >
                  {badgeMeta.label}
                </span>
              )
            })}
          </div>
        </div>

        <div
          data-testid="match-card-stats-panel"
          className="-mx-3 flex flex-col items-center justify-center border-y px-3 py-2 text-center"
          style={{
            backgroundColor: outcomeStyle.panelBackground,
            borderColor: outcomeStyle.panelBorder,
          }}
        >
          <p
            data-testid="match-card-score"
            className="text-2xl font-bold leading-none tracking-tight"
            style={{ color: outcomeStyle.scoreColor }}
          >
            {scoreLabel}
          </p>
        </div>

        {/* Badge solo/escouade + placement — zone réservée h fixe pour alignement */}
        <div className="min-h-[2rem] flex items-center justify-center gap-3">
          {hasMatchMeta && (
            <>
            {isWithFriends != null && (
              <span
                className="rounded-full px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wider leading-none"
                style={isWithFriends
                  ? { backgroundColor: 'rgba(56,189,248,0.15)', color: '#38bdf8' } // color-allow: bleu sky pour pill "Escouade"
                  : { backgroundColor: 'rgba(168,85,247,0.15)', color: '#a855f7' } // color-allow: violet pour pill "Solo"
                }
              >
                {isWithFriends ? 'Escouade' : 'Solo'}
              </span>
            )}
            {rankInTeam != null && (
              <span className="text-[10px] font-medium text-muted-foreground leading-none">
                Placement : #{rankInTeam}
              </span>
            )}
            </>
          )}
        </div>

        {/* Bloc perf + skill rating + barre KDA */}
        {(hasPerfOrSkill || hasKDA || hasDamageBar) && (
          <div className="-mx-3 mb-2 bg-white/5 overflow-hidden">
            {/* Ligne du haut : Performance + Skill — centrée */}
            {hasPerfOrSkill && (
              <div className="flex items-center justify-center gap-0 pt-3 pb-1.5">
                {/* Colonne gauche : Performance */}
                {perfScore != null && (
                  <div className="flex flex-col items-center justify-center shrink-0 px-3 gap-0.5">
                    <span
                      className="text-[36px] font-black leading-none"
                      style={{ color: getPerfColor(perfScore) }}
                    >
                      {perfScore}
                    </span>
                    <span className="text-[10px] font-medium leading-none text-muted-foreground">Performance</span>
                  </div>
                )}
                {/* Séparateur vertical fin blanc */}
                {perfScore != null && skillValue != null && (
                  <div className="w-px self-stretch bg-white/20 shrink-0 my-1" />
                )}
                {/* Colonne droite : Skill rating */}
                {skillValue != null && (
                  <div className="flex items-center gap-2 px-3">
                    {/* Badge de rang */}
                    {skillBadgeURL && (
                      <img
                        src={skillBadgeURL}
                        alt={skillTierLabel ?? skillType}
                        className="h-[44px] w-[44px] shrink-0 object-contain"
                        loading="lazy"
                      />
                    )}
                    {/* Infos texte */}
                    <div className="flex flex-col gap-1">
                      {/* Titre du rang */}
                      {skillTierLabel && (
                        <span className="text-sm font-bold text-white leading-none">
                          {skillTierLabel}
                        </span>
                      )}
                      {/* Delta sur sa propre ligne */}
                      {skillDelta != null && (
                        <span
                          className="text-xs font-semibold leading-none"
                          style={{ color: tokenCssVar(skillDeltaScale(skillDelta)) }}
                        >
                          {skillDelta >= 0 ? `+${skillDelta.toFixed(1)}` : skillDelta.toFixed(1)} pts
                        </span>
                      )}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Barre composite frags / assistances / décès */}
            {hasKDA && (
              <div data-testid="match-card-kda-bar" className="px-3 pt-2.5 pb-2 space-y-1.5">
                <div className="h-2 w-full rounded-full overflow-hidden flex">
                  {kills > 0 && <div className="h-full" style={{ width: kdaTotal > 0 ? `${(kills / kdaTotal) * 100}%` : '0%', backgroundColor: tokenCssVar('outcome-win') }} />}
                  {assists > 0 && <div className="h-full" style={{ width: kdaTotal > 0 ? `${(assists / kdaTotal) * 100}%` : '0%', backgroundColor: tokenCssVar('perf-tier-2') }} />}
                  {deaths > 0 && <div className="h-full" style={{ width: kdaTotal > 0 ? `${(deaths / kdaTotal) * 100}%` : '0%', backgroundColor: tokenCssVar('outcome-loss') }} />}
                </div>
                <div className="flex justify-center gap-5 mt-2">
                  <div className="flex flex-col items-center gap-0.5">
                    <span className="text-sm font-bold text-white leading-none">{kills}</span>
                    <span className="text-[10px] font-medium leading-none" style={{ color: tokenCssVar('outcome-win') }}>frags</span>
                  </div>
                  <div className="flex flex-col items-center gap-0.5">
                    <span className="text-sm font-bold text-white leading-none">{assists}</span>
                    <span className="text-[10px] font-medium leading-none" style={{ color: tokenCssVar('perf-tier-2') }}>assist.</span>
                  </div>
                  <div className="flex flex-col items-center gap-0.5">
                    <span className="text-sm font-bold text-white leading-none">{deaths}</span>
                    <span className="text-[10px] font-medium leading-none" style={{ color: tokenCssVar('outcome-loss') }}>morts</span>
                  </div>
                </div>
              </div>
            )}

            {/* Barre composite rendement offensif / résistance défensive */}
            {hasDamageBar && (
              <>
                {m.kda != null && (
                  <div className="flex items-center justify-center gap-0 pt-3 pb-2">
                    {/* Colonne gauche : Tirs à la tête — espace réservé même si absent */}
                    <div className="w-16 flex flex-col items-center gap-0.5">
                      {m.headshot_kills != null && m.headshot_kills > 0 ? (
                        <>
                          <span className="text-lg font-black text-white leading-none">{m.headshot_kills}</span>
                          <span className="text-[10px] font-medium leading-none text-muted-foreground">T. tête</span>
                        </>
                      ) : null}
                    </div>
                    {/* Centre : KDA */}
                    <div className="flex flex-col items-center gap-0.5 px-3">
                      <span
                        className="text-2xl font-black leading-none"
                        style={{ color: tokenCssVar(kdScale(m.kda)) }}
                      >
                        {m.kda.toFixed(2)}
                      </span>
                      <span className="text-[10px] font-medium leading-none text-muted-foreground">FDA</span>
                    </div>
                    {/* Colonne droite : Frags parfaits — espace réservé même si absent */}
                    <div className="w-16 flex flex-col items-center gap-0.5">
                      {m.perfect_kills != null && m.perfect_kills > 0 ? (
                        <>
                          <span className="text-lg font-black leading-none" style={{ color: tokenCssVar('perf-tier-3') }}>{m.perfect_kills}</span>
                          <span className="text-[10px] font-medium leading-none" style={{ color: tokenCssVar('perf-tier-3') }}>Parfaits</span>
                        </>
                      ) : null}
                    </div>
                  </div>
                )}
              <div data-testid="match-card-damage-bar" className="px-3 pt-2.5 pb-2 space-y-1.5">
                <div className="h-2 w-full rounded-full overflow-hidden flex">
                  {offConv > 0 && <div className="h-full" style={{ width: damageTotal > 0 ? `${(offConv / damageTotal) * 100}%` : '0%', backgroundColor: tokenCssVar('outcome-win') }} />}
                  {defRes > 0 && <div className="h-full" style={{ width: damageTotal > 0 ? `${(defRes / damageTotal) * 100}%` : '0%', backgroundColor: tokenCssVar('perf-tier-2') }} />}
                </div>
                <div className="flex justify-center gap-5 mt-2">
                  <div className="flex flex-col items-center gap-0.5">
                    <span className="text-sm font-bold text-white leading-none">{offConv.toFixed(2)}</span>
                    <span className="text-[10px] font-medium leading-none" style={{ color: tokenCssVar('outcome-win') }}>Rendement</span>
                  </div>
                  <div className="flex flex-col items-center gap-0.5">
                    <span className="text-sm font-bold text-white leading-none">{defRes.toFixed(2)}</span>
                    <span className="text-[10px] font-medium leading-none" style={{ color: tokenCssVar('perf-tier-2') }}>Résistance</span>
                  </div>
                </div>
              </div>
            </>)}

            {/* Précision et vie moyenne */}
            {hasAccuracyLine && (
              <div data-testid="match-card-accuracy-line" className="px-3 pt-2.5 pb-2 flex justify-center gap-5">
                {m.accuracy != null && (
                  <div className="flex flex-col items-center gap-0.5">
                    <span className="text-sm font-bold text-white leading-none">{m.accuracy.toFixed(0)} %</span>
                    <span className="text-[10px] font-medium leading-none text-muted-foreground">Précision</span>
                  </div>
                )}
                {m.avg_life_secs != null && (
                  <div className="flex flex-col items-center gap-0.5">
                    <span className="text-sm font-bold text-white leading-none">{m.avg_life_secs.toFixed(1)} s</span>
                    <span className="text-[10px] font-medium leading-none text-muted-foreground">Vie moy.</span>
                  </div>
                )}
              </div>
            )}

            {/* MMR face-off : Équipe vs Adversaires */}
            {m.team_mmr != null && m.enemy_mmr != null && (
              <div data-testid="match-card-mmr" className="px-3 pt-2.5 pb-2 flex flex-col items-center gap-1">
                <div className="flex w-full items-center">
                  <div className="flex flex-1 flex-col items-center gap-0.5">
                    <span className="text-sm font-bold text-white leading-none">{Math.round(m.team_mmr)}</span>
                    <span className="text-[10px] font-medium leading-none text-muted-foreground">Équipe</span>
                  </div>
                  <div className="w-8 flex justify-center">
                    <span className="text-muted-foreground/50 text-[13px] leading-none">⟷</span>
                  </div>
                  <div className="flex flex-1 flex-col items-center gap-0.5">
                    <span className="text-sm font-bold text-white leading-none">{Math.round(m.enemy_mmr)}</span>
                    <span className="text-[10px] font-medium leading-none text-muted-foreground">Adversaires</span>
                  </div>
                </div>
                {m.delta_mmr != null && (
                  <span
                    className="rounded px-1.5 py-0.5 text-[9px] font-bold leading-none"
                    style={(() => {
                      const t = mmrDeltaScale(m.delta_mmr)
                      const c = tokenCssVar(t)
                      return {
                        backgroundColor: `color-mix(in srgb, ${c} 15%, transparent)`,
                        color: c,
                      }
                    })()}
                  >
                    {m.delta_mmr > 0 ? `+${Math.round(m.delta_mmr)}` : Math.round(m.delta_mmr)}
                  </span>
                )}
              </div>
            )}

            {/* Médailles — max 4, triées par count DESC */}
            {m.top_medals && m.top_medals.length > 0 && (
              <div
                data-testid="match-card-medals"
                className="px-3 pb-3 pt-2.5 flex justify-center gap-3 border-t border-white/[0.06] mt-3"
              >
                {m.top_medals.slice(0, 4).map((medal) => (
                  <div
                    key={medal.medal_id}
                    title={medal.description ?? medal.name ?? undefined}
                    className="flex flex-col items-center gap-0.5 cursor-default"
                  >
                    <img
                      src={medal.image_url}
                      alt={medal.name ?? String(medal.medal_id)}
                      className="w-8 h-8 object-contain"
                      onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
                    />
                    {medal.name && medal.name.trim() !== '' && (
                      <span className="text-[9px] text-muted-foreground/80 leading-tight text-center max-w-[40px] truncate">
                        {medal.name}
                      </span>
                    )}
                    <span className="text-[9px] font-semibold text-foreground/70 leading-none">
                      ×{medal.count}
                    </span>
                  </div>
                ))}
              </div>
            )}

            {/* Citations — max 3, filtrées et triées par delta DESC */}
            {m.top_citations && m.top_citations.length > 0 && (
              <div
                data-testid="match-card-citations"
                className="px-3 pb-3 pt-2.5 flex justify-center gap-3 border-t border-white/[0.06] mt-3"
              >
                {m.top_citations.map((cit) => (
                  <div
                    key={cit.key}
                    title={cit.description ?? cit.name}
                    className="flex flex-col items-center gap-0.5 cursor-default"
                  >
                    <CitationProgressRing
                      pct={cit.progress_pct}
                      imageUrl={cit.image_url ?? undefined}
                      isNewlyMastered={cit.is_newly_mastered}
                    />
                    {cit.name && (
                      <span className="text-[9px] text-muted-foreground/80 leading-tight text-center max-w-[40px] truncate">
                        {cit.name}
                      </span>
                    )}
                    <span className="text-[9px] font-semibold text-info leading-none">
                      +{cit.delta}
                    </span>
                    {cit.is_newly_mastered && (
                      <span className="text-[8px] font-bold text-warning leading-none">
                        Maîtrisé !
                      </span>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Footer : durée et date/heure du match — ancré en bas de la tuile */}
      {(m.duration_secs != null && m.duration_secs > 0 || m.started_at) && (
        <div
          data-testid="match-card-footer"
          className="border-t border-white/10 px-3 py-2 text-center bg-white/[0.02]"
        >
          {m.duration_secs != null && m.duration_secs > 0 && (
            <p className="text-[11px] text-muted-foreground leading-tight">
              <span className="font-semibold text-muted-foreground/90">Durée :</span>{' '}
              {formatMatchDuration(m.duration_secs)}
            </p>
          )}
          {m.started_at && (
            <p className="text-[11px] text-muted-foreground/70 leading-tight">
              <span className="font-semibold text-muted-foreground/90">Date :</span>{' '}
              {formatMatchDateTime(m.started_at, timezone, locale)}
            </p>
          )}
        </div>
      )}
    </div>
  )
}
