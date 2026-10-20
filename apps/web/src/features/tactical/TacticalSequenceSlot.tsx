/**
 * TacticalSequenceSlot — la ZONE « SÉQUENCE » du pied du plan : un bloc de hauteur minimale fixe, sous la
 * légende, qui montre TOUJOURS un état pour une contribution choisie (cf. `sequenceCause.logic.ts`) :
 *
 *   séquence servie   le lecteur — Rejouer / Pause / Reprendre, l'horloge du rejeu, et la barre des étapes
 *                     (« Instantané » puis une étape par seconde, de −4 s à +3 s) ;
 *   chargement        une ligne de statut ;
 *   analyse en cours  l'emplacement d'attente de l'app (`AnalysisPendingSlot`) ;
 *   séquence absente  un état vide qui dit pourquoi (`EmptyStateNotice` dans le bloc) ;
 *   autre qu'une mort une phrase : seules les morts ont une séquence.
 *
 * La hauteur minimale du bloc est la même dans tous les états : choisir une autre tuile ne fait pas
 * sauter le plan. La zone n'est pas rendue du tout quand la fonction est fermée (aucun état).
 */
import { useSyncExternalStore } from 'react'

import { AnalysisPendingSlot } from '@/components/ui/analysis-pending-slot'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { Locale } from '@/lib/i18n/locale'
import { formatClock } from '@/lib/replay/replayLogic'

import type { TacticalText } from './i18n'
import { libelleDeDecalage } from './mort.logic'
import type { EtatDeLaSequence } from './sequenceCause.logic'
import { horlogeAffichee } from './sequenceLecture.logic'
import type { LectureDeSequence } from './useLectureDeSequence'
import type { VueDeMort } from './vueDeMort'
import { decimale } from './zone.logic'

/** La hauteur minimale du bloc, quel que soit l'état : 6 rem. */
const BLOC = 'flex min-h-24 flex-col justify-center'

interface ZoneProps {
  t: TacticalText
  locale: Locale
  vue: VueDeMort | null
  etat: EtatDeLaSequence | null
}

/** ZoneSequence — le bloc, ou rien quand ni séquence servie ni état à dire. */
export function ZoneSequence({ t, locale, vue, etat }: ZoneProps) {
  if (vue?.sequence) {
    return (
      <section className={BLOC} data-testid="tactical-sequence-slot" data-etat="servie">
        <Lecteur t={t} locale={locale} vue={vue} />
      </section>
    )
  }
  if (!etat || etat.kind === 'servie') return null
  return (
    <section
      className={BLOC}
      data-testid="tactical-sequence-slot"
      data-etat={etat.kind === 'absente' ? etat.cause : etat.kind}
    >
      <EtatVide t={t} etat={etat} />
    </section>
  )
}

function EtatVide({ t, etat }: { t: TacticalText; etat: Exclude<EtatDeLaSequence, { kind: 'servie' }> }) {
  switch (etat.kind) {
    case 'chargement':
      return (
        <p role="status" className="m-0 text-center text-xs text-muted-foreground">
          {t.seqLoading}
        </p>
      )
    case 'sur_les_morts':
      return <EmptyStateNotice title={t.seqOnlyDeaths.title} description={t.seqOnlyDeaths.description} />
    case 'absente':
      if (etat.cause === 'analyse_en_cours') return <AnalysisPendingSlot label={t.seqPending} />
      return <EmptyStateNotice title={t.seqMissing[etat.cause].title} description={t.seqMissing[etat.cause].description} />
  }
}

const BOUTON =
  'min-h-8 rounded-md border px-1.5 py-1 font-mono text-[11px] tabular-nums transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring'
const ACTIF = 'border-foreground bg-foreground text-background'
const REPOS = 'border-border bg-card text-muted-foreground hover:bg-accent'

function Lecteur({ t, locale, vue }: { t: TacticalText; locale: Locale; vue: VueDeMort }) {
  return (
    <div className="flex flex-col gap-2">
      {vue.lecture && vue.sequence && <Commande t={t} locale={locale} lecture={vue.lecture} sequence={vue.sequence} />}
      <BarreDEtapes t={t} locale={locale} vue={vue} />
    </div>
  )
}

/** Commande — le bouton de lecture et l'horloge du rejeu à la tête de lecture. */
function Commande({
  t,
  locale,
  lecture,
  sequence,
}: {
  t: TacticalText
  locale: Locale
  lecture: LectureDeSequence
  sequence: NonNullable<VueDeMort['sequence']>
}) {
  const libelle = lecture.etat === 'lecture' ? t.seqPlayPause : lecture.etat === 'fin' ? t.seqPlayReplay : t.seqPlayResume
  return (
    <div className="flex items-center gap-3">
      <button
        type="button"
        onClick={lecture.basculer}
        className={`${BOUTON} ${lecture.etat === 'lecture' ? REPOS : ACTIF} min-w-[84px]`}
        data-testid="tactical-sequence-lecture"
        data-etat={lecture.etat}
      >
        {libelle}
      </button>
      <HorlogeEcrite t={t} locale={locale} lecture={lecture} sequence={sequence} />
    </div>
  )
}

/**
 * HorlogeEcrite — « À 7:20 » : l'horloge du rejeu à la tête de lecture. Elle s'abonne à l'horloge de
 * lecture mais n'en lit qu'un TEXTE : elle ne se repeint que quand la seconde affichée change.
 */
function HorlogeEcrite({
  t,
  locale,
  lecture,
  sequence,
}: {
  t: TacticalText
  locale: Locale
  lecture: LectureDeSequence
  sequence: NonNullable<VueDeMort['sequence']>
}) {
  const texte = useSyncExternalStore(lecture.horloge.abonner, () => {
    const offset = lecture.horloge.lire()
    const ms = horlogeAffichee(sequence, offset)
    return ms === null ? libelleDeDecalage(Math.round(offset / 100) * 100, locale) : formatClock(ms)
  })
  return (
    <span className="font-mono text-[12.5px] font-semibold tabular-nums" data-testid="tactical-sequence-horloge">
      {t.seqAt(texte)}
    </span>
  )
}

/** BarreDEtapes — « Instantané » puis une étape par seconde, de −4 s à +3 s ; un clic met en pause et saute à l'étape. */
function BarreDEtapes({ t, locale, vue }: { t: TacticalText; locale: Locale; vue: VueDeMort }) {
  const etapes = vue.sequence?.etapes ?? []
  return (
    <div role="group" aria-label={t.seqGroup} className="flex flex-wrap items-center gap-1" data-testid="tactical-sequence-etapes">
      <button
        type="button"
        aria-pressed={vue.etape === null}
        onClick={() => vue.onEtape(null)}
        className={`${BOUTON} ${vue.etape === null ? ACTIF : REPOS}`}
      >
        {t.seqSnapshot}
      </button>
      {etapes.map((e, i) => (
        <button
          key={e.offset_ms}
          type="button"
          aria-pressed={vue.etape === i}
          aria-label={t.seqStage(decimale(e.offset_ms / 1000, locale))}
          onClick={() => vue.onEtape(i)}
          className={`${BOUTON} ${vue.etape === i ? ACTIF : e.offset_ms === 0 ? 'border-foreground bg-card text-foreground' : REPOS}`}
          data-testid="tactical-sequence-etape"
        >
          {libelleDeDecalage(e.offset_ms, locale)}
        </button>
      ))}
    </div>
  )
}
