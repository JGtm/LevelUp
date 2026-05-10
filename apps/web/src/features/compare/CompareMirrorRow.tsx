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
}

export function CompareMirrorRow({
  label, valueA, valueB, valueC,
  rawA, rawB, rawC,
  winnerAB, winnerAC,
  sampleNoteB, sampleNoteC,
}: CompareMirrorRowProps) {
  const colorA = tokenCssVar('compare-a' as SemanticToken)
  const colorB = tokenCssVar('compare-b' as SemanticToken)

  // Barre gauche : B | A (B en couleur compare-b à gauche, A en compare-a à droite)
  const a1 = Number.isFinite(rawA) ? rawA : 0
  const b1 = Number.isFinite(rawB) ? rawB : 0
  const totalLeft = a1 + b1
  const bRatio = totalLeft > 0 ? Math.max(0.05, Math.min(0.95, b1 / totalLeft)) : 0.5

  // Barre droite : A | C (A en compare-a à gauche, C en compare-b à droite)
  const a2 = Number.isFinite(rawA) ? rawA : 0
  const c2 = Number.isFinite(rawC) ? rawC : 0
  const totalRight = a2 + c2
  const aRatio = totalRight > 0 ? Math.max(0.05, Math.min(0.95, a2 / totalRight)) : 0.5

  const leftBarStyle: CSSProperties = {
    background: `linear-gradient(to right, ${colorB} ${(bRatio * 100).toFixed(1)}%, ${colorA} ${(bRatio * 100).toFixed(1)}%)`,
    opacity: winnerAB === 'tie' ? 0.75 : 1,
  }

  const rightBarStyle: CSSProperties = {
    background: `linear-gradient(to right, ${colorA} ${(aRatio * 100).toFixed(1)}%, ${colorB} ${(aRatio * 100).toFixed(1)}%)`,
    opacity: winnerAC === 'tie' ? 0.75 : 1,
  }

  return (
    <div className="space-y-1 w-full">
      <p className="text-center text-xs text-muted-foreground leading-tight">{label}</p>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '3.5rem 1fr 4rem 1fr 3.5rem',
          gap: '0.5rem',
          alignItems: 'center',
        }}
      >
        {/* Valeur B (challenger gauche) */}
        <div className="flex flex-col items-end">
          <span
            className="text-sm tabular-nums leading-tight"
            style={winnerAB === 'b' ? { color: colorB, fontWeight: 600 } : undefined}
          >
            {valueB}
          </span>
          {sampleNoteB && (
            <span className="text-[10px] leading-tight text-muted-foreground">{sampleNoteB}</span>
          )}
        </div>

        {/* Barre B vs A */}
        <div className="h-3 rounded-sm" style={leftBarStyle} />

        {/* Valeur A (joueur actif — ancre centrale) */}
        <div className="flex flex-col items-center">
          <span
            className="text-sm tabular-nums leading-tight font-medium"
            style={{ color: colorA }}
          >
            {valueA}
          </span>
        </div>

        {/* Barre A vs C */}
        <div className="h-3 rounded-sm" style={rightBarStyle} />

        {/* Valeur C (challenger droite) */}
        <div className="flex flex-col items-start">
          <span
            className="text-sm tabular-nums leading-tight"
            style={winnerAC === 'b' ? { color: colorB, fontWeight: 600 } : undefined}
          >
            {valueC}
          </span>
          {sampleNoteC && (
            <span className="text-[10px] leading-tight text-muted-foreground">{sampleNoteC}</span>
          )}
        </div>
      </div>
    </div>
  )
}
