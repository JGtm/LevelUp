import * as React from 'react'

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className = '', ...props }, ref) => (
    <input
      ref={ref}
      className={`flex h-9 w-full rounded-md border border-gray-300 bg-white px-3 py-1 text-sm shadow-sm
        placeholder:text-gray-400 focus-visible:outline-none focus-visible:ring-2
        focus-visible:ring-purple-500 disabled:cursor-not-allowed disabled:opacity-50 ${className}`}
      {...props}
    />
  ),
)
Input.displayName = 'Input'
