/**
 * TacticalCoordinationCard — la section « Coordination d'équipe » de la vue d'analyse
 * (maquette 034b1915, portée le 2026-09-13).
 *
 * TROIS CHIFFRES ET UNE FORME. Les deux taux (échange après ma mort, morts en isolement)
 * répondent « combien » ; la distance médiane et l'histogramme répondent « de combien ».
 * Deux joueurs à 63 % d'isolement dont l'un meurt à 26 m de son équipier et l'autre à 70 m
 * ne jouent pas le même jeu — et le taux seul ne le dit pas.
 *
 * TOUT VIENT DU SERVEUR, Y COMPRIS LES INTERVALLES (binning serveur, ADR 0010) : cette
 * carte ne calcule ni médiane, ni bucket, ni seuil. Elle dessine ce qu'elle reçoit.
 *
 * LE SEUIL D'ISOLEMENT EST LA PORTÉE DU RADAR EN JEU, MESURÉE PAR VARIANTE
 * (`config/titles/{slug}/mappings/regulation.toml` : 18 m en Arène, 24 m en BTB) — pas le
 * « 25 m » de la maquette, qui était un chiffre de maquette. Un filtre qui mélange les deux
 * formats mélange DEUX RÈGLES DU JEU : les deux seuils sont alors tracés, jamais leur
 * moyenne, qui ne serait la règle d'aucun match.
 *
 * LES MORTS SANS COÉQUIPIER VISIBLE N'ONT PAS DE DISTANCE. Elles comptent dans le taux
 * d'isolement (elles sont isolées par construction) mais pas dans l'histogramme : les
 * verser dans « 50+ » inventerait une mesure. Leur nombre est dit en pied de carte.
 */
import { useMemo } from 'react'

import { HistogramChart, type ChartPointHistogram } from '@/components/charts/HistogramChart'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionCard } from '@/components/ui/section-card'
import type { TacticalCoordination, TacticalCouverture } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import { libelleRayons, positionCategorie } from './tacticalView.logic'

export interface TacticalCoordinationCardProps {
  t: TacticalText
  locale: Locale
  coordination: TacticalCoordination
  /** Taux d'échange — servi par la lecture, `null` quand le titre ne sait pas le mesurer. */
  echange: TacticalCouverture | null
  /** Taux d'isolement — servi par la lecture, `null` quand aucune mort n'a pu être lue. */
  isolement: TacticalCouverture | null
  matchsFiltres: number
}

export function TacticalCoordinationCard({
  t,
  locale,
  coordination,
  echange,
  isolement,
  matchsFiltres,
}: TacticalCoordinationCardProps) {
  const pct = new Intl.NumberFormat(intlLocale(locale), {
    style: 'percent',
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  })
  // Mémoïsés : ce sont les dépendances des deux `useMemo` ci-dessous, et un `?? []` rend
  // un tableau NEUF à chaque rendu — les séries du graphe seraient alors reconstruites en
  // boucle (et le lint anti-cascade le signale).
  const rayons = useMemo(() => coordination.rayons_m ?? [], [coordination.rayons_m])
  const bins = useMemo(
    () => coordination.distribution_distances ?? [],
    [coordination.distribution_distances],
  )
  const rayonTexte = libelleRayons(t, rayons)

  const series = useMemo(
    () => [
      {
        key: 'distance-equipier',
        name: t.coordinationChartTitle,
        datapoints: bins.map(
          (b): ChartPointHistogram => ({
            binStart: b.min_m,
            // La borne haute du DERNIER intervalle est ouverte (`max_m` absent) : on
            // l'étiquette « 50+ » (`formatBin` ci-dessous), la valeur ici ne sert qu'à
            // garder un point bien formé.
            binEnd: b.max_m ?? b.min_m,
            count: b.n,
          }),
        ),
      },
    ],
    [bins, t.coordinationChartTitle],
  )

  // Les seuils tombent à leur VRAIE position sur l'axe des intervalles : 18 m sur des
  // intervalles de 10 m = 1,8 catégorie, entre la deuxième et la troisième barre.
  const seuils = useMemo(
    () =>
      rayons
        .map((rayon) => ({ at: positionCategorie(rayon, bins), label: t.radiusValue(rayon) }))
        .filter((s) => s.at !== null) as { at: number; label: string }[],
    [rayons, bins, t],
  )
  // Une barre est APPUYÉE quand elle est AU-DELÀ du plus petit seuil : c'est la zone
  // d'isolement, celle dont le taux parle. Les autres restent atténuées (maquette).
  const seuilBas = rayons.length > 0 ? Math.min(...rayons) : null

  const aucuneMesure = coordination.n_distances === 0 && !isolement && !echange

  return (
    <SectionCard title={t.coordinationTitle} label={t.coordinationTitle}>
      <div className="flex flex-col gap-3 p-3" data-testid="tactical-coordination">
        {aucuneMesure ? (
          <EmptyStateNotice
            title={t.coordinationEmpty}
            description={t.coordinationEmptyDescription}
          />
        ) : (
          <>
            <dl className="flex flex-col">
              {echange && (
                <Ligne
                  label={t.kpiTrade}
                  value={pct.format(echange.taux)}
                  testid="tactical-coordination-trade"
                />
              )}
              {isolement && (
                <Ligne
                  label={t.kpiIsolation}
                  value={pct.format(isolement.taux)}
                  testid="tactical-coordination-isolation"
                />
              )}
              {coordination.distance_mediane_m != null && (
                <Ligne
                  label={t.coordinationMedian}
                  value={t.radiusValue(coordination.distance_mediane_m)}
                  testid="tactical-coordination-median"
                />
              )}
            </dl>
            {coordination.n_distances > 0 && (
              <HistogramChart
                title={t.coordinationChartTitle}
                series={series}
                height={180}
                colorToken="chart-series-2"
                yAxisLabel={t.coordinationChartY}
                formatBin={(point) =>
                  point.binEnd > point.binStart
                    ? t.coordinationBucket(point.binStart, point.binEnd)
                    : t.coordinationBucketLast(point.binStart)
                }
                binAttenuated={(point) => seuilBas !== null && point.binStart < seuilBas}
                thresholds={seuils}
              />
            )}
            <div className="flex flex-col gap-1 border-t border-border pt-2 text-2xs text-muted-foreground">
              <p data-testid="tactical-coordination-rules">
                {t.coordinationNoteRules(coordination.fenetre_echange_secondes, rayonTexte)}
              </p>
              <p>{t.coordinationNoteCoverage(coordination.matchs_mesures, matchsFiltres)}</p>
              {coordination.morts_sans_distance > 0 && (
                <p data-testid="tactical-coordination-no-distance">
                  {t.coordinationNoDistance(coordination.morts_sans_distance)}
                </p>
              )}
            </div>
          </>
        )}
      </div>
    </SectionCard>
  )
}

function Ligne({ label, value, testid }: { label: string; value: string; testid: string }) {
  return (
    <div className="flex items-baseline gap-2 border-b border-border py-1 text-xs last:border-b-0">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="ml-auto font-mono tabular-nums text-foreground" data-testid={testid}>
        {value}
      </dd>
    </div>
  )
}

