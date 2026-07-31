/**
 * ReplayCoverage.tsx — le bandeau qui dit CE QUI N'EST PAS À L'ÉCRAN.
 *
 * POURQUOI CE COMPOSANT EXISTE. Le rejeu affichait un nombre de tirs sans jamais dire combien
 * le film en portait. Un trou de 34 secondes est ainsi resté invisible jusqu'à ce qu'un
 * utilisateur le remarque en regardant la carte. Le serveur publie désormais, pour chaque
 * calque, « rattachés sur disponibles » et la cause de chaque rejet ; ce composant les montre.
 *
 * AUCUNE LOGIQUE ICI : tout le calcul vit dans coverageLogic.ts, testé sans rendu (règle du
 * dépôt sur la logique métier dans les composants). Aucune couleur en dur non plus : les
 * statuts passent par les jetons sémantiques.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { ReplayDocument } from '@/lib/api/types'

import { bridgeIsRead, summarizeAll, type LayerSummary } from './coverageLogic'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'

interface Props {
  doc: ReplayDocument | undefined
  locale: ReplayLocale
}

/** Libellé i18n d'une cause de rejet. */
function causeLabel(cause: string, t: (typeof REPLAY_TEXT)['fr']): string {
  switch (cause) {
    case 'noSlot':
      return t.causeNoSlot
    case 'ambiguous':
      return t.causeAmbiguous
    case 'outOfWindow':
      return t.causeOutOfWindow
    default:
      return t.causeUnpublished
  }
}

/** Libellé i18n d'un calque. */
function layerLabel(key: string, t: (typeof REPLAY_TEXT)['fr']): string {
  return key === 'shots' ? t.layerShots : t.layerGrenades
}

function LayerRow({ s, t }: { s: LayerSummary; t: (typeof REPLAY_TEXT)['fr'] }) {
  // Le taux colore le RAPPORT, jamais le nombre seul : c'est le rapport qui se juge.
  const token = s.nominal ? 'success' : 'warning'
  return (
    <li className="flex flex-col gap-0.5 py-1">
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-sm">{layerLabel(s.key, t)}</span>
        <span
          className="font-mono text-sm tabular-nums"
          style={{ color: tokenCssVar(token) }}
          title={s.verdict}
        >
          {s.attached} / {s.available}
        </span>
      </div>
      {s.rejects.length > 0 && (
        <ul className="text-muted-foreground pl-3 text-xs">
          {s.rejects.map((r) => (
            <li key={r.cause}>
              {r.n} — {causeLabel(r.cause, t)}
            </li>
          ))}
        </ul>
      )}
    </li>
  )
}

/**
 * ReplayCoverageBanner affiche la couverture par calque. Il ne rend RIEN quand l'artefact n'en
 * porte pas : un artefact ancien ne doit pas se voir attribuer un dénominateur inventé.
 */
export function ReplayCoverageBanner({ doc, locale }: Props) {
  const t = REPLAY_TEXT[locale]
  const layers = summarizeAll(doc)
  if (layers.length === 0) return null

  const read = bridgeIsRead(doc)
  const leaking = layers.some((l) => !l.nominal && l.rejects.length === 0)

  return (
    <section className="border-border rounded-md border p-3">
      <h3 className="mb-1 text-sm font-medium">{t.coverageTitle}</h3>
      <ul className="divide-border divide-y">
        {layers.map((s) => (
          <LayerRow key={s.key} s={s} t={t} />
        ))}
      </ul>
      <p className="text-muted-foreground mt-2 text-xs">{t.coverageHint}</p>
      <p className="mt-1 text-xs" style={{ color: tokenCssVar(read ? 'success' : 'warning') }}>
        {read ? t.bridgeRead : t.bridgeNotRead}
      </p>
      {leaking && (
        <p className="mt-1 text-xs" style={{ color: tokenCssVar('destructive') }}>
          {t.coverageLeak}
        </p>
      )}
    </section>
  )
}
