/**
 * TacticalMortDetail — le contenu du bloc du bas de la colonne de droite quand une tuile est choisie
 * (désélection = retour au bilan de la zone) : « Mort · date · instant » et le bouton de rejeu, puis le
 * texte de la vue que le plan peint :
 *
 *   instantané  le tueur et les coéquipiers (plein dans le rayon, creux au-delà, hors de vue, en
 *               attente) avec leur distance à la mort et au tueur, le résumé de l'équipe en pied ; pour
 *               un frag, la victime ;
 *   étape (D)   l'horloge du rejeu à l'étape, le compte de coéquipiers dans le rayon, la distance de
 *               chaque acteur au centre du cercle.
 *
 * Le bouton de rejeu ouvre le rejeu à l'instant par le MÊME chemin que la tuile (`modeleDeTuile` : mêmes
 * `search`, même libellé, même recul avant l'événement). Le bloc lui-même (hauteur fixe) est posé par
 * `TacticalZoneCard`.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { MatchReplayLink } from '@/lib/match-nav/MatchReplayLink'
import { formatClock } from '@/lib/replay/replayLogic'
import { useAppShellStore } from '@/stores/appShellStore'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import {
  distancesDEtape,
  libelleDeDecalage,
  lignesDeLaFiche,
  metresEntiers,
  resumeDeLEquipe,
  type LigneDeFiche,
} from './mort.logic'
import type { VueDeMort } from './vueDeMort'
import { modeleDeTuile } from './zone.logic'

const COLONNES = 'grid grid-cols-[minmax(0,1fr)_64px_64px] items-center gap-x-2 gap-y-1 text-[11.5px]'

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

/** Le contenu d'une fiche : le corps (défile dans le bloc) et le pied (toujours visible). */
interface Fiche {
  corps: React.ReactNode
  pied: React.ReactNode
}

function ficheDeLInstantane({ t, locale, vue }: { t: TacticalText; locale: Locale; vue: VueDeMort }): Fiche {
  const c = vue.contribution
  if (c.face === 'frag') {
    return {
      corps: c.autre_gamertag ? (
        <div className="text-xs" data-testid="tactical-mort-fiche">
          {t.mortDetailVictim} · {c.autre_gamertag}
        </div>
      ) : (
        <div data-testid="tactical-mort-fiche" />
      ),
      pied: null,
    }
  }
  const lignes = lignesDeLaFiche(t, c, locale)
  const resume = resumeDeLEquipe(t, c, locale)
  return {
    corps: (
      <div className={COLONNES} data-testid="tactical-mort-fiche">
        <span className="text-[10.5px] text-muted-foreground">{t.mortDetailPlayer}</span>
        <span className="text-right text-[10.5px] text-muted-foreground">{t.mortDetailFromDeath}</span>
        <span className="text-right text-[10.5px] text-muted-foreground">{t.mortDetailFromKiller}</span>
        {lignes.map((l) => (
          <LigneDeLaFiche key={l.cle} l={l} />
        ))}
      </div>
    ),
    pied: resume,
  }
}

function LigneDeLaFiche({ l }: { l: LigneDeFiche }) {
  const sansDistance = l.marque === 'aucune' && l.role === 'coequipier'
  return (
    <>
      <span className={sansDistance ? 'col-span-3 text-muted-foreground' : 'min-w-0 truncate'} title={l.texte}>
        <Marque marque={l.marque} role={l.role} />
        {l.texte}
      </span>
      {!sansDistance && (
        <>
          <Distance valeur={l.distanceMort} />
          <Distance valeur={l.distanceTueur} />
        </>
      )}
    </>
  )
}


function ficheDEtape({ t, locale, vue }: { t: TacticalText; locale: Locale; vue: VueDeMort }): Fiche | null {
  const seq = vue.sequence
  const index = vue.etape
  if (!seq || index === null) return null
  const etape = seq.etapes?.[index]
  if (!etape) return null
  const horloge = etape.instant_affiche_ms == null ? libelleDeDecalage(etape.offset_ms, locale) : formatClock(etape.instant_affiche_ms)
  const lignes = distancesDEtape(t, seq, index, locale)
  return {
    corps: (
      <div className="flex flex-col gap-1.5 text-xs" data-testid="tactical-etape-fiche">
        <div className="flex items-baseline justify-between gap-2">
          <b className="text-[12.5px] font-semibold">{t.seqAt(horloge)}</b>
          <span className="text-muted-foreground">
            {t.seqSummary(etape.dans_le_rayon, etape.positionnes, metresEntiers(seq.rayon_m, locale))}
          </span>
        </div>
        <ul className="m-0 flex list-none flex-col gap-1 p-0">
          {lignes.map((l) => (
            <li key={l.cle} className="flex justify-between gap-2">
              <span className="min-w-0 truncate">
                <Marque marque={l.marque} role={l.role} />
                {l.nom}
              </span>
              <Distance valeur={l.distance} />
            </li>
          ))}
        </ul>
      </div>
    ),
    pied: t.seqDistanceNote,
  }
}

interface TacticalMortDetailProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  vue: VueDeMort
}

/**
 * TacticalMortDetail — le CONTENU du bloc du bas quand une tuile est choisie : l'en-tête (« Mort · date
 * · instant » et le bouton de rejeu sans bordure), le corps (qui défile dans le bloc) et le pied. Le
 * bloc (hauteur fixe, carte séparée) est posé par `TacticalZoneCard`.
 */
export function TacticalMortDetail({ t, locale, playerSlug, vue }: TacticalMortDetailProps) {
  const timezone = useAppShellStore((s) => s.userTimezone)
  const c = vue.contribution
  const m = modeleDeTuile(t, c, locale, timezone, null)
  const titre = t.mortDetailTitle(m.date, m.instant, c.face === 'frag')
  const fiche = (vue.etape !== null && vue.sequence ? ficheDEtape({ t, locale, vue }) : null) ?? ficheDeLInstantane({ t, locale, vue })
  return (
    <section className="flex h-full min-h-0 flex-col gap-2" aria-label={titre} data-testid="tactical-mort-detail">
      <div className="flex flex-none items-center justify-between gap-2">
        <h4 className="m-0 min-w-0 truncate text-[12.5px] font-semibold" title={titre}>
          {titre}
        </h4>
        <MatchReplayLink
          available={m.rejeu.disponible}
          matchId={c.match_id}
          playerSlug={playerSlug}
          label={m.rejeu.libelle}
          variant="large"
          search={m.rejeu.search}
        />
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto">{fiche.corps}</div>
      {fiche.pied && <p className="m-0 flex-none border-t border-border pt-2 text-[11.5px] text-muted-foreground">{fiche.pied}</p>}
    </section>
  )
}
