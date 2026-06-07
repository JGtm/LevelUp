import * as React from 'react'

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: 'default' | 'secondary' | 'success' | 'info' | 'destructive' | 'outline'
}

export function Badge({ className = '', variant = 'default', ...props }: BadgeProps) {
  const base =
    'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors'
  const variants: Record<string, string> = {
    default: 'bg-primary/20 text-primary',
    secondary: 'bg-secondary text-secondary-foreground',
    success: 'bg-success/20 text-success',
    info: 'bg-info/20 text-info',
    destructive: 'bg-destructive/20 text-destructive',
    outline: 'border border-border text-foreground',
  }
  return <span className={`${base} ${variants[variant]} ${className}`} {...props} />
}
