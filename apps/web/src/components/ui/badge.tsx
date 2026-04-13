import * as React from 'react'

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: 'default' | 'secondary' | 'success' | 'destructive' | 'outline'
}

export function Badge({ className = '', variant = 'default', ...props }: BadgeProps) {
  const base =
    'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors'
  const variants: Record<string, string> = {
    default: 'bg-purple-100 text-purple-800',
    secondary: 'bg-gray-100 text-gray-700',
    success: 'bg-green-100 text-green-800',
    destructive: 'bg-red-100 text-red-700',
    outline: 'border border-gray-300 text-gray-700',
  }
  return <span className={`${base} ${variants[variant]} ${className}`} {...props} />
}
