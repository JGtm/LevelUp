/**
 * HomeSkillPeakCard — carte de skill peak (CSR/LUSR) avec badge image.
 *
 * Évolution mai 2026 : la source de vérité de l'état placement est désormais
 * `peak.measurement_matches_remaining` (10 → 0). Le backend renvoie un peak
 * même en placement (avec rating_value=0 et badge_image_url pointant sur
 * unranked_(10-N).png). Le front ne devine plus via `hasHistory` + `mode`.
 */
import type { HomeSkillPeakSummary } from '@/lib/api/types'
import { unrankedBadgeURL } from '@/lib/staticAssets'
import { localizeTierLabel } from '@/lib/skillTiers'
import { KpiCard } from '@/components/cards/KpiCard'
import { CompositeProgressBar } from '@/components/ui/composite-progress-bar'

export interface HomeSkillPeakCardProps {
  label: string
  peak: HomeSkillPeakSummary | null
  numberLocale: string
  /** Locale UI ('fr'/'en') — localise le libellé de palier (tier_label baké). */
  locale: 'fr' | 'en'
  testIdPrefix: string
  state: 'value' | 'placement' | 'neutral' | 'absent'
  detail: string
  /**
   * Formate le libellé « Atteint le {date} » (i18n FR/EN). Optionnel : si absent
   * ou si peak.peak_achieved_at est nul, la date d'obtention n'est pas affichée
   * (dégradation gracieuse — ex. pic CSR all-time sans date sourçable).
   */
  reachedOnLabel?: (date: string) => string
}

export function HomeSkillPeakCard({
  label,
  peak,
  numberLocale,
  locale,
  testIdPrefix,
  state,
  detail,
  reachedOnLabel,
}: HomeSkillPeakCardProps) {
  const isPlacement = state === 'placement'
  const showRatingValue = state === 'value' && peak !== null && peak.rating_value > 0

  // Date d'obtention du pic, discrète, sous le rating. Uniquement en phase
  // « value » (matured) et si le backend a pu la sourcer (peak_achieved_at).
  const peakReachedText =
    state === 'value' && peak?.peak_achieved_at && reachedOnLabel
      ? reachedOnLabel(
          new Date(peak.peak_achieved_at).toLocaleDateString(numberLocale, {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
          }),
        )
      : null
  // Rang « par paliers » sans valeur numérique (CSR Halo 5 : tier seul, pas
  // d'échelle chiffrée) : on AFFICHE le palier (ligne principale) mais on NE
  // montre PAS le tiret « — » placeholder (qui n'a de sens qu'en placement /
  // non classé). Cf. issue user : ne pas afficher « — » quand le titre n'expose
  // pas de CSR chiffré.
  const isTierOnlyRating =
    state === 'value' && peak !== null && peak.rating_value <= 0 && !!peak.tier_label

  // Barre de progression à DROITE du rating. Progression ORDINALE via le
  // sous-palier (backend analysis.SkillTierBand) : Onyx → pleine, placement /
  // sans rang → absente (on n'affiche alors que le rating).
  const tierProgress =
    state === 'value' && peak != null && peak.tier_progress_pct != null
      ? peak.tier_progress_pct
      : null

  return (
    <div className="flex h-full flex-col gap-1.5">
      {/* Titre en en-tête de section, SORTI de la carte (cf. demande user :
          « comme un titre de section »). Identité CSR/LUSR par carte. */}
      <h3 className="text-sm font-semibold text-foreground">{label}</h3>
      <KpiCard
        testId={`${testIdPrefix}-card`}
        className="flex flex-1 flex-col justify-center px-4 py-3"
      >
        <div className="flex items-center gap-3">
          {peak?.badge_image_url ? (
            <img
              data-testid={isPlacement ? `${testIdPrefix}-unranked` : `${testIdPrefix}-badge`}
              src={peak.badge_image_url}
              alt={isPlacement ? 'En placement' : label}
              className={`h-12 w-12 shrink-0 object-contain ${isPlacement ? 'opacity-80' : ''}`}
              loading="lazy"
              decoding="async"
            />
          ) : isPlacement || state === 'absent' ? (
            // Legacy path : backend antérieur à mai 2026 ne fournissait pas le
            // badge unranked dans le peak. Fallback sur l'image statique générique.
            // État 'absent' (aucun historique) : même image unranked_0, label "Non classé".
            <img
              data-testid={`${testIdPrefix}-unranked`}
              src={unrankedBadgeURL()}
              alt={isPlacement ? 'En placement' : 'Non classé'}
              className="h-12 w-12 shrink-0 object-contain opacity-80"
              loading="lazy"
              decoding="async"
            />
          ) : (
            <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-border bg-muted text-2xs font-semibold uppercase tracking-label-xl text-muted-foreground">
              {label.replace(/[^A-Z]/gi, '').slice(0, 4) || 'MMR'}
            </div>
          )}

          {/* Gauche (extrémité) : LABEL de palier mis en avant (le user le connaît
              mieux que le chiffre) + rating en secondaire dessous. */}
          <div className="shrink-0">
            {/* Rang (palier) + valeur numérique sur la MÊME ligne : la valeur passe
                à DROITE du rang, alignée sur sa ligne de base (taille/couleur
                inchangées). Rang en text-xl (au lieu de 2xl). */}
            <div className="flex items-baseline gap-1.5">
              {(peak?.tier_label || detail) && (
                <p
                  data-testid={peak?.tier_label && !isPlacement ? `${testIdPrefix}-tier` : `${testIdPrefix}-detail`}
                  className="truncate text-xl font-semibold text-foreground"
                >
                  {isPlacement ? detail : (localizeTierLabel(peak?.tier_label, locale) ?? detail)}
                </p>
              )}
              {isTierOnlyRating ? null : (
                <p
                  data-testid={`${testIdPrefix}-value`}
                  className="text-xs font-medium text-muted-foreground"
                >
                  {showRatingValue ? peak!.rating_value.toLocaleString(numberLocale, { maximumFractionDigits: 0 }) : '—'}
                </p>
              )}
            </div>
            {peakReachedText && (
              <p
                data-testid={`${testIdPrefix}-reached-on`}
                className="mt-0.5 text-2xs font-medium text-muted-foreground/80"
              >
                {peakReachedText}
              </p>
            )}
          </div>

          {/* Barre CENTRÉE verticalement (items-center de la rangée) + sous-palier
              suivant à l'extrémité droite. */}
          {tierProgress != null && (
            <div className="flex min-w-0 flex-1 items-center gap-2">
              <div className="min-w-0 flex-1">
                <CompositeProgressBar value={tierProgress} fillTestId={`${testIdPrefix}-tier-progress-fill`} />
              </div>
              {peak!.next_tier_label && (
                <span
                  data-testid={`${testIdPrefix}-next-tier`}
                  className="shrink-0 truncate text-2xs font-medium text-muted-foreground"
                >
                  {localizeTierLabel(peak!.next_tier_label, locale)}
                </span>
              )}
            </div>
          )}
        </div>
      </KpiCard>
    </div>
  )
}

// resolveSkillPeakState est extrait dans ./skillPeakState (react-refresh :
// le module de composant n'exporte que des composants).
