import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

interface ThemeToggleProps {
  className?: string
}

function MoonIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 20 20"
      fill="currentColor"
      className="h-3.5 w-3.5"
      aria-hidden="true"
    >
      <path d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z" />
    </svg>
  )
}

function SunIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 20 20"
      fill="currentColor"
      className="h-3.5 w-3.5"
      aria-hidden="true"
    >
      <path
        fillRule="evenodd"
        d="M10 2a.75.75 0 01.75.75V4a.75.75 0 01-1.5 0V2.75A.75.75 0 0110 2zm0 11.5a3.5 3.5 0 100-7 3.5 3.5 0 000 7zm0 1.5a5 5 0 110-10 5 5 0 010 10zm6.25-5a.75.75 0 01.75-.75h1.25a.75.75 0 010 1.5H17a.75.75 0 01-.75-.75zm-14 0A.75.75 0 013 9.25h1.25a.75.75 0 010 1.5H3A.75.75 0 012.25 10zm10.58-4.33a.75.75 0 011.06 0l.88.88a.75.75 0 11-1.06 1.06l-.88-.88a.75.75 0 010-1.06zm-7.66 7.66a.75.75 0 011.06 0l.88.88a.75.75 0 11-1.06 1.06l-.88-.88a.75.75 0 010-1.06zm8.72 1.94a.75.75 0 010-1.06l.88-.88a.75.75 0 111.06 1.06l-.88.88a.75.75 0 01-1.06 0zm-7.66-7.66a.75.75 0 010-1.06l.88-.88a.75.75 0 111.06 1.06l-.88.88a.75.75 0 01-1.06 0z"
        clipRule="evenodd"
      />
    </svg>
  )
}

export function ThemeToggle({ className = '' }: ThemeToggleProps) {
  const theme = useSettingsDraftStore((state) => state.localUiPrefs.theme)
  const toggleTheme = useSettingsDraftStore((state) => state.toggleTheme)
  const isDark = theme === 'dark'
  const label = isDark ? 'Passer au thème clair' : 'Passer au thème sombre'

  return (
    <button
      type="button"
      role="switch"
      aria-checked={isDark}
      aria-label={label}
      title={label}
      onClick={toggleTheme}
      className={[
        'flex h-7 w-12 shrink-0 items-center rounded-full border border-sidebar-border px-1 text-sidebar-foreground/70 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40',
        isDark ? 'bg-sidebar-accent hover:bg-sidebar-accent/80' : 'bg-sidebar-primary/20 hover:bg-sidebar-primary/25',
        className,
      ].join(' ')}
    >
      <span className="sr-only">Thème</span>
      <span
        className={[
          'flex h-5 w-5 items-center justify-center rounded-full bg-background text-sidebar-foreground shadow-sm transition-transform duration-200',
          isDark ? 'translate-x-0' : 'translate-x-5',
        ].join(' ')}
      >
        {isDark ? <MoonIcon /> : <SunIcon />}
      </span>
    </button>
  )
}