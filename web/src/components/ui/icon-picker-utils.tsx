import type { ReactNode } from 'react'
// @ts-expect-error — no type declarations for internal DynamicIcon module
import DynamicIcon from 'lucide-react/dist/esm/DynamicIcon'

export const PROJECT_COLORS = [
  '#f59e0b', '#3b82f6', '#10b981', '#8b5cf6', '#ec4899', '#06b6d4',
  '#ef4444', '#f97316', '#84cc16', '#14b8a6', '#6366f1', '#a855f7',
]

export function resolveIcon(value: string): ReactNode {
  if (!value) return null
  if (value.startsWith('lucide:')) {
    const name = value.slice(7)
    return <DynamicIcon name={name as never} className="h-5 w-5" />
  }
  return <span className="text-lg leading-none">{value}</span>
}
