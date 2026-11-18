/**
 * FicheEliminations — « ÉLIMINATIONS PAR ARME » : une barre empilée des éliminations À L'INSTANT par classe d'arme,
 * puis UN tableau, par classe puis par arme, à trois colonnes — à l'instant, part, match.
 *
 * LES CLASSES SONT CELLES DE L'APPLICATION : mêmes jetons (`frag-*`, via `fragClassCssVar`) et mêmes libellés
 * (`frags.class.*`) que la page Synthèse, « Lourde » comprise. Les éliminations sans classe établie forment la
 * dernière ligne, au libellé et au jeton neutre de la classe « non attribuée » : aucune couleur de classe n'est
 * inventée pour elles. La barre est absente tant que le joueur n'a aucune élimination à l'instant (le tableau,
 * lui, existe dès que le match en compte, à zéro mesuré). Une part n'est écrite que quand elle existe.
 *
 * Le rendu ne lit rien : la barre et les lignes lui sont données (`KillsView`).
 */
import { Fragment } from 'react'

import { fragClassCssVar, type FragClassKey } from '@/lib/accessibility/scales/fragClass'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'

import { REPLAY_TEXT, type ReplayLocale } from '../../i18n/i18n'
import { COTES_FICHE_INSTANT } from '../../model/cotesFicheInstant'
import { formatCount, formatPercent } from '../../model/ficheFormat'
import type { KillsView } from '../../model/playerInstantView'
import { FicheSection } from './ficheParts'

const C = COTES_FICHE_INSTANT.tableauKills

/** Le libellé d'une classe : celui du manifeste de l'application. */
function classLabel(classKey: FragClassKey, locale: ReplayLocale): string {
  return formatMessage(fragsManifest, `frags.class.${classKey}` as never, locale)
}

const ROW_STYLE = { gridTemplateColumns: C.colonnes, columnGap: C.ecartPx, fontSize: C.taillePx }

/** La barre : un segment par classe, large de sa part des éliminations à l'instant. */
function Bar({ bar, locale }: { bar: KillsView['bar']; locale: ReplayLocale }) {
  const B = COTES_FICHE_INSTANT.barre
  const summary = bar.map((p) => `${classLabel(p.classKey, locale)} ${formatPercent(p.share * 100, locale)}`).join(', ')
  return (
    <div role="img" aria-label={summary} className="flex overflow-hidden" style={{ height: B.hauteurPx, borderRadius: B.rayonPx, gap: B.ecartPx }}>
      {bar.map((p) => (
        <i
          key={p.classKey}
          data-fiche-segment={p.classKey}
          className="block"
          style={{ flex: `${p.share} 1 0`, background: fragClassCssVar(p.classKey) }}
        />
      ))}
    </div>
  )
}

/** Une ligne de classe : la pastille de sa couleur, son libellé en gras, ses trois comptes. */
function ClassRow({ classKey, atInstant, share, atMatch, locale }: {
  classKey: FragClassKey
  atInstant: number
  share: number | null
  atMatch: number
  locale: ReplayLocale
}) {
  return (
    <div role="row" className="grid items-center font-semibold text-foreground" style={ROW_STYLE} data-fiche-classe={classKey}>
      <span role="cell" className="flex min-w-0 items-center">
        <i
          aria-hidden
          className="inline-block shrink-0"
          style={{ width: C.pastillePx, height: C.pastillePx, borderRadius: C.pastilleRayonPx, marginRight: C.pastilleEcartPx, background: fragClassCssVar(classKey) }}
        />
        <span className="truncate">{classLabel(classKey, locale)}</span>
      </span>
      <Counts atInstant={atInstant} share={share} atMatch={atMatch} locale={locale} />
    </div>
  )
}

/** Les trois colonnes de chiffres : à l'instant, part (en encre discrète), match. */
function Counts({ atInstant, share, atMatch, locale }: { atInstant: number; share: number | null; atMatch: number; locale: ReplayLocale }) {
  return (
    <>
      <span role="cell" className="text-right tabular-nums">{formatCount(atInstant, locale)}</span>
      <span role="cell" className="text-right tabular-nums text-muted-foreground">
        {share === null ? '' : formatPercent(share * 100, locale)}
      </span>
      <span role="cell" className="text-right tabular-nums">{formatCount(atMatch, locale)}</span>
    </>
  )
}

export function FicheEliminations({ kills, locale }: { kills: KillsView; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale].playerInstant
  const u = kills.unclassified
  return (
    <FicheSection title={t.killsByWeapon}>
      {kills.bar.length > 0 && <Bar bar={kills.bar} locale={locale} />}
      <div role="table" aria-label={t.killsByWeapon} className="flex flex-col" style={{ rowGap: COTES_FICHE_INSTANT.section.ecartPx }}>
        <div role="row" className="grid items-center text-muted-foreground" style={{ ...ROW_STYLE, fontSize: C.tailleEntetePx }}>
          <span role="columnheader" />
          <span role="columnheader" className="text-right">{t.now}</span>
          <span role="columnheader" className="text-right">{t.colShare}</span>
          <span role="columnheader" className="text-right">{t.colMatch}</span>
        </div>
        {kills.classes.map((cls) => (
          <Fragment key={cls.classKey}>
            <ClassRow classKey={cls.classKey} atInstant={cls.atInstant} share={cls.share} atMatch={cls.atMatch} locale={locale} />
            {cls.weapons.map((w, i) => (
              <div
                key={`${cls.classKey}-${i}`}
                role="row"
                className="grid items-center text-muted-foreground"
                style={{ ...ROW_STYLE, fontSize: C.tailleArmePx, paddingLeft: C.armeRetraitPx }}
              >
                <span role="cell" className="truncate">{w.name ?? t.unnamedWeapon}</span>
                <Counts atInstant={w.atInstant} share={w.share} atMatch={w.atMatch} locale={locale} />
              </div>
            ))}
          </Fragment>
        ))}
        {u && <ClassRow classKey="unattributed" atInstant={u.atInstant} share={u.share} atMatch={u.atMatch} locale={locale} />}
      </div>
    </FicheSection>
  )
}
