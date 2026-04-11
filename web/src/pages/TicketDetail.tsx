import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api, isApiError, type Ticket, type Activity, type User, type Project, type IssueComment } from '@/lib/api'
import { useAuth } from '@/lib/use-auth'
import { Button } from '@/components/ui/button'
import { Select } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Check, ExternalLink, ArrowLeft, Pencil, Send } from 'lucide-react'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { cn } from '@/lib/utils'
import { toast } from '@/lib/use-toast'
import { Breadcrumb } from '@/components/ui/breadcrumb'

const STATUS_STYLE: Record<string, string> = {
  acknowledged: 'bg-amber-500/15 text-amber-400',
  in_progress: 'bg-blue-500/15 text-blue-400',
  in_review: 'bg-purple-500/15 text-purple-400',
  done: 'bg-emerald-500/15 text-emerald-400',
  wontfix: 'bg-slate-500/15 text-slate-400',
  escalated: 'bg-orange-500/15 text-orange-400',
}

const STATUS_LABEL: Record<string, string> = {
  acknowledged: 'Acknowledged',
  in_progress: 'In Progress',
  in_review: 'In Review',
  done: 'Done',
  wontfix: "Won't Fix",
  escalated: 'Escalated',
}

const TRANSITION_LABEL: Record<string, string> = {
  in_progress: 'Start',
  in_review: 'Submit for Review',
  done: 'Done',
  wontfix: "Won't Fix",
  escalated: 'Escalate',
  acknowledged: 'Reopen',
}

const PRIORITY_LABEL: Record<number, string> = { 90: 'P1 Critical', 70: 'P2 High', 50: 'P3 Medium', 25: 'P4 Low' }

export default function TicketDetail() {
  const { projectId, ticketId } = useParams<{ projectId: string; ticketId: string }>()
  const { user: currentUser } = useAuth()
  const [ticket, setTicket] = useState<Ticket | null>(null)
  const [project, setProject] = useState<Project | null>(null)
  const [users, setUsers] = useState<User[]>([])
  const [transitions, setTransitions] = useState<string[]>([])
  const [activities, setActivities] = useState<Activity[]>([])
  const [loading, setLoading] = useState(true)
  const [issueTitle, setIssueTitle] = useState('')
  const [issueCulprit, setIssueCulprit] = useState('')
  const [issueLevel, setIssueLevel] = useState('')
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState('')
  const [editingDescription, setEditingDescription] = useState(false)
  const [descriptionDraft, setDescriptionDraft] = useState('')
  const [comments, setComments] = useState<IssueComment[]>([])
  const [commentBody, setCommentBody] = useState('')
  const [submittingComment, setSubmittingComment] = useState(false)

  // Resolution dialog
  const [showResolution, setShowResolution] = useState(false)
  const [resolutionType, setResolutionType] = useState('fixed')
  const [fixReference, setFixReference] = useState('')
  const [resolutionNotes, setResolutionNotes] = useState('')

  useEffect(() => {
    if (!projectId || !ticketId) return
    Promise.all([
      api.getProject(projectId).then(setProject),
      api.getTicket(projectId, ticketId).then(async t => {
        setTicket(t)
        try {
          const issue = await api.getIssue(projectId, t.issue_id)
          setIssueTitle(issue.title)
          setIssueCulprit(issue.culprit || '')
          setIssueLevel(issue.level || '')
          api.listActivities(projectId, t.issue_id, { limit: 100 }).then(r => setActivities(r.activities)).catch(() => {})
          api.listComments(projectId, t.issue_id).then(setComments).catch(() => {})
        } catch { /* */ }
        api.getTicketTransitions(projectId, ticketId).then(r => setTransitions(r.transitions)).catch(() => {})
      }),
      api.listUsers().then(setUsers),
    ]).finally(() => setLoading(false))
  }, [projectId, ticketId])

  const refreshTicket = async () => {
    if (!projectId || !ticketId) return
    try {
      const t = await api.getTicket(projectId, ticketId)
      setTicket(t)
      api.getTicketTransitions(projectId, ticketId).then(r => setTransitions(r.transitions)).catch(() => {})
      if (t.issue_id) {
        api.listActivities(projectId, t.issue_id, { limit: 100 }).then(r => setActivities(r.activities)).catch(() => {})
      }
    } catch { /* */ }
  }

  const handleStatusChange = async (newStatus: string, force = false) => {
    if (!projectId || !ticketId || !ticket) return
    if (newStatus === 'done' && !force) {
      setResolutionType('fixed')
      setFixReference('')
      setResolutionNotes('')
      setShowResolution(true)
      return
    }
    try {
      const updated = await api.updateTicket(projectId, ticketId, { status: newStatus, force: force || undefined })
      setTicket(updated)
      api.getTicketTransitions(projectId, ticketId).then(r => setTransitions(r.transitions)).catch(() => {})
      if (ticket.issue_id) {
        api.listActivities(projectId, ticket.issue_id, { limit: 100 }).then(r => setActivities(r.activities)).catch(() => {})
      }
      toast.success(`Ticket → ${STATUS_LABEL[newStatus] || newStatus}${force ? ' (forced)' : ''}`)
    } catch (e: unknown) {
      if (isApiError(e) && e.status === 409 && e.body.can_force) {
        if (confirm(`Transition from "${ticket.status}" to "${newStatus}" is not standard. Force it?`)) {
          handleStatusChange(newStatus, true)
        }
      } else {
        toast.error(e instanceof Error ? e.message : 'Failed to update ticket')
      }
    }
  }

  const handleResolve = async () => {
    if (!projectId || !ticketId || !ticket) return
    try {
      const updated = await api.updateTicket(projectId, ticketId, {
        status: 'done',
        resolution_type: resolutionType,
        fix_reference: fixReference || undefined,
        resolution_notes: resolutionNotes || undefined,
      })
      setTicket(updated)
      setShowResolution(false)
      await refreshTicket()
      toast.success('Ticket resolved')
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed to resolve ticket')
    }
  }

  const handleAssign = async (userId: string | null) => {
    if (!projectId || !ticketId) return
    try {
      const updated = await api.updateTicket(projectId, ticketId, { assigned_to: userId || '' })
      setTicket(updated)
      await refreshTicket()
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed to assign')
    }
  }

  const handlePriorityChange = async (priority: number) => {
    if (!projectId || !ticketId) return
    try {
      const updated = await api.updateTicket(projectId, ticketId, { priority })
      setTicket(updated)
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed to change priority')
    }
  }

  const handleSaveDescription = async () => {
    if (!projectId || !ticketId) return
    try {
      const updated = await api.updateTicket(projectId, ticketId, { description: descriptionDraft })
      setTicket(updated)
      setEditingDescription(false)
      toast.success('Description saved')
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed to save description')
    }
  }

  const handleAddComment = async () => {
    if (!projectId || !ticket || !commentBody.trim()) return
    setSubmittingComment(true)
    try {
      const created = await api.createComment(projectId, ticket.issue_id, commentBody.trim())
      setComments(prev => [...prev, created])
      setCommentBody('')
      toast.success('Comment added')
    } finally {
      setSubmittingComment(false)
    }
  }

  if (loading) return <div className="p-8 text-center text-muted-foreground">Loading...</div>
  if (!ticket) return <div className="p-8 text-center text-muted-foreground">Ticket not found</div>

  return (
    <div>
      <Breadcrumb items={[
        { label: 'Projects', to: '/' },
        { label: project?.name || '', to: `/projects/${projectId}` },
        { label: 'Tickets', to: `/projects/${projectId}/tickets` },
        { label: `Ticket` },
      ]} />

      {/* Header */}
      <div className="flex items-start justify-between gap-4 mb-6">
        <div className="min-w-0">
          {issueCulprit && (
            <p className="text-sm font-semibold text-muted-foreground mb-0.5">{issueCulprit}</p>
          )}
          {editingTitle ? (
            <div className="flex items-center gap-2">
              <input
                value={titleDraft}
                onChange={e => setTitleDraft(e.target.value)}
                className="text-xl font-semibold bg-background border rounded px-2 py-0.5 flex-1"
                autoFocus
                onKeyDown={async e => {
                  if (e.key === 'Enter' && projectId && ticketId) {
                    const updated = await api.updateTicket(projectId, ticketId, { title: titleDraft })
                    setTicket(updated)
                    setEditingTitle(false)
                  }
                  if (e.key === 'Escape') setEditingTitle(false)
                }}
              />
              <Button size="sm" onClick={async () => {
                if (!projectId || !ticketId) return
                const updated = await api.updateTicket(projectId, ticketId, { title: titleDraft })
                setTicket(updated)
                setEditingTitle(false)
              }}>Save</Button>
              <Button size="sm" variant="outline" onClick={() => setEditingTitle(false)}>Cancel</Button>
            </div>
          ) : (
            <h1
              className="text-xl font-semibold truncate cursor-pointer hover:text-primary/80 transition-colors group"
              onClick={() => { setTitleDraft(ticket.title || issueTitle); setEditingTitle(true) }}
            >
              {ticket.title || issueTitle || 'Ticket'}
              <Pencil className="h-3.5 w-3.5 inline ml-2 opacity-0 group-hover:opacity-50" />
            </h1>
          )}
          {ticket.title && ticket.title !== issueTitle && (
            <p className="text-sm text-muted-foreground truncate">{issueTitle}</p>
          )}
          <div className="flex items-center gap-2 mt-1">
            <span className={cn('text-xs font-medium px-2 py-0.5 rounded-full', STATUS_STYLE[ticket.status] || 'bg-muted text-muted-foreground')}>
              {STATUS_LABEL[ticket.status] || ticket.status}
            </span>
            {issueLevel && (
              <span className={cn('text-xs font-medium px-2 py-0.5 rounded-full',
                issueLevel === 'fatal' || issueLevel === 'error' ? 'bg-red-500/15 text-red-400' :
                issueLevel === 'warning' ? 'bg-amber-500/15 text-amber-400' :
                'bg-muted text-muted-foreground'
              )}>{issueLevel}</span>
            )}
            <span className="text-xs text-muted-foreground">
              {PRIORITY_LABEL[ticket.priority] || `P${ticket.priority}`}
            </span>
            <Link
              to={`/projects/${projectId}/issues/${ticket.issue_id}`}
              className="text-xs text-primary hover:underline flex items-center gap-1"
            >
              <ArrowLeft className="h-3 w-3" /> View issue
            </Link>
          </div>
        </div>
      </div>

      {/* Escalated banner */}
      {ticket.status === 'escalated' && ticket.escalated_url && (
        <div className="mb-4 rounded-md bg-orange-500/10 border border-orange-500/20 p-3 text-sm">
          Tracked in{' '}
          <a href={ticket.escalated_url} target="_blank" rel="noopener noreferrer" className="font-medium text-orange-400 hover:underline">
            {ticket.escalated_key || 'external tracker'} <ExternalLink className="inline h-3 w-3" />
          </a>
        </div>
      )}

      {/* Resolution info */}
      {ticket.status === 'done' && ticket.resolution_type && (
        <div className="mb-4 rounded-md bg-emerald-500/10 border border-emerald-500/20 p-3 text-sm space-y-0.5">
          <div><span className="text-muted-foreground">Resolution:</span> <span className="font-medium">{ticket.resolution_type.replace('_', ' ')}</span></div>
          {ticket.fix_reference && <div><span className="text-muted-foreground">Fix:</span> <span className="font-mono text-xs">{ticket.fix_reference}</span></div>}
          {ticket.resolution_notes && <div className="text-xs text-muted-foreground mt-1">{ticket.resolution_notes}</div>}
        </div>
      )}

      {/* Management panel */}
      <div className="grid gap-4 md:grid-cols-2 mb-6">
        <div className="rounded-lg border bg-card p-4 space-y-4">
          <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Details</h3>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Assignee</span>
              <Select
                value={ticket.assigned_to || ''}
                onChange={e => handleAssign(e.target.value || null)}
                className="h-7 text-xs w-auto max-w-[180px]"
              >
                <option value="">Unassigned</option>
                {users.map(u => (
                  <option key={u.id} value={u.id}>{u.name || u.email}</option>
                ))}
              </Select>
            </div>

            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Priority</span>
              <Select
                value={String(ticket.priority)}
                onChange={e => handlePriorityChange(parseInt(e.target.value))}
                className="h-7 text-xs w-auto"
              >
                <option value="90">P1 Critical</option>
                <option value="70">P2 High</option>
                <option value="50">P3 Medium</option>
                <option value="25">P4 Low</option>
              </Select>
            </div>

            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Created</span>
              <span className="text-xs text-muted-foreground">{new Date(ticket.created_at).toLocaleString()}</span>
            </div>

            {ticket.due_date && (
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Due</span>
                <span className={cn(
                  'text-xs',
                  new Date(ticket.due_date) < new Date() ? 'text-red-400' : 'text-muted-foreground'
                )}>{new Date(ticket.due_date).toLocaleDateString()}</span>
              </div>
            )}
          </div>
        </div>

        <div className="rounded-lg border bg-card p-4 space-y-4">
          <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Actions</h3>
          <div className="flex flex-wrap gap-2">
            {transitions.map(t => (
              <Button
                key={t}
                size="sm"
                variant={t === 'done' ? 'default' : 'outline'}
                onClick={() => handleStatusChange(t)}
              >
                {TRANSITION_LABEL[t] || t.replace('_', ' ')}
              </Button>
            ))}
          </div>

          {/* All statuses as secondary options for forced transitions */}
          <div className="border-t pt-3">
            <p className="text-xs text-muted-foreground mb-2">Force transition to:</p>
            <div className="flex flex-wrap gap-1">
              {['acknowledged', 'in_progress', 'in_review', 'done', 'wontfix', 'escalated']
                .filter(s => s !== ticket.status && !transitions.includes(s))
                .map(s => (
                  <button
                    key={s}
                    className="text-xs px-2 py-1 rounded border text-muted-foreground hover:text-foreground hover:border-foreground/30 transition-colors"
                    onClick={() => handleStatusChange(s)}
                  >
                    {STATUS_LABEL[s] || s}
                  </button>
                ))}
            </div>
          </div>
        </div>
      </div>

      {/* Description */}
      <div className="mb-6 rounded-lg border bg-card p-4">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Description</h3>
          {!editingDescription && (
            <button
              onClick={() => { setDescriptionDraft(ticket.description || ''); setEditingDescription(true) }}
              className="text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              <Pencil className="h-3 w-3 inline mr-0.5" /> Edit
            </button>
          )}
        </div>
        {editingDescription ? (
          <div>
            <textarea
              value={descriptionDraft}
              onChange={e => setDescriptionDraft(e.target.value)}
              placeholder="Describe the problem, investigation notes, root cause..."
              className="w-full rounded-md border bg-background px-3 py-2 text-sm min-h-[120px] resize-y"
            />
            <div className="flex justify-end gap-2 mt-2">
              <Button size="sm" variant="outline" onClick={() => setEditingDescription(false)}>Cancel</Button>
              <Button size="sm" onClick={handleSaveDescription}>Save</Button>
            </div>
          </div>
        ) : ticket.description ? (
          <div className="text-sm prose prose-sm prose-invert max-w-none [&_pre]:bg-muted [&_pre]:p-2 [&_pre]:rounded [&_code]:text-xs [&_a]:text-primary [&_p]:my-1">
            <Markdown remarkPlugins={[remarkGfm]}>{ticket.description}</Markdown>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground/50 italic">No description yet. Click edit to add one.</p>
        )}
      </div>

      {/* Comments */}
      <div className="mb-6">
        <h2 className="text-base font-semibold mb-3">Comments ({comments.length})</h2>
        <div className="space-y-3">
          {comments.map(c => (
            <div key={c.id} className="flex gap-2">
              <div className="shrink-0">
                {c.user_avatar ? (
                  <img src={c.user_avatar} alt="" className="h-6 w-6 rounded-full" />
                ) : (
                  <div className="h-6 w-6 rounded-full bg-primary/20 flex items-center justify-center text-[10px] font-medium text-primary">
                    {(c.user_name || c.user_email || '?')[0]?.toUpperCase()}
                  </div>
                )}
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2 mb-0.5">
                  <span className="text-xs font-medium">{c.user_name || c.user_email}</span>
                  <span className="text-[10px] text-muted-foreground">{new Date(c.created_at).toLocaleString()}</span>
                </div>
                <div className="text-sm prose prose-sm prose-invert max-w-none [&_pre]:bg-muted [&_pre]:p-2 [&_pre]:rounded [&_code]:text-xs [&_a]:text-primary [&_p]:my-1">
                  <Markdown remarkPlugins={[remarkGfm]}>{c.body.replace(/@([\w.-]+)/g, '**@$1**')}</Markdown>
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Add comment */}
        <div className="flex gap-2 items-start mt-4">
          <div className="shrink-0 mt-0.5">
            {currentUser?.avatar_url ? (
              <img src={currentUser.avatar_url} alt="" className="h-6 w-6 rounded-full" />
            ) : (
              <div className="h-6 w-6 rounded-full bg-primary/20 flex items-center justify-center text-[10px] font-medium text-primary">
                {(currentUser?.name || currentUser?.email || '?')[0]?.toUpperCase()}
              </div>
            )}
          </div>
          <textarea
            value={commentBody}
            onChange={e => setCommentBody(e.target.value)}
            placeholder="Add a comment... (supports Markdown)"
            className="flex-1 text-sm rounded-md border bg-background px-3 py-2 min-h-[50px] resize-none"
          />
          <Button size="sm" disabled={!commentBody.trim() || submittingComment} onClick={handleAddComment} className="mt-0.5">
            <Send className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Activity timeline */}
      {activities.length > 0 && (
        <div className="mb-6">
          <h2 className="text-base font-semibold mb-3">Activity</h2>
          <div className="space-y-2">
            {[...activities].reverse().map(a => (
              <div key={a.id} className="flex items-start gap-2 text-xs text-muted-foreground">
                <div className={cn(
                  'mt-0.5 h-5 w-5 rounded-full flex items-center justify-center shrink-0 text-[10px] font-bold',
                  a.user_id ? 'bg-primary/20 text-primary' : 'bg-muted text-muted-foreground'
                )}>
                  {a.user_name ? a.user_name.charAt(0).toUpperCase() : 'S'}
                </div>
                <div className="min-w-0">
                  <span className="font-medium text-foreground">{a.user_name || 'System'}</span>
                  {' '}
                  {a.action === 'ticket_status_changed' ? (
                    <>changed status to <span className="font-mono">{a.new_value}</span></>
                  ) : a.action === 'ticket_created' ? (
                    <>created this ticket</>
                  ) : a.action === 'ticket_assigned' ? (
                    <>assigned the ticket</>
                  ) : a.action === 'ticket_priority_changed' ? (
                    <>changed priority to {a.new_value}</>
                  ) : a.action === 'status_changed' ? (
                    <>changed issue status to <span className="font-mono">{a.new_value}</span></>
                  ) : a.action === 'commented' ? (
                    <>added a comment</>
                  ) : a.action === 'auto_reopened' ? (
                    <>issue auto-reopened</>
                  ) : a.action === 'first_seen' ? (
                    <>issue detected</>
                  ) : (
                    <>{a.action.replace(/_/g, ' ')}</>
                  )}
                  <span className="ml-1.5 opacity-60">{new Date(a.created_at).toLocaleString()}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Resolution Dialog */}
      <Dialog open={showResolution} onOpenChange={setShowResolution}>
        <DialogContent>
          <DialogTitle>Resolve Ticket</DialogTitle>
          <DialogDescription className="sr-only">Provide resolution details</DialogDescription>
          <div className="mt-4 space-y-4">
            <div>
              <label className="text-sm font-medium">Resolution type</label>
              <Select value={resolutionType} onChange={e => setResolutionType(e.target.value)} className="mt-1">
                <option value="fixed">Fixed</option>
                <option value="wontfix">Won't fix</option>
                <option value="duplicate">Duplicate</option>
                <option value="cannot_reproduce">Cannot reproduce</option>
                <option value="by_design">By design</option>
              </Select>
            </div>
            <div>
              <label className="text-sm font-medium">Fix reference</label>
              <Input
                value={fixReference}
                onChange={e => setFixReference(e.target.value)}
                placeholder="Commit SHA, PR URL, release version..."
                className="mt-1"
              />
            </div>
            <div>
              <label className="text-sm font-medium">Notes</label>
              <textarea
                value={resolutionNotes}
                onChange={e => setResolutionNotes(e.target.value)}
                placeholder="What was the fix?"
                className="mt-1 w-full rounded-md border bg-background px-3 py-2 text-sm min-h-[80px] resize-y"
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setShowResolution(false)}>Cancel</Button>
              <Button onClick={handleResolve}>
                <Check className="h-4 w-4 mr-1" /> Resolve
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
