/**
 * TacticalZoneBilan — la VUE DE LA ZONE (C), dans le bloc du bas de la colonne de droite : ce que les
 * morts de la zone disent des coéquipiers, sur deux rangées. Rangée 1 : « Coéquipier le plus proche » et
 * sa distance médiane, la barre fine des quatre statuts, les quatre comptes sur UNE ligne. Rangée 2 : la
 * rose des directions des tueurs et « N sur M par la droite · médiane D m ».
 *
 * Tous les comptes viennent du serveur (`bilan`), en « N sur M » : aucun taux nu, aucun champ « non
 * mesuré ». Une mesure absente (pas de médiane, pas de tueur rangé) n'est pas écrite.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { TacticalBilanMorts } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import {
  metresEntiers,
  ORDRE_DES_STATUTS,
  partsDeBarre,
  phraseDeLaRose,
  type PartDeBarre,
  type StatutDeMort,
} from './mort.logic'
import { TacticalRose } from './TacticalRose'

/** Le style d'un segment ou d'une pastille : le jeton allié plein ou atténué, ou une teinte neutre du thème. */
const SEGMENT: Record<StatutDeMort, { className: string; style?: { backgroundColor: string; opacity?: number } }> = {
  accompagnee: { className: '', style: { backgroundColor: tokenCssVar('team-ally') } },
  visible_hors_rayon: { className: '', style: { backgroundColor: tokenCssVar('team-ally'), opacity: 0.45 } },
  aucun_visible: { className: 'bg-muted-foreground' },
  equipe_a_terre: { className: 'bg-border' },
}

function BarreDesStatuts({ t, parts }: { t: TacticalText; parts: readonly PartDeBarre[] }) {
  return (
    <>
      <div className="flex h-1.5 flex-none gap-0.5 overflow-hidden rounded-full" role="img" aria-label={t.mortBilanTitle} data-testid="tactical-bilan-barre">
        {parts
          .filter((p) => p.n > 0)
          .map((p) => (
            <div
              key={p.statut}
              className={SEGMENT[p.statut].className}
              style={{ flex: p.n, ...SEGMENT[p.statut].style }}
              data-statut={p.statut}
            />
          ))}
      </div>
      <ul className="m-0 flex list-none flex-wrap gap-x-3 gap-y-0.5 p-0 text-[11px] text-muted-foreground" data-testid="tactical-bilan-statuts">
        {ORDRE_DES_STATUTS.map((s) => {
          const p = parts.find((q) => q.statut === s)
          return (
            <li key={s} className="flex items-center gap-1 whitespace-nowrap">
              <span aria-hidden className={`inline-block size-1.5 rounded-full ${SEGMENT[s].className}`} style={SEGMENT[s].style} />
              <b className="font-medium tabular-nums text-foreground">{p?.n ?? 0}</b> {t.mortStatusLabels[s]}
            </li>
          )
        })}
      </ul>
    </>
  )
}

function RangeeDuPlusProche({ t, locale, b }: { t: TacticalText; locale: Locale; b: TacticalBilanMorts }) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-[12.5px] font-semibold">{t.mortCardNearest}</span>
        {b.mediane_plus_proche_m != null && (
          <span className="text-[11.5px] text-muted-foreground">
            {t.mortCardMedianLabel} <span className="font-mono tabular-nums text-foreground">{t.mortCardMedian(metresEntiers(b.mediane_plus_proche_m, locale))}</span>
          </span>
        )}
      </div>
      {b.examinees > 0 && <BarreDesStatuts t={t} parts={partsDeBarre(b)} />}
    </div>
  )
}

function RangeeDeLaRose({ t, locale, b }: { t: TacticalText; locale: Locale; b: TacticalBilanMorts }) {
  const phrase = phraseDeLaRose(t, b)
  if (!phrase) return null
  return (
    <div className="flex items-center gap-3 border-t border-border pt-2.5">
      <TacticalRose rose={b.rose ?? []} label={t.mortRoseLabel} />
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="text-[12.5px] font-semibold">{t.mortRoseTitle}</span>
        <span className="text-[11.5px] text-muted-foreground">
          {phrase}
          {b.mediane_tueur_m != null && <> · {t.mortRoseMedian(metresEntiers(b.mediane_tueur_m, locale))}</>}
        </span>
      </div>
    </div>
  )
}

interface TacticalZoneBilanProps {
  t: TacticalText
  locale: Locale
  morts: TacticalBilanMorts
}

/** TacticalZoneBilan — le bilan des morts de la zone (lectures morts et isole). */
export function TacticalZoneBilan({ t, locale, morts }: TacticalZoneBilanProps) {
  return (
    <section className="flex flex-col gap-2.5" aria-label={t.mortZoneSection} data-testid="tactical-zone-bilan">
      <RangeeDuPlusProche t={t} locale={locale} b={morts} />
      <RangeeDeLaRose t={t} locale={locale} b={morts} />
    </section>
  )
}
