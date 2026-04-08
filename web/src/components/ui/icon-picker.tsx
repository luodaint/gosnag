import { useState } from 'react'
import { cn } from '@/lib/utils'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import {
  Bug, Shield, Zap, Globe, Server, Database, Code, Terminal, Cpu, Cloud,
  Smartphone, Monitor, Lock, Key, Webhook, Layers, Package, Rocket,
  ShoppingCart, CreditCard, Users, Mail, Bell, Heart, Flame, Star,
  Activity, BarChart3, GitBranch, Boxes,
} from 'lucide-react'

const LUCIDE_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  bug: Bug, shield: Shield, zap: Zap, globe: Globe, server: Server,
  database: Database, code: Code, terminal: Terminal, cpu: Cpu, cloud: Cloud,
  smartphone: Smartphone, monitor: Monitor, lock: Lock, key: Key,
  webhook: Webhook, layers: Layers, package: Package, rocket: Rocket,
  'shopping-cart': ShoppingCart, 'credit-card': CreditCard, users: Users,
  mail: Mail, bell: Bell, heart: Heart, flame: Flame, star: Star,
  activity: Activity, 'bar-chart-3': BarChart3, 'git-branch': GitBranch, boxes: Boxes,
}

const EMOJIS = [
  '🚀', '🔥', '⚡', '🛡️', '🐛', '💳', '🛒', '📱', '🖥️', '☁️',
  '🔒', '📧', '🔔', '❤️', '⭐', '🎯', '🏠', '📊', '🔧', '⚙️',
  '🌐', '💾', '📦', '🎮', '🏦', '🏥', '🎓', '✈️', '🚗', '🍕',
  '🎵', '📸', '🔬', '💡', '🎨', '📝', '🗂️', '💬', '🤖', '🧪',
]

export const PROJECT_COLORS = [
  '#f59e0b', '#3b82f6', '#10b981', '#8b5cf6', '#ec4899', '#06b6d4',
  '#ef4444', '#f97316', '#84cc16', '#14b8a6', '#6366f1', '#a855f7',
]

export function resolveIcon(value: string): React.ReactNode {
  if (!value) return null
  if (value.startsWith('lucide:')) {
    const name = value.slice(7)
    const Icon = LUCIDE_ICONS[name]
    return Icon ? <Icon className="h-5 w-5" /> : null
  }
  return <span className="text-lg leading-none">{value}</span>
}

interface IconPickerProps {
  value: string
  color: string
  fallbackColor: string
  onChange: (icon: string, color: string) => void
  className?: string
}

export function IconPicker({ value, color, fallbackColor, onChange, className }: IconPickerProps) {
  const [open, setOpen] = useState(false)
  const [tab, setTab] = useState<'emoji' | 'icons'>('emoji')

  const activeColor = color || fallbackColor

  return (
    <div className={className}>
      <button
        type="button"
        onClick={(e) => { e.preventDefault(); e.stopPropagation(); setOpen(true) }}
        className={cn(
          'flex items-center justify-center rounded-md transition-colors',
          'h-8 w-8 hover:bg-accent/60',
          !value && 'text-muted-foreground/30 hover:text-muted-foreground/60'
        )}
      >
        {value ? resolveIcon(value) : (
          <span className="text-lg leading-none">+</span>
        )}
      </button>

      <Dialog open={open} onOpenChange={(v) => { if (!v) setOpen(false) }}>
        <DialogContent className="max-w-sm" onClick={(e) => e.stopPropagation()}>
          <DialogTitle>Icon & Color</DialogTitle>
          <DialogDescription className="sr-only">Choose an icon and color for this project</DialogDescription>

          <div className="space-y-4 mt-2">
            {/* Tabs */}
            <div className="flex gap-1 border-b border-border/40 pb-2">
              <button
                type="button"
                className={cn('px-2.5 py-1 text-xs rounded-md transition-colors', tab === 'emoji' ? 'bg-accent text-foreground' : 'text-muted-foreground hover:text-foreground')}
                onClick={() => setTab('emoji')}
              >Emoji</button>
              <button
                type="button"
                className={cn('px-2.5 py-1 text-xs rounded-md transition-colors', tab === 'icons' ? 'bg-accent text-foreground' : 'text-muted-foreground hover:text-foreground')}
                onClick={() => setTab('icons')}
              >Icons</button>
              {value && (
                <button
                  type="button"
                  className="ml-auto px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                  onClick={() => { onChange('', color); setOpen(false) }}
                >Clear icon</button>
              )}
            </div>

            {/* Emoji grid */}
            {tab === 'emoji' && (
              <div className="grid grid-cols-10 gap-1">
                {EMOJIS.map(emoji => (
                  <button
                    key={emoji}
                    type="button"
                    className={cn(
                      'h-8 w-8 flex items-center justify-center rounded text-lg hover:bg-accent/60 transition-colors',
                      value === emoji && 'bg-accent ring-1 ring-primary/40'
                    )}
                    onClick={() => { onChange(emoji, color); setOpen(false) }}
                  >{emoji}</button>
                ))}
              </div>
            )}

            {/* Lucide grid */}
            {tab === 'icons' && (
              <div className="grid grid-cols-10 gap-1">
                {Object.entries(LUCIDE_ICONS).map(([name, Icon]) => (
                  <button
                    key={name}
                    type="button"
                    className={cn(
                      'h-8 w-8 flex items-center justify-center rounded hover:bg-accent/60 transition-colors',
                      value === `lucide:${name}` && 'bg-accent ring-1 ring-primary/40'
                    )}
                    onClick={() => { onChange(`lucide:${name}`, color); setOpen(false) }}
                  >
                    <Icon className="h-4 w-4" />
                  </button>
                ))}
              </div>
            )}

            {/* Color row */}
            <div className="border-t border-border/40 pt-3">
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground/60 mb-2">Color</p>
              <div className="flex gap-2 flex-wrap">
                {PROJECT_COLORS.map(c => (
                  <button
                    key={c}
                    type="button"
                    className={cn(
                      'h-6 w-6 rounded-full transition-all',
                      activeColor === c ? 'ring-2 ring-offset-2 ring-offset-card scale-110' : 'hover:scale-110'
                    )}
                    style={{ backgroundColor: c }}
                    onClick={() => onChange(value, c)}
                  />
                ))}
                {color && (
                  <button
                    type="button"
                    className="h-6 px-2 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
                    onClick={() => onChange(value, '')}
                  >Reset</button>
                )}
              </div>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
