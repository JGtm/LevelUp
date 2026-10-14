/**
 * TacticalZoneBilan — la VUE DE LA ZONE (C) : ce que les morts de la zone disent des coéquipiers, au-dessus
 * de la liste des tuiles. Barre des quatre statuts du coéquipier le plus proche, rose des directions des
 * tueurs, « Tueur abattu en N s : X sur Y » et distance médiane au coéquipier le plus proche. Pour les
 * lectures de frags, le miroir : « Abattu en retour en N s : X sur Y ».
 *
 * Tous les comptes viennent du serveur (`bilan`), en « N sur M » : aucun taux nu, aucun champ « non
 * mesuré ». Une mesure absente (pas de médiane, pas de tueur rangé) n'est pas écrite. La fenêtre de la
 * riposte est celle que le serveur publie (`fenetre_riposte_ms`), jamais recopiée ici.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { TacticalBilanFrags, TacticalBilanMorts } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import {
  metresEntiers,
  ORDRE_DES_STATUTS,
  partsDeBarre,
  phraseDeLaRose,
  secondesDeFenetre,
  type PartDeBarre,
  type StatutDeMort,
} from './mort.logic'
import { TacticalRose } from './TacticalRose'

/** Le style d'un segment de la barre : le jeton allié plein ou atténué, ou une teinte neutre du thème. */
const SEGMENT: Record<StatutDeMort, { className: string; style?: { backgroundColor: string; opacity?: number } }> = {
  accompagnee: { className: '', style: { backgroundColor: tokenCssVar('team-ally') } },
  visible_hors_rayon: { className: '', style: { backgroundColor: tokenCssVar('team-ally'), opacity: 0.45 } },
  aucun_visible: { className: 'bg-muted-foreground' },
  equipe_a_terre: { className: 'bg-border' },
}

function BarreDesStatuts({ t, parts }: { t: TacticalText; parts: readonly PartDeBarre[] }) {
  return (
    <>
      <div className="flex h-3.5 gap-0.5 overflow-hidden rounded" role="img" aria-label={t.mortBilanTitle} data-testid="tactical-bilan-barre">
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
      <ul className="grid grid-cols-2 gap-x-3 gap-y-0.5 text-xs text-muted-foreground" data-testid="tactical-bilan-statuts">
        {ORDRE_DES_STATUTS.map((s) => {
          const p = parts.find((q) => q.statut === s)
          return (
            <li key={s} className="flex items-center gap-1.5">
              <span
                aria-hidden
                className={`inline-block size-2 rounded-sm ${SEGMENT[s].className}`}
                style={SEGMENT[s].style}
              />
              {t.mortStatusLabels[s]} · <b className="font-medium tabular-nums text-foreground">{p?.n ?? 0}</b>
            </li>
          )
        })}
      </ul>
    </>
  )
}

function Chiffre({ titre, valeur, suite }: { titre: string; valeur: string; suite: string }) {
  return (
    <div className="flex flex-col rounded-md bg-muted px-2.5 py-1.5" data-testid="tactical-bilan-chiffre">
      <span className="text-[11px] text-muted-foreground">{titre}</span>
      <span className="font-mono text-base font-semibold leading-5 tabular-nums">
        {valeur} <span className="text-xs font-normal text-muted-foreground">{suite}</span>
      </span>
    </div>
  )
}

function BilanDesMorts({ t, locale, b }: { t: TacticalText; locale: Locale; b: TacticalBilanMorts }) {
  const phrase = phraseDeLaRose(t, b)
  const secondes = secondesDeFenetre(b.fenetre_riposte_ms, locale)
  return (
    <section className="flex flex-col gap-2.5" aria-label={t.mortZoneSection} data-testid="tactical-zone-bilan">
      <h4 className="text-xs font-semibold">{t.mortBilanTitle}</h4>
      {b.examinees > 0 && <BarreDesStatuts t={t} parts={partsDeBarre(b)} />}
      {phrase && (
        <div className="flex items-center gap-3">
          <TacticalRose rose={b.rose ?? []} label={t.mortRoseLabel} />
          <div className="flex min-w-0 flex-col gap-1 text-xs">
            <b className="font-semibold">{t.mortRoseTitle}</b>
            <span className="text-muted-foreground">{phrase}</span>
            {b.mediane_tueur_m != null && (
              <span className="text-muted-foreground">{t.mortRoseMedian(metresEntiers(b.mediane_tueur_m, locale))}</span>
            )}
          </div>
        </div>
      )}
      <div className="grid grid-cols-2 gap-2">
        {b.tueur_abattu.m > 0 && (
          <Chiffre titre={t.mortCardAvenged(secondes)} valeur={String(b.tueur_abattu.n)} suite={t.mortCardOf(b.tueur_abattu.m)} />
        )}
        {b.mediane_plus_proche_m != null && (
          <Chiffre
            titre={t.mortCardNearest}
            valeur={t.mortCardMedian(metresEntiers(b.mediane_plus_proche_m, locale))}
            suite={t.mortCardMedianLabel}
          />
        )}
      </div>
    </section>
  )
}

interface TacticalZoneBilanProps {
  t: TacticalText
  locale: Locale
  morts?: TacticalBilanMorts
  frags?: TacticalBilanFrags
}

/** TacticalZoneBilan — le bilan des morts (lectures morts et isole) et/ou le miroir des frags (kills, solde). */
export function TacticalZoneBilan({ t, locale, morts, frags }: TacticalZoneBilanProps) {
  if (!morts && !frags) return null
  return (
    <div className="flex flex-none flex-col gap-3">
      {morts && <BilanDesMorts t={t} locale={locale} b={morts} />}
      {frags && frags.abattu_en_retour.m > 0 && (
        <div className="grid grid-cols-2 gap-2" data-testid="tactical-bilan-frags">
          <Chiffre
            titre={t.mortCardReturned(secondesDeFenetre(frags.fenetre_riposte_ms, locale))}
            valeur={String(frags.abattu_en_retour.n)}
            suite={t.mortCardOf(frags.abattu_en_retour.m)}
          />
        </div>
      )}
    </div>
  )
}
