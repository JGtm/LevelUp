/**
 * Tests AchievementCard — couvre la sélection bilingue (avec fallback),
 * le rendering du gamerscore, de la barre de progression et de la date d'unlock.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { AchievementCard } from './AchievementCard'
import type { AchievementEntry } from '@/lib/api/types'

const baseAchievement: AchievementEntry = {
  achievement_id: 'a',
  name_en: 'First Blood',
  name_fr: 'Premier sang',
  description_en: 'Score your first kill',
  description_fr: 'Réalise ta première élimination',
  gamerscore: 10,
  is_secret: false,
  unlocked: false,
}

describe('AchievementCard', () => {
  it('affiche le nom FR quand locale=fr', () => {
    render(<AchievementCard achievement={baseAchievement} locale="fr" />)
    expect(screen.getByText('Premier sang')).toBeInTheDocument()
    expect(screen.queryByText('First Blood')).not.toBeInTheDocument()
  })

  it('affiche le nom EN quand locale=en', () => {
    render(<AchievementCard achievement={baseAchievement} locale="en" />)
    expect(screen.getByText('First Blood')).toBeInTheDocument()
    expect(screen.queryByText('Premier sang')).not.toBeInTheDocument()
  })

  it('fallback EN→FR quand locale=fr et name_fr vide', () => {
    const a = { ...baseAchievement, name_fr: '' }
    render(<AchievementCard achievement={a} locale="fr" />)
    expect(screen.getByText('First Blood')).toBeInTheDocument()
  })

  it('fallback FR→EN quand locale=en et name_en vide', () => {
    const a = { ...baseAchievement, name_en: '' }
    render(<AchievementCard achievement={a} locale="en" />)
    expect(screen.getByText('Premier sang')).toBeInTheDocument()
  })

  it('affiche le gamerscore', () => {
    render(<AchievementCard achievement={baseAchievement} locale="fr" />)
    expect(screen.getByText('10 G')).toBeInTheDocument()
  })

  it('affiche la barre de progression quand current_progress et target > 0', () => {
    const a: AchievementEntry = {
      ...baseAchievement,
      current_progress: 5,
      target_progress: 10,
    }
    render(<AchievementCard achievement={a} locale="fr" />)
    expect(screen.getByText('5 / 10')).toBeInTheDocument()
  })

  it('n\'affiche pas la barre de progression quand pas en cours', () => {
    render(<AchievementCard achievement={baseAchievement} locale="fr" />)
    expect(screen.queryByText(/\/ \d+/)).not.toBeInTheDocument()
  })

  it('affiche la date d\'unlock quand unlocked + unlocked_at', () => {
    const a: AchievementEntry = {
      ...baseAchievement,
      unlocked: true,
      unlocked_at: '2026-04-15T10:00:00Z',
    }
    render(<AchievementCard achievement={a} locale="fr" />)
    expect(screen.getByText(/Débloqué le/)).toBeInTheDocument()
    expect(screen.getByText(/2026/)).toBeInTheDocument()
  })

  it('utilise locked_desc quand locked, et description sinon', () => {
    const lockedFR = 'Pas encore débloqué'
    const a: AchievementEntry = {
      ...baseAchievement,
      locked_desc_en: 'Not yet earned',
      locked_desc_fr: lockedFR,
    }
    render(<AchievementCard achievement={a} locale="fr" />)
    expect(screen.getByText(lockedFR)).toBeInTheDocument()
    expect(screen.queryByText(baseAchievement.description_fr)).not.toBeInTheDocument()
  })

  it('utilise description quand unlocked (pas locked_desc)', () => {
    const a: AchievementEntry = {
      ...baseAchievement,
      unlocked: true,
      locked_desc_en: 'Not yet earned',
      locked_desc_fr: 'Pas encore débloqué',
    }
    render(<AchievementCard achievement={a} locale="fr" />)
    expect(screen.getByText(baseAchievement.description_fr)).toBeInTheDocument()
    expect(screen.queryByText('Pas encore débloqué')).not.toBeInTheDocument()
  })

  it('fallback locked_desc → description quand locked et locked_desc vide', () => {
    const a: AchievementEntry = {
      ...baseAchievement,
      locked_desc_en: '',
      locked_desc_fr: '',
    }
    render(<AchievementCard achievement={a} locale="fr" />)
    // Doit afficher la description normale comme fallback
    expect(screen.getByText(baseAchievement.description_fr)).toBeInTheDocument()
  })
})
