import { useState, useMemo, useCallback, useRef } from 'react'
import { cn } from '@/lib/utils'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import data from '@emoji-mart/data'
import Picker from '@emoji-mart/react'
// @ts-expect-error — no type declarations for internal DynamicIcon module
import DynamicIcon, { iconNames } from 'lucide-react/dist/esm/DynamicIcon'
import { PROJECT_COLORS, resolveIcon } from '@/components/ui/icon-picker-utils'

// Deduplicate icon names (some are aliases)
const ICON_NAMES = iconNames.filter((n: string) => !n.includes('_'))

const ICONS_PER_PAGE = 80

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
  const [iconSearch, setIconSearch] = useState('')
  const [iconPage, setIconPage] = useState(0)
  const gridRef = useRef<HTMLDivElement>(null)

  const activeColor = color || fallbackColor

  const resetIconBrowser = useCallback(() => {
    setIconSearch('')
    setIconPage(0)
    gridRef.current?.scrollTo(0, 0)
  }, [])

  const handleOpenChange = useCallback((nextOpen: boolean) => {
    setOpen(nextOpen)
    if (nextOpen) resetIconBrowser()
  }, [resetIconBrowser])

  const handleTabChange = useCallback((nextTab: 'emoji' | 'icons') => {
    setTab(nextTab)
    resetIconBrowser()
  }, [resetIconBrowser])

  const filteredIcons = useMemo(() => {
    if (!iconSearch) return ICON_NAMES
    const q = iconSearch.toLowerCase()
    return ICON_NAMES.filter((name: string) => name.includes(q))
  }, [iconSearch])

  const pagedIcons = useMemo(() => {
    const start = iconPage * ICONS_PER_PAGE
    return filteredIcons.slice(start, start + ICONS_PER_PAGE)
  }, [filteredIcons, iconPage])

  const totalPages = Math.ceil(filteredIcons.length / ICONS_PER_PAGE)

  const handleEmojiSelect = useCallback((emoji: { native: string }) => {
    onChange(emoji.native, color)
    setOpen(false)
  }, [onChange, color])

  const handleIconSelect = useCallback((name: string) => {
    onChange(`lucide:${name}`, color)
    setOpen(false)
  }, [onChange, color])

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

      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="max-w-md p-0 overflow-hidden" onClick={(e) => e.stopPropagation()}>
          <div className="px-5 pt-5 pb-0">
            <DialogTitle>Icon & Color</DialogTitle>
            <DialogDescription className="sr-only">Choose an icon and color for this project</DialogDescription>
          </div>

          <div className="space-y-3 px-5 pb-5">
            {/* Tabs */}
            <div className="flex gap-1 border-b border-border/40 pb-2">
              <button
                type="button"
                className={cn('px-2.5 py-1 text-xs rounded-md transition-colors', tab === 'emoji' ? 'bg-accent text-foreground' : 'text-muted-foreground hover:text-foreground')}
                onClick={() => handleTabChange('emoji')}
              >Emoji</button>
              <button
                type="button"
                className={cn('px-2.5 py-1 text-xs rounded-md transition-colors', tab === 'icons' ? 'bg-accent text-foreground' : 'text-muted-foreground hover:text-foreground')}
                onClick={() => handleTabChange('icons')}
              >Icons</button>
              {value && (
                <button
                  type="button"
                  className="ml-auto px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                  onClick={() => { onChange('', color); setOpen(false) }}
                >Clear icon</button>
              )}
            </div>

            {/* Emoji picker (emoji-mart) */}
            {tab === 'emoji' && (
              <div className="[&_.em-emoji-picker]:!border-0 [&_.em-emoji-picker]:!bg-transparent [&_.em-emoji-picker]:!font-sans">
                <Picker
                  data={data}
                  onEmojiSelect={handleEmojiSelect}
                  theme="dark"
                  set="native"
                  skinTonePosition="search"
                  previewPosition="none"
                  navPosition="top"
                  perLine={9}
                  maxFrequentRows={1}
                  emojiSize={28}
                  emojiButtonSize={36}
                />
              </div>
            )}

            {/* Lucide icons grid with search */}
            {tab === 'icons' && (
              <div className="space-y-2">
                <Input
                  placeholder="Search icons..."
                  value={iconSearch}
                  onChange={(e) => { setIconSearch(e.target.value); setIconPage(0) }}
                  className="h-8 text-sm"
                  autoFocus
                />
                <div ref={gridRef} className="grid grid-cols-10 gap-1 min-h-[200px] max-h-[280px] overflow-y-auto">
                  {pagedIcons.map((name: string) => (
                    <button
                      key={name}
                      type="button"
                      title={name}
                      className={cn(
                        'h-8 w-8 flex items-center justify-center rounded hover:bg-accent/60 transition-colors',
                        value === `lucide:${name}` && 'bg-accent ring-1 ring-primary/40'
                      )}
                      onClick={() => handleIconSelect(name)}
                    >
                      <DynamicIcon name={name as never} className="h-4 w-4" />
                    </button>
                  ))}
                  {pagedIcons.length === 0 && (
                    <p className="col-span-10 text-center text-xs text-muted-foreground py-8">No icons found</p>
                  )}
                </div>
                {totalPages > 1 && (
                  <div className="flex items-center justify-between text-xs text-muted-foreground pt-1">
                    <span>{filteredIcons.length} icons</span>
                    <div className="flex gap-1">
                      <button
                        type="button"
                        disabled={iconPage === 0}
                        onClick={() => { setIconPage(p => p - 1); gridRef.current?.scrollTo(0, 0) }}
                        className="px-2 py-0.5 rounded hover:bg-accent disabled:opacity-30 disabled:cursor-not-allowed"
                      >&laquo; Prev</button>
                      <span className="px-1">{iconPage + 1}/{totalPages}</span>
                      <button
                        type="button"
                        disabled={iconPage >= totalPages - 1}
                        onClick={() => { setIconPage(p => p + 1); gridRef.current?.scrollTo(0, 0) }}
                        className="px-2 py-0.5 rounded hover:bg-accent disabled:opacity-30 disabled:cursor-not-allowed"
                      >Next &raquo;</button>
                    </div>
                  </div>
                )}
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
