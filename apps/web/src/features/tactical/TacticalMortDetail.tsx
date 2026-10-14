/**
 * TacticalMortDetail — l'encadré « Instant de la mort · date · instant » du bas de la colonne de droite,
 * ouvert par la SÉLECTION d'une tuile (désélection = retour à la vue de la zone). Il porte le texte de la
 * vue que le plan peint :
 *
 *   instantané (A, B)  le tueur, les coéquipiers (● dans le rayon, ○ au-delà, hors de vue, en attente)
 *                      avec leur distance à la mort et au tueur, le résumé de l'équipe ; pour un frag,
 *                      la victime et l'échange ;
 *   étape (D)          l'horloge du rejeu à l'étape, le compte de coéquipiers dans le rayon, la distance
 *                      de chaque acteur au centre du cercle.
 *
 * Le bouton de rejeu ouvre le rejeu à l'instant par le MÊME chemin que la tuile (`modeleDeTuile` : mêmes
 * `search`, même libellé, même recul avant l'événement).
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { MatchReplayLink } from '@/lib/match-nav/MatchReplayLink'
import { formatClock } from '@/lib/replay/replayLogic'
import { useAppShellStore } from '@/stores/appShellStore'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import {
  distancesDEtape,
  echangeDeLaMort,
  libelleDeDecalage,
  lignesDeLaFiche,
  metresEntiers,
  resumeDeLEquipe,
  type LigneDeFiche,
} from './mort.logic'
import type { VueDeMort } from './vueDeMort'
import { modeleDeTuile } from './zone.logic'

const COLONNES = 'grid grid-cols-[minmax(0,1fr)_64px_64px] items-center gap-x-2 gap-y-1.5'

function Marque({ marque, role }: { marque: LigneDeFiche['marque']; role: LigneDeFiche['role'] }) {
  if (role === 'tueur') {
    return <span aria-hidden className="mr-1.5 inline-block size-2 rotate-45" style={{ backgroundColor: tokenCssVar('team-enemy') }} data-forme="losange" />
  }
  if (marque === 'plein') {
    return <span aria-hidden className="mr-1.5 inline-block size-2.5 rounded-full" style={{ backgroundColor: tokenCssVar('team-ally') }} data-forme="plein" />
  }
  if (marque === 'creux') {
    return <span aria-hidden className="mr-1.5 inline-block size-2.5 rounded-full border-2" style={{ borderColor: tokenCssVar('team-ally') }} data-forme="creux" />
  }
  return null
}

function Distance({ valeur }: { valeur: string | null }) {
  return <span className="text-right font-mono tabular-nums">{valeur === null ? '—' : `${valeur} m`}</span>
}

function FicheDeLInstantane({ t, locale, vue }: { t: TacticalText; locale: Locale; vue: VueDeMort }) {
  const c = vue.contribution
  const lignes = lignesDeLaFiche(t, c, locale)
  const resume = resumeDeLEquipe(t, c, locale)
  const echange = c.face === 'frag' ? echangeDeLaMort(t, c, locale) : null
  if (c.face === 'frag') {
    return (
      <div className="flex flex-col gap-1.5 text-xs" data-testid="tactical-mort-fiche">
        {c.autre_gamertag && (
          <div className="flex justify-between gap-2">
            <span>{t.mortDetailVictim} · {c.autre_gamertag}</span>
          </div>
        )}
        {echange && (
          <p className="m-0 text-muted-foreground">
            <b className="font-semibold text-foreground">{echange.titre}</b>
            {echange.detail ? ` · ${echange.detail}` : ''}
          </p>
        )}
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-2 text-xs" data-testid="tactical-mort-fiche">
      <div className={COLONNES}>
        <span className="text-[11px] text-muted-foreground">{t.mortDetailPlayer}</span>
        <span className="text-right text-[11px] text-muted-foreground">{t.mortDetailFromDeath}</span>
        <span className="text-right text-[11px] text-muted-foreground">{t.mortDetailFromKiller}</span>
        {lignes.map((l) => (
          <LigneDeLaFiche key={l.cle} l={l} />
        ))}
      </div>
      {resume && <p className="m-0 border-t border-border pt-2 text-muted-foreground">{resume}</p>}
    </div>
  )
}

function LigneDeLaFiche({ l }: { l: LigneDeFiche }) {
  const sansDistance = l.marque === 'aucune' && l.role === 'coequipier'
  return (
    <>
      <span className={`min-w-0 truncate ${sansDistance ? 'text-muted-foreground' : ''}`} title={l.texte}>
        <Marque marque={l.marque} role={l.role} />
        {l.texte}
      </span>
      {sansDistance ? (
        <>
          <span />
          <span />
        </>
      ) : (
        <>
          <Distance valeur={l.distanceMort} />
          <Distance valeur={l.distanceTueur} />
        </>
      )}
    </>
  )
}

function FicheDEtape({ t, locale, vue }: { t: TacticalText; locale: Locale; vue: VueDeMort }) {
  const seq = vue.sequence
  const index = vue.etape
  if (!seq || index === null) return null
  const etape = seq.etapes?.[index]
  if (!etape) return null
  const horloge = etape.instant_affiche_ms == null ? libelleDeDecalage(etape.offset_ms, locale) : formatClock(etape.instant_affiche_ms)
  const lignes = distancesDEtape(t, seq, index, locale)
  return (
    <div className="flex flex-col gap-2 text-xs" data-testid="tactical-etape-fiche">
      <div className="flex items-baseline justify-between gap-2">
        <b className="text-[13px] font-semibold">{t.seqAt(horloge)}</b>
        <span className="text-muted-foreground">
          {t.seqSummary(etape.dans_le_rayon, etape.positionnes, metresEntiers(seq.rayon_m, locale))}
        </span>
      </div>
      <ul className="m-0 flex list-none flex-col gap-1 p-0">
        {lignes.map((l) => (
          <li key={l.cle} className="flex justify-between gap-2">
            <span className="min-w-0 truncate">
              <Marque marque={l.role === 'coequipier' ? 'plein' : 'aucune'} role={l.role} />
              {l.nom}
            </span>
            <Distance valeur={l.distance} />
          </li>
        ))}
      </ul>
      <p className="m-0 border-t border-border pt-2 text-[11px] text-muted-foreground">{t.seqDistanceNote}</p>
    </div>
  )
}

interface TacticalMortDetailProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  vue: VueDeMort
}

export function TacticalMortDetail({ t, locale, playerSlug, vue }: TacticalMortDetailProps) {
  const timezone = useAppShellStore((s) => s.userTimezone)
  const c = vue.contribution
  const m = modeleDeTuile(t, c, locale, timezone, null)
  const frag = c.face === 'frag'
  const titre = t.mortDetailTitle(m.date, m.instant, frag)
  return (
    <section
      className="flex flex-none flex-col gap-2.5 rounded-lg border border-input bg-card p-3"
      aria-label={titre}
      data-testid="tactical-mort-detail"
    >
      <div className="flex items-center justify-between gap-2">
        <h4 className="m-0 min-w-0 truncate text-[13px] font-semibold" title={titre}>
          {titre}
        </h4>
        <MatchReplayLink
          available={m.rejeu.disponible}
          matchId={c.match_id}
          playerSlug={playerSlug}
          label={m.rejeu.libelle}
          variant="button"
          search={m.rejeu.search}
        />
      </div>
      {vue.etape !== null && vue.sequence ? <FicheDEtape t={t} locale={locale} vue={vue} /> : <FicheDeLInstantane t={t} locale={locale} vue={vue} />}
    </section>
  )
}
