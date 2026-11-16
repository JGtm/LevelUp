/**
 * TacticalReapparitionsColonne — la colonne de droite de la lecture « Réapparitions adverses » : l'en-tête de
 * la zone tenue (nom, matchs, réapparitions) puis un graphique en barres comparées — barre épaisse = part de
 * chaque zone PENDANT la tenue, barre fine = part SANS condition — avec son titre neutre, son aide ⓘ et sa
 * légende en bas.
 *
 * ÉTAT VIDE : un bloc pointillé dit ce qui manque (jauges de matchs et de réapparitions contre les planchers
 * du serveur, ou la cause nommée), et la répartition sans condition reste sous lui « pour mémoire », en
 * barres fines seules. Aucune action impossible n'est proposée.
 */
import { SectionCard } from '@/components/ui/section-card'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import {
  blocVideDeLaLecture,
  formaterLaPart,
  type BlocVide,
  type JaugeDuBloc,
  type LectureReapparitions,
} from './reapparitions.logic'

export interface TacticalReapparitionsColonneProps {
  t: TacticalText
  locale: Locale
  /** La lecture ; `null` tant que la réponse n'est pas là. */
  modele: LectureReapparitions | null
  /** La requête est en cours. */
  enAttente: boolean
  /** La requête a échoué (refus du serveur ou panne). */
  enErreur: boolean
  /** La carte n'a aucune zone nommée qu'on puisse tenir : il n'y a rien à lire. */
  sansZones: boolean
  estompe: string
}

export function TacticalReapparitionsColonne({
  t,
  locale,
  modele,
  enAttente,
  enErreur,
  sansZones,
  estompe,
}: TacticalReapparitionsColonneProps) {
  return (
    <div className={`min-h-0 lg:self-stretch lg:[contain:size]${estompe}`} data-testid="tactical-reapp-colonne">
      <SectionCard
        title={t.reappHeldCaption}
        label={t.reappHeldCaption}
        className="h-full"
        titleAdornment={() => <EnTete t={t} modele={modele} />}
      >
        <div className="min-h-0 flex-1 overflow-y-auto p-3">
          <Corps t={t} locale={locale} modele={modele} enAttente={enAttente} enErreur={enErreur} sansZones={sansZones} />
        </div>
      </SectionCard>
    </div>
  )
}

/** En-tête : la légende « Zone tenue », le nom de la zone et ses effectifs. */
function EnTete({ t, modele }: { t: TacticalText; modele: LectureReapparitions | null }) {
  const zone = modele?.zone ?? null
  return (
    <span className="flex flex-col gap-0.5">
      <small className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground">{t.reappHeldCaption}</small>
      <span className="flex items-baseline gap-2">
        <b className="min-w-0 truncate text-base font-semibold" data-testid="tactical-reapp-zone-nom">
          {zone ? zone.nom : t.reappHeldNone}
        </b>
        {zone && modele && (
          <span className="ml-auto whitespace-nowrap text-[11.5px] font-normal text-muted-foreground">
            {t.reappHeldCounts(modele.tenue.matchs, modele.tenue.reapparitions)}
          </span>
        )}
      </span>
    </span>
  )
}

function Corps({
  t,
  locale,
  modele,
  enAttente,
  enErreur,
  sansZones,
}: Omit<TacticalReapparitionsColonneProps, 'estompe'>) {
  if (sansZones) return <BlocPointille bloc={{ titre: t.reappEmptyNoZonesTitle, corps: t.reappEmptyNoZonesBody, jauges: [] }} t={t} />
  if (enErreur) return <p role="alert" className="text-sm text-muted-foreground">{t.reappError}</p>
  if (!modele) return enAttente ? <p role="status" className="text-sm text-muted-foreground">{t.reappLoading}</p> : null
  const bloc = blocVideDeLaLecture(t, modele)
  const titre = modele.zone ? t.reappChartTitle(modele.zone.nom) : t.reappChartTitleReference
  const info = t.reappChartInfo(modele.plancher.joueursPourTenir, modele.reference.reapparitions, modele.reference.matchs, {
    floorMatchs: modele.plancher.matchs,
    floorN: modele.plancher.reapparitions,
  })
  return (
    <>
      <h4 className="mb-2.5 text-[12.5px] font-semibold">{titleWithInfo(info)(titre)}</h4>
      {bloc && <BlocPointille bloc={bloc} t={t} />}
      {bloc && modele.lignes.length > 0 && (
        <p className="mb-2 mt-3.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">{t.reappSubReference}</p>
      )}
      {modele.lignes.length > 0 && <Barres locale={locale} modele={modele} />}
      {modele.lignes.length > 0 && <LegendeDesBarres t={t} modele={modele} />}
    </>
  )
}

/** BlocPointille — le bloc d'un état vide : titre, phrase, jauges ; bordure pointillée, rien à cliquer. */
function BlocPointille({ bloc, t }: { bloc: BlocVide; t: TacticalText }) {
  return (
    <div
      role="status"
      className="flex flex-col gap-1.5 rounded-[10px] border-[1.5px] border-dashed border-placeholder-border p-3.5"
      data-testid="tactical-reapp-vide"
    >
      <b className="text-[13.5px] font-semibold">{bloc.titre}</b>
      <p className="m-0 text-[12.5px] text-muted-foreground">{bloc.corps}</p>
      {bloc.jauges.length > 0 && (
        <div className="mt-1 flex flex-wrap gap-2">
          {bloc.jauges.map((j) => (
            <Jauge key={j.cle} t={t} jauge={j} />
          ))}
        </div>
      )}
    </div>
  )
}

/** Jauge — un effectif contre son plancher ; verte (jeton `success`) une fois le plancher atteint. */
function Jauge({ t, jauge }: { t: TacticalText; jauge: JaugeDuBloc }) {
  const texte = jauge.cle === 'matchs' ? t.reappGaugeMatchs(jauge.valeur, jauge.plancher) : t.reappGaugeRespawns(jauge.valeur, jauge.plancher)
  return (
    <div className="flex min-w-[120px] flex-1 flex-col gap-1 text-[11.5px] text-muted-foreground" data-testid={`tactical-reapp-jauge-${jauge.cle}`} data-atteint={jauge.atteint}>
      <span>{texte}</span>
      <div
        className="relative h-[5px] overflow-hidden rounded-[3px] bg-muted"
        role="progressbar"
        aria-label={texte}
        aria-valuemin={0}
        aria-valuemax={jauge.plancher}
        aria-valuenow={Math.min(jauge.valeur, jauge.plancher)}
      >
        <i
          className="absolute inset-y-0 left-0 rounded-[3px]"
          style={{ width: `${jauge.fraction * 100}%`, backgroundColor: jauge.atteint ? tokenCssVar('success') : 'currentColor' }}
        />
      </div>
    </div>
  )
}

/** Barres — une ligne par zone : nom, barre épaisse (tenue) et barre fine (sans condition), parts à droite. */
function Barres({ locale, modele }: { locale: Locale; modele: LectureReapparitions }) {
  const servie = modele.etat === 'servie'
  const maximum = Math.max(servie ? modele.maxTenue : 0, modele.maxReference) || 1
  return (
    <ul className="flex flex-col gap-[7px]" data-testid="tactical-reapp-barres">
      {modele.lignes.map((l) => (
        <li key={l.volumeIndex} className="grid grid-cols-[minmax(0,104px)_minmax(0,1fr)_54px] items-center gap-x-2 text-xs" data-testid={`tactical-reapp-barre-${l.volumeIndex}`}>
          <span className="truncate" title={l.nom}>
            {l.nom}
          </span>
          <span className="flex flex-col gap-0.5" aria-hidden>
            {servie && (
              <span
                className="h-[9px] min-w-[2px] rounded-sm transition-[width] duration-500"
                style={{ width: `${((l.tenue ?? 0) / maximum) * 100}%`, backgroundColor: tokenCssVar('team-enemy') }}
              />
            )}
            <span className="h-1 min-w-[2px] rounded-sm bg-muted-foreground opacity-55 transition-[width] duration-500" style={{ width: `${(l.reference / maximum) * 100}%` }} />
          </span>
          <span className="text-right leading-[1.1]">
            {servie ? (
              <>
                <b className="block text-[12.5px]">{formaterLaPart(l.tenue ?? 0, locale)}</b>
                <span className="text-[10.5px] text-muted-foreground">{formaterLaPart(l.reference, locale)}</span>
              </>
            ) : (
              <span className="text-[10.5px] text-muted-foreground">{formaterLaPart(l.reference, locale)}</span>
            )}
          </span>
        </li>
      ))}
    </ul>
  )
}

/** LegendeDesBarres — sous le graphique : ce que disent la barre épaisse et la barre fine. */
function LegendeDesBarres({ t, modele }: { t: TacticalText; modele: LectureReapparitions }) {
  return (
    <div className="mt-3 flex flex-wrap gap-x-3.5 gap-y-1.5 border-t border-border pt-2.5 text-[11.5px] text-muted-foreground" data-testid="tactical-reapp-barres-legende">
      {modele.etat === 'servie' && modele.zone && (
        <span className="inline-flex items-center gap-1.5">
          <i className="h-[9px] w-3 rounded-[3px]" style={{ backgroundColor: tokenCssVar('team-enemy') }} aria-hidden />
          {t.reappBarHeld(modele.zone.nom, modele.tenue.reapparitions, modele.tenue.matchs)}
        </span>
      )}
      <span className="inline-flex items-center gap-1.5">
        <i className="h-1 w-3 rounded-[3px] bg-muted-foreground opacity-55" aria-hidden />
        {t.reappBarReference(modele.reference.reapparitions, modele.reference.matchs)}
      </span>
    </div>
  )
}
