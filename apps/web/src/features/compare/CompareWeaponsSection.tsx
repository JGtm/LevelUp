/**
 * CompareWeaponsSection — LE PROFIL D'ARMES DU FACE-À-FACE, en trois blocs.
 *
 * Plan `.ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md`, lot 4 :
 *
 *  1. la PART DES FRAGS PAR CLASSE, une barre par classe, les deux joueurs face à face ;
 *  2. la PORTÉE PAR RÔLE, deux graphes (« Où ils fraguent », « Où ils meurent »), chacun
 *     superposant les deux joueurs sur la même bande ;
 *  3. les ARMES LES PLUS UTILISÉES, trois par joueur, icône et nom.
 *
 * # AUCUN VAINQUEUR N'EST ÉLU (D3), ET CE N'EST PAS UN OUBLI
 *
 * Toutes les lignes passent `winner={null}`, rendu comme une égalité. Une distance plus longue
 * n'est pas meilleure ; une part de frags décrit un STYLE. Élire un gagnant ferait dire à ces
 * nombres ce qu'ils ne mesurent pas.
 *
 * # CE COMPOSANT NE CALCULE RIEN
 *
 * L'union des classes, l'union des rôles, le tri, les libellés et les couvertures vivent dans
 * `compareWeapons_logic.ts` (pur, testé hors rendu) ; l'option ECharts vient du module de la
 * Synthèse, rendu title-agnostic pour être partagé (`top`/`bottom` et non `kills`/`deaths`).
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { CompareResponse, CompareWeaponSide, CompareTopWeapon } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { CompareBar } from './CompareBar'
import { CompareMirrorRow } from './CompareMirrorRow'
import { CompareWeaponsRange } from './CompareWeaponsRange'
import { fragClassRows, hasWeaponProfile } from './compareWeapons_logic'
import {
  TOKEN_A,
  TOKEN_B,
  TOKEN_C,
  useRoleName,
  useWeaponFormats,
} from './compareWeaponsShared'
import type { CompareText } from './i18n'

// ─── Bloc 1 : la part des frags par classe ────────────────────────────────────

/**
 * FragClassBars — une ligne par classe, les deux (ou trois) joueurs face à face.
 *
 * LA VALEUR AFFICHÉE EST LA PART, pas le nombre de frags : c'est elle qui rend deux joueurs
 * aux volumes très différents comparables. Le nombre brut reste dans l'attribut de titre de la
 * barre, pour qui veut le dénominateur.
 */
function FragClassBars({
  left,
  right,
  names,
  text,
  locale,
}: {
  left: CompareResponse
  right?: CompareResponse
  names: { a: string; b: string; c?: string }
  text: CompareText
  locale: Locale
}) {
  const f = useWeaponFormats(locale)
  const roleName = useRoleName(locale)
  const sideA = left.weapons?.player_a
  const sideB = left.weapons?.player_b
  const sideC = right?.weapons?.player_b

  const rowsAB = fragClassRows(sideA, sideB)
  const rowsAC = fragClassRows(sideA, sideC)
  // En miroir, l'union des deux unions : une classe que seul C porte doit avoir sa ligne.
  const cles = right
    ? [...new Set([...rowsAB.map((r) => r.classKey), ...rowsAC.map((r) => r.classKey)])]
    : rowsAB.map((r) => r.classKey)
  if (cles.length === 0) return null

  const parAB = new Map(rowsAB.map((r) => [r.classKey, r]))
  const parAC = new Map(rowsAC.map((r) => [r.classKey, r]))

  return (
    <div className="space-y-3">
      <p className="text-xs font-medium text-muted-foreground">{text.weaponsClassesTitle}</p>
      {cles.map((key) => {
        const ab = parAB.get(key)
        const ac = parAC.get(key)
        const label = roleName(key)
        if (right) {
          return (
            <CompareMirrorRow
              key={key}
              label={label}
              valueA={f.percent(ab?.sharePctA ?? ac?.sharePctA ?? 0)}
              valueB={f.percent(ab?.sharePctB ?? 0)}
              valueC={f.percent(ac?.sharePctB ?? 0)}
              rawA={ab?.sharePctA ?? ac?.sharePctA ?? 0}
              rawB={ab?.sharePctB ?? 0}
              rawC={ac?.sharePctB ?? 0}
              // Aucun vainqueur (D3) : une part de frags décrit un style, pas une performance.
              winnerAB={null}
              winnerAC={null}
              sampleNoteB={sideB ? text.weaponsMatches(sideB.matches) : undefined}
              sampleNoteC={sideC ? text.weaponsMatches(sideC.matches) : undefined}
            />
          )
        }
        return (
          <CompareBar
            key={key}
            label={label}
            valueA={f.percent(ab?.sharePctA ?? 0)}
            valueB={f.percent(ab?.sharePctB ?? 0)}
            rawA={ab?.sharePctA ?? 0}
            rawB={ab?.sharePctB ?? 0}
            winner={null}
            ariaLabel={`${label} — ${names.a} / ${names.b}`}
            sampleNote={sideB ? text.weaponsMatches(sideB.matches) : undefined}
          />
        )
      })}
    </div>
  )
}

// ─── Bloc 3 : les armes les plus utilisées ────────────────────────────────────

/** TopWeaponsColumn — les trois armes d'UN joueur, icône et frags. */
function TopWeaponsColumn({
  name,
  side,
  token,
  locale,
  fmtCount,
}: {
  name: string
  side: CompareWeaponSide | null | undefined
  token: SemanticToken
  locale: Locale
  fmtCount: (n: number) => string
}) {
  const armes = side?.top_weapons ?? []
  const nomDe = (w: CompareTopWeapon) =>
    (locale === 'en' ? w.label_en : w.label)?.trim() || w.label || w.label_en || ''
  return (
    <div className="space-y-1.5">
      <p className="text-xs font-semibold" style={{ color: tokenCssVar(token) }}>
        {name}
      </p>
      {armes.map((w, i) => (
        <div key={`${w.label}-${i}`} className="flex items-center gap-2 text-sm">
          {/* Sans URL, `WeaponIcon` ne rend RIEN : le nom seul, pas un cadre vide. */}
          <WeaponIcon
            imageUrl={w.image_url}
            tinted={w.image_tinted}
            label={nomDe(w)}
            width={28}
            height={18}
          />
          <span className="min-w-0 flex-1 truncate">{nomDe(w)}</span>
          <span className="shrink-0 tabular-nums text-muted-foreground">{fmtCount(w.kills)}</span>
        </div>
      ))}
    </div>
  )
}

// ─── Section ──────────────────────────────────────────────────────────────────

export interface CompareWeaponsSectionProps {
  left: CompareResponse
  /** Présent = mode miroir (B | A | C). */
  right?: CompareResponse
  text: CompareText
  locale: Locale
}

export function CompareWeaponsSection({ left, right, text, locale }: CompareWeaponsSectionProps) {
  const f = useWeaponFormats(locale)
  const sideA = left.weapons?.player_a
  const sideB = left.weapons?.player_b
  const sideC = right?.weapons?.player_b

  // SECTION ENTIÈREMENT ABSENTE quand aucune réponse ne porte de profil — jamais une section
  // vide, qui se lirait comme un chargement bloqué.
  if (!hasWeaponProfile(sideA, sideB) && !hasWeaponProfile(sideA, sideC)) return null

  const nomA = left.player_a.gamertag
  const nomB = left.player_b.gamertag
  const nomC = right?.player_b.gamertag ?? ''

  return (
    <section className="space-y-5">
      <h2 className="text-sm font-semibold">{text.catWeapons}</h2>

      <FragClassBars
        left={left}
        right={right}
        names={{ a: nomA, b: nomB, c: nomC }}
        text={text}
        locale={locale}
      />

      <div className={right ? 'grid grid-cols-1 gap-4 xl:grid-cols-2' : undefined}>
        <CompareWeaponsRange
          sideA={sideA}
          sideB={sideB}
          names={{ a: nomA, b: nomB }}
          colorBottomToken={TOKEN_B}
          text={text}
          locale={locale}
        />
        {right && (
          <CompareWeaponsRange
            sideA={sideA}
            sideB={sideC}
            names={{ a: nomA, b: nomC }}
            colorBottomToken={TOKEN_C}
            text={text}
            locale={locale}
          />
        )}
      </div>

      <TopWeaponsRow
        colonnes={
          right
            ? // En miroir, l'ordre de lecture de la page est B | A | C.
              [
                { name: nomB, side: sideB, token: TOKEN_B },
                { name: nomA, side: sideA, token: TOKEN_A },
                { name: nomC, side: sideC, token: TOKEN_C },
              ]
            : [
                { name: nomA, side: sideA, token: TOKEN_A },
                { name: nomB, side: sideB, token: TOKEN_B },
              ]
        }
        text={text}
        locale={locale}
        fmtCount={f.count}
      />
    </section>
  )
}

/** TopWeaponsRow — le bloc 3 : une colonne d'armes par joueur, dans l'ordre de la page. */
function TopWeaponsRow({
  colonnes,
  text,
  locale,
  fmtCount,
}: {
  colonnes: { name: string; side: CompareWeaponSide | null | undefined; token: SemanticToken }[]
  text: CompareText
  locale: Locale
  fmtCount: (n: number) => string
}) {
  return (
    <div className="space-y-2">
      <p className="text-xs font-medium text-muted-foreground">{text.weaponsTopTitle}</p>
      <div
        className="grid gap-6"
        style={{ gridTemplateColumns: `repeat(${colonnes.length}, minmax(0, 1fr))` }}
      >
        {colonnes.map((c) => (
          <TopWeaponsColumn
            key={c.name}
            name={c.name}
            side={c.side}
            token={c.token}
            locale={locale}
            fmtCount={fmtCount}
          />
        ))}
      </div>
    </div>
  )
}
