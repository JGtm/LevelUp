import { describe, expect, it, beforeEach } from 'vitest'
import { useFeedbackDrawerStore } from './feedbackDrawer.store'

beforeEach(() => {
  useFeedbackDrawerStore.setState({ isOpen: false })
  window.localStorage.removeItem('levelup-feedback-drawer')
})

describe('feedbackDrawer.store', () => {
  it('open() ouvre le drawer', () => {
    useFeedbackDrawerStore.getState().open()
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(true)
  })

  it('close() ferme le drawer', () => {
    useFeedbackDrawerStore.setState({ isOpen: true })
    useFeedbackDrawerStore.getState().close()
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(false)
  })

  it('toggle() bascule', () => {
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(false)
    useFeedbackDrawerStore.getState().toggle()
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(true)
    useFeedbackDrawerStore.getState().toggle()
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(false)
  })
})
