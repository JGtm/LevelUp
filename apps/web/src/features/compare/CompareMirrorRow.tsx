import type { CSSProperties } from 'react'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

export interface CompareMirrorRowProps {
  label: string
  valueA: string  // joueur actif — centre
  valueB: string  // challenger gauche
  valueC: string  // challenger droite
  rawA: number
  rawB: number
  rawC: number
  winnerAB: 'a' | 'b' | 'tie' | null  // 'a'=A gagne, 'b'=B gagne
  winnerAC: 'a' | 'b' | 'tie' | null  // 'a'=A gagne, 'b'=C gagne (API renvoie 'b' pour le target)
  sampleNoteB?: string
  sampleNoteC?: string
  /** false = valeur A indisponible (N/A) sur la métrique. */
  availableA?: boolean
  /** false = valeur B indisponible (N/A) — neutralise la barre gauche et le gagnant AB. */
  availableB?: boolean
  /** false = valeur C indisponible (N/A) — neutralise la barre droite et le gagnant AC. */
  availableC?: boolean
  /** Sens de la métrique : true = valeur basse meilleure (deaths_per_game, rendement…). Sert au calcul du top des 3. */
  lessIsBetter?: boolean
}

// Détermine quels joueurs ont la meilleure valeur parmi {A, B, C} disponibles.
// Renvoie l'ensemble des joueurs à égalité au sommet : un seul en cas de meilleur
// net, deux (ou trois) en cas d'ex æquo pour la meilleure stat — tous sont alors
// mis en valeur. Vide si moins de 2 comparables (rien à départager).
function pickTopOfThree(
  rawA: number, rawB: number, rawC: number,
  availA: boolean, availB: boolean, availC: boolean,
  lessIsBetter: boolean,
): Set<'a' | 'b' | 'c'> {
  const eps = 0.001
  const candidates: Array<{ key: 'a' | 'b' | 'c'; val: number }> = []
  if (availA && Number.isFinite(rawA)) candidates.push({ key: 'a', val: rawA })
  if (availB && Number.isFinite(rawB)) candidates.push({ key: 'b', val: rawB })
  if (availC && Number.isFinite(rawC)) candidates.push({ key: 'c', val: rawC })
  if (candidates.length < 2) return new Set() // pas assez de comparables pour parler de "top"
  const best = candidates.reduce((m, c) => ((lessIsBetter ? c.val < m.val : c.val > m.val) ? c : m))
  // Tous les candidats à epsilon de la meilleure valeur sont co-leaders (ex æquo).
  return new Set(candidates.filter((c) => Math.abs(c.val - best.val) <= eps).map((c) => c.key))
}

export function CompareMirrorRow({
  label, valueA, valueB, valueC,
  rawA, rawB, rawC,
  winnerAB, winnerAC,
  sampleNoteB, sampleNoteC,
  availableA = true, availableB = true, availableC = true,
  lessIsBetter = false,
}: CompareMirrorRowProps) {
  const colorA = tokenCssVar('compare-a' as SemanticToken)
  const colorB = tokenCssVar('compare-b' as SemanticToken)
  const colorC = tokenCssVar('compare-c' as SemanticToken)

  const leftPairAvail = availableA && availableB
  const rightPairAvail = availableA && availableC
  const effWinnerAB = leftPairAvail ? winnerAB : 'tie'
  const effWinnerAC = rightPairAvail ? winnerAC : 'tie'

  // Top des 3 joueurs : chaque joueur à la meilleure valeur globale (y compris
  // les ex æquo) est mis en gras + couleur.
  const topPlayers = pickTopOfThree(rawA, rawB, rawC, availableA, availableB, availableC, lessIsBetter)

  // Barre gauche : B | A (B en couleur compare-b à gauche, A en compare-a à droite)
  const a1 = availableA && Number.isFinite(rawA) ? rawA : 0
  const b1 = availableB && Number.isFinite(rawB) ? rawB : 0
  const totalLeft = a1 + b1
  const bRatio = leftPairAvail && totalLeft > 0
    ? Math.max(0.05, Math.min(0.95, b1 / totalLeft))
    : 0.5

  // Barre droite : A | C (A en compare-a à gauche, C en compare-c à droite)
  const a2 = availableA && Number.isFinite(rawA) ? rawA : 0
  const c2 = availableC && Number.isFinite(rawC) ? rawC : 0
  const totalRight = a2 + c2
  const aRatio = rightPairAvail && totalRight > 0
    ? Math.max(0.05, Math.min(0.95, a2 / totalRight))
    : 0.5

  const leftBarStyle: CSSProperties = {
    background: `linear-gradient(to right, ${colorB} ${(bRatio * 100).toFixed(1)}%, ${colorA} ${(bRatio * 100).toFixed(1)}%)`,
    opacity: leftPairAvail ? (effWinnerAB === 'tie' ? 0.75 : 1) : 0.3,
  }

  const rightBarStyle: CSSProperties = {
    background: `linear-gradient(to right, ${colorA} ${(aRatio * 100).toFixed(1)}%, ${colorC} ${(aRatio * 100).toFixed(1)}%)`,
    opacity: rightPairAvail ? (effWinnerAC === 'tie' ? 0.75 : 1) : 0.3,
  }

  const naClass = 'text-sm tabular-nums leading-tight text-muted-foreground italic'
  const valueClass = 'text-sm tabular-nums leading-tight'

  return (
    <div className="space-y-1 w-full">
      <p className="text-center text-xs text-muted-foreground leading-tight">{label}</p>
      <div
        style={{
          // Colonnes de valeur élargies (4.5rem) : les libellés texte « Rang
          // carrière » / « CSR » (ex. "Diamant 3", "Non classé") tiennent sur une
          // seule ligne et restent alignés comme les valeurs numériques.
          display: 'grid',
          gridTemplateColumns: '4.5rem 1fr 4.5rem 1fr 4.5rem',
          gap: '0.5rem',
          alignItems: 'center',
        }}
      >
        {/* Valeur B (challenger gauche) */}
        <div className="flex min-w-0 flex-col items-end">
          <span
            title={valueB}
            className={`w-full truncate text-right ${availableB ? valueClass : naClass}`}
            style={availableB && topPlayers.has('b') ? { color: colorB, fontWeight: 600 } : undefined}
          >
            {valueB}
          </span>
          {sampleNoteB && (
            <span className="text-2xs leading-tight text-muted-foreground">{sampleNoteB}</span>
          )}
        </div>

        {/* Barre B vs A */}
        <div className="h-3 rounded-sm" style={leftBarStyle} />

        {/* Valeur A (joueur actif — ancre centrale) */}
        <div className="flex min-w-0 flex-col items-center">
          <span
            title={valueA}
            className={`w-full truncate text-center ${availableA ? valueClass : naClass}`}
            style={availableA && topPlayers.has('a') ? { color: colorA, fontWeight: 600 } : undefined}
          >
            {valueA}
          </span>
        </div>

        {/* Barre A vs C */}
        <div className="h-3 rounded-sm" style={rightBarStyle} />

        {/* Valeur C (challenger droite) */}
        <div className="flex min-w-0 flex-col items-start">
          <span
            title={valueC}
            className={`w-full truncate text-left ${availableC ? valueClass : naClass}`}
            style={availableC && topPlayers.has('c') ? { color: colorC, fontWeight: 600 } : undefined}
          >
            {valueC}
          </span>
          {sampleNoteC && (
            <span className="text-2xs leading-tight text-muted-foreground">{sampleNoteC}</span>
          )}
        </div>
      </div>
    </div>
  )
}
