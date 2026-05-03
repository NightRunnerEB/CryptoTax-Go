import { useCallback, useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import {
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Clock,
  Download,
  FileText,
  Pause,
  PlayCircle,
  Plus,
  RefreshCw,
  XCircle,
} from 'lucide-react'
import { getTaxReportStatus, listTaxReports, startTaxReport, type TaxJob, type TaxPolicy } from '../api/taxService'
import { useAuth } from '../auth/AuthContext'
import { useNotifications } from '../components/notifications/NotificationProvider'
import { toErrorMessage } from '../utils/errors'

const UTC_DATE_TIME_FORMATTER = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'UTC',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
})

const TAX_YEAR_MIN = 2000
const TAX_YEAR_MAX = new Date().getUTCFullYear() + 1
const DEFAULT_LIMIT = 20
const JURISDICTION_OPTIONS = ['RU', 'KZ'] as const
const COST_BASIS_OPTIONS: Array<TaxPolicy['costBasisMethod']> = ['FIFO', 'LIFO', 'AVG']
const STATUS_POLL_INTERVAL_MS = 5_000

type JurisdictionOption = (typeof JURISDICTION_OPTIONS)[number]

interface CreateReportFormState {
  taxYear: number
  jurisdiction: JurisdictionOption
  costBasisMethod: TaxPolicy['costBasisMethod']
  treatCryptoCryptoAsDisposal: boolean
}

const INITIAL_CREATE_FORM: CreateReportFormState = {
  taxYear: new Date().getUTCFullYear() - 1,
  jurisdiction: 'RU',
  costBasisMethod: 'FIFO',
  treatCryptoCryptoAsDisposal: false,
}

function formatUtcTimestamp(value?: string): string {
  if (!value) {
    return '—'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return `${UTC_DATE_TIME_FORMATTER.format(date)} UTC`
}

function normalizeStatus(status: string): string {
  return status.trim().toLowerCase()
}

function resolveArtifactUrl(value?: string): string | null {
  if (!value || value.trim() === '') {
    return null
  }

  try {
    const url = new URL(value)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') {
      return null
    }
    return url.toString()
  } catch {
    return null
  }
}

function reportDisplayName(job: TaxJob): string {
  return `${job.taxYear} Tax Year Report`
}

function StatusBadge({ status }: { status: string }) {
  const normalized = normalizeStatus(status)

  const config = (() => {
    switch (normalized) {
      case 'queued':
        return {
          icon: Clock,
          label: 'Queued',
          bgColor: 'var(--status-queued-bg)',
          textColor: 'var(--status-queued)',
        }
      case 'running':
        return {
          icon: PlayCircle,
          label: 'Running',
          bgColor: 'var(--status-running-bg)',
          textColor: 'var(--status-running)',
        }
      case 'success':
        return {
          icon: CheckCircle2,
          label: 'Success',
          bgColor: 'var(--status-success-bg)',
          textColor: 'var(--status-success)',
        }
      case 'failed':
        return {
          icon: XCircle,
          label: 'Failed',
          bgColor: 'var(--status-failed-bg)',
          textColor: 'var(--status-failed)',
        }
      case 'canceled':
        return {
          icon: Pause,
          label: 'Canceled',
          bgColor: 'var(--status-canceled-bg)',
          textColor: 'var(--status-canceled)',
        }
      default:
        return {
          icon: Clock,
          label: normalized === '' ? 'Unknown' : normalized,
          bgColor: 'var(--status-canceled-bg)',
          textColor: 'var(--muted-foreground)',
        }
    }
  })()

  const Icon = config.icon

  return (
    <span
      className="inline-flex items-center gap-1.5 px-3 py-1 text-xs font-medium rounded-full"
      style={{ backgroundColor: config.bgColor, color: config.textColor }}
    >
      <Icon className="w-3.5 h-3.5" />
      {config.label}
    </span>
  )
}

export function ReportsPage() {
  const { session } = useAuth()
  const notifications = useNotifications()

  const [jobs, setJobs] = useState<TaxJob[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)

  const [isLoadingList, setIsLoadingList] = useState(false)
  const [listError, setListError] = useState<string | null>(null)

  const [expandedReportId, setExpandedReportId] = useState<string | null>(null)
  const [detailsByReportId, setDetailsByReportId] = useState<Record<string, TaxJob>>({})
  const [detailsLoadingByReportId, setDetailsLoadingByReportId] = useState<Record<string, boolean>>({})
  const [detailsErrorByReportId, setDetailsErrorByReportId] = useState<Record<string, string>>({})

  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)
  const [isCreating, setIsCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  const [createForm, setCreateForm] = useState<CreateReportFormState>(INITIAL_CREATE_FORM)

  const loadList = useCallback(
    async (nextOffset: number, options?: { silent?: boolean }): Promise<void> => {
      if (!session) {
        return
      }

      if (!options?.silent) {
        setIsLoadingList(true)
      }
      setListError(null)

      try {
        const response = await listTaxReports({ limit: DEFAULT_LIMIT, offset: nextOffset })
        setJobs(response.jobs)
        setTotal(response.total)
        setOffset(nextOffset)
      } catch (error) {
        setListError(toErrorMessage(error, 'Unable to load tax reports.'))
      } finally {
        if (!options?.silent) {
          setIsLoadingList(false)
        }
      }
    },
    [session],
  )

  useEffect(() => {
    void loadList(0)
  }, [loadList])

  const loadDetails = useCallback(
    async (reportId: string, force = false, options?: { silent?: boolean; notifyOnError?: boolean }): Promise<void> => {
      if (!session) {
        return
      }

      if (!force && detailsByReportId[reportId]) {
        return
      }

      if (!options?.silent) {
        setDetailsLoadingByReportId((prev) => ({
          ...prev,
          [reportId]: true,
        }))
      }

      setDetailsErrorByReportId((prev) => {
        const next = { ...prev }
        delete next[reportId]
        return next
      })

      try {
        const job = await getTaxReportStatus(reportId)
        setDetailsByReportId((prev) => ({
          ...prev,
          [reportId]: job,
        }))
      } catch (error) {
        const message = toErrorMessage(error, 'Unable to load report details.')
        setDetailsErrorByReportId((prev) => ({
          ...prev,
          [reportId]: message,
        }))

        if (options?.notifyOnError ?? true) {
          notifications.error('Unable to load report details', message)
        }
      } finally {
        if (!options?.silent) {
          setDetailsLoadingByReportId((prev) => ({
            ...prev,
            [reportId]: false,
          }))
        }
      }
    },
    [session, detailsByReportId, notifications],
  )

  useEffect(() => {
    if (!session) {
      return
    }

    const intervalId = window.setInterval(() => {
      void loadList(offset, { silent: true })

      if (expandedReportId) {
        void loadDetails(expandedReportId, true, {
          silent: true,
          notifyOnError: false,
        })
      }
    }, STATUS_POLL_INTERVAL_MS)

    return () => {
      window.clearInterval(intervalId)
    }
  }, [session, loadList, loadDetails, offset, expandedReportId])

  const toggleReport = (reportId: string): void => {
    const nextExpanded = expandedReportId === reportId ? null : reportId
    setExpandedReportId(nextExpanded)

    if (nextExpanded) {
      void loadDetails(nextExpanded)
    }
  }

  const handleCreateReport = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault()

    if (!session) {
      return
    }

    if (!Number.isInteger(createForm.taxYear) || createForm.taxYear < TAX_YEAR_MIN || createForm.taxYear > TAX_YEAR_MAX) {
      setCreateError(`Tax year must be between ${TAX_YEAR_MIN} and ${TAX_YEAR_MAX}.`)
      return
    }

    setIsCreating(true)
    setCreateError(null)

    try {
      const created = await startTaxReport({
        taxYear: createForm.taxYear,
        taxPolicy: {
          treatCryptoCryptoAsDisposal: createForm.treatCryptoCryptoAsDisposal,
          costBasisMethod: createForm.costBasisMethod,
          jurisdiction: createForm.jurisdiction,
        },
      })

      notifications.success('Report created', `Report ID: ${created.reportId}`)
      setIsCreateDialogOpen(false)
      setExpandedReportId(created.reportId)
      await loadList(0)
      void loadDetails(created.reportId, true)
    } catch (error) {
      setCreateError(toErrorMessage(error, 'Unable to create report.'))
    } finally {
      setIsCreating(false)
    }
  }

  const runningJobsCount = useMemo(
    () => jobs.filter((job) => normalizeStatus(job.status) === 'running').length,
    [jobs],
  )

  return (
    <div className="max-w-6xl">
      <div className="mb-8 flex items-start justify-between">
        <div className="flex flex-col gap-2">
          <h2 className="text-foreground">Tax Reports</h2>
          <p className="text-muted-foreground text-sm">Create and monitor asynchronous tax calculation jobs</p>
        </div>

        <button
          type="button"
          onClick={() => {
            setCreateError(null)
            setCreateForm(INITIAL_CREATE_FORM)
            setIsCreateDialogOpen(true)
          }}
          className="flex items-center gap-2 px-5 py-2.5 bg-primary hover:bg-primary-dark text-primary-foreground rounded-lg font-medium transition-all"
        >
          <Plus className="w-4 h-4" />
          Create Report
        </button>
      </div>

      {listError ? (
        <div className="bg-surface rounded-xl border border-[var(--status-failed)]/30 p-5 mb-6" style={{ boxShadow: 'var(--shadow-sm)' }}>
          <p className="text-sm text-[var(--status-failed)]">{listError}</p>
        </div>
      ) : null}

      <div className="bg-surface rounded-lg border border-border p-4 mb-6 flex items-center gap-3">
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 rounded-full bg-primary animate-pulse" />
          <span className="text-sm text-muted-foreground">Live monitoring - {runningJobsCount} job(s) running</span>
        </div>
        {runningJobsCount > 0 ? <div className="ml-auto text-sm text-muted-foreground">Auto-refresh in 5s</div> : null}
      </div>

      {isLoadingList && jobs.length === 0 ? (
        <div className="bg-surface rounded-xl border border-border p-8 mb-6 flex items-center justify-center gap-3" style={{ boxShadow: 'var(--shadow-sm)' }}>
          <RefreshCw className="w-5 h-5 animate-spin text-primary" />
          <span className="text-sm text-muted-foreground">Loading tax reports...</span>
        </div>
      ) : null}

      {!isLoadingList && jobs.length === 0 && !listError ? (
        <div className="bg-surface rounded-xl border border-border p-12 text-center" style={{ boxShadow: 'var(--shadow-sm)' }}>
          <FileText className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
          <p className="text-muted-foreground">No tax reports yet</p>
          <p className="text-sm text-muted-foreground mt-1">Create the first report job to start asynchronous tax calculation workflow.</p>
        </div>
      ) : null}

      <div className="space-y-4">
        {jobs.map((job) => {
          const expanded = expandedReportId === job.reportId
          const details = detailsByReportId[job.reportId] ?? job
          const detailsLoading = detailsLoadingByReportId[job.reportId] ?? false
          const detailsError = detailsErrorByReportId[job.reportId] ?? null
          const auditZipUrl = resolveArtifactUrl(job.auditZipUrl)
          const reportUrl = resolveArtifactUrl(job.reportUrl)
          const detailsAuditZipUrl = resolveArtifactUrl(details.auditZipUrl)
          const detailsReportUrl = resolveArtifactUrl(details.reportUrl)

          return (
            <div
              key={job.reportId}
              className="bg-surface rounded-xl border border-border overflow-hidden transition-all hover:border-primary/30"
              style={{ boxShadow: 'var(--shadow-md)' }}
            >
              <div className="p-6 cursor-pointer flex items-center justify-between" onClick={() => toggleReport(job.reportId)}>
                <div className="flex items-center gap-4 flex-1 min-w-0">
                  {expanded ? (
                    <ChevronDown className="w-5 h-5 text-muted-foreground flex-shrink-0" />
                  ) : (
                    <ChevronRight className="w-5 h-5 text-muted-foreground flex-shrink-0" />
                  )}

                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3 mb-2 flex-wrap">
                      <h3 className="text-foreground">{reportDisplayName(job)}</h3>
                      <StatusBadge status={job.status} />
                    </div>
                    <div className="flex items-center gap-6 text-sm text-muted-foreground flex-wrap">
                      <span>Year: {job.taxYear}</span>
                      <span>Created: {formatUtcTimestamp(job.createdAt)}</span>
                      {job.finishedAt ? <span>Completed: {formatUtcTimestamp(job.finishedAt)}</span> : null}
                    </div>
                  </div>

                  {normalizeStatus(job.status) === 'success' && (reportUrl || auditZipUrl) ? (
                    <div className="flex items-center gap-2" onClick={(event) => event.stopPropagation()}>
                      {reportUrl ? (
                        <a className="p-2 hover:bg-muted rounded-lg transition-colors" href={reportUrl} target="_blank" rel="noreferrer" aria-label="Open report">
                          <Download className="w-4 h-4 text-primary" />
                        </a>
                      ) : null}
                    </div>
                  ) : null}
                </div>
              </div>

              {expanded ? (
                <div className="border-t border-border bg-surface-secondary/30 p-6">
                  <div className="flex items-center justify-between gap-4 mb-6 flex-wrap">
                    <p className="text-xs font-mono text-muted-foreground break-all">{details.reportId}</p>
                    <button
                      type="button"
                      onClick={() => void loadDetails(job.reportId, true)}
                      disabled={detailsLoading}
                      className="inline-flex items-center gap-2 text-sm text-primary hover:text-primary-dark disabled:opacity-50"
                    >
                      <RefreshCw className={`w-4 h-4 ${detailsLoading ? 'animate-spin' : ''}`} />
                      {detailsLoading ? 'Refreshing...' : 'Refresh details'}
                    </button>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6 text-sm">
                    <div className="bg-surface rounded-lg p-4 border border-border">
                      <div className="flex justify-between gap-4">
                        <span className="text-muted-foreground">Started At</span>
                        <span className="text-foreground">{formatUtcTimestamp(details.startedAt)}</span>
                      </div>
                    </div>
                    <div className="bg-surface rounded-lg p-4 border border-border">
                      <div className="flex justify-between gap-4">
                        <span className="text-muted-foreground">Finished At</span>
                        <span className="text-foreground">{formatUtcTimestamp(details.finishedAt)}</span>
                      </div>
                    </div>
                    <div className="bg-surface rounded-lg p-4 border border-border">
                      <div className="flex justify-between gap-4">
                        <span className="text-muted-foreground">Attempts</span>
                        <span className="text-foreground font-medium font-mono">{details.attempts}</span>
                      </div>
                    </div>
                    <div className="bg-surface rounded-lg p-4 border border-border">
                      <div className="flex justify-between gap-4">
                        <span className="text-muted-foreground">Last Error Code</span>
                        <span className="text-foreground font-medium font-mono">{details.lastErrorCode || '—'}</span>
                      </div>
                    </div>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                    <div>
                      <h4 className="text-foreground mb-4 flex items-center gap-2">
                        <FileText className="w-4 h-4" />
                        Policy Snapshot
                      </h4>
                      <div className="space-y-3 bg-surface rounded-lg p-4 border border-border">
                        <div className="flex justify-between gap-4">
                          <span className="text-sm text-muted-foreground">Method:</span>
                          <span className="text-sm text-foreground font-medium">{details.policySnapshot.costBasisMethod}</span>
                        </div>
                        <div className="flex justify-between gap-4">
                          <span className="text-sm text-muted-foreground">Jurisdiction:</span>
                          <span className="text-sm text-foreground font-medium">{details.policySnapshot.jurisdiction}</span>
                        </div>
                        <div className="flex justify-between gap-4">
                          <span className="text-sm text-muted-foreground">Crypto Disposal:</span>
                          <span className="text-sm text-foreground font-medium">{details.policySnapshot.treatCryptoCryptoAsDisposal ? 'Enabled' : 'Disabled'}</span>
                        </div>
                      </div>
                    </div>

                    <div>
                      <h4 className="text-foreground mb-4">Summary</h4>
                      <div className="space-y-3 bg-surface rounded-lg p-4 border border-border">
                        {details.summary ? (
                          <>
                            <div className="flex justify-between gap-4">
                              <span className="text-sm text-muted-foreground">Total Income:</span>
                              <span className="text-sm text-foreground font-medium font-mono">{details.summary.totalIncomeFiat}</span>
                            </div>
                            <div className="flex justify-between gap-4">
                              <span className="text-sm text-muted-foreground">Total Expense:</span>
                              <span className="text-sm text-foreground font-medium font-mono">{details.summary.totalExpenseFiat}</span>
                            </div>
                            <div className="flex justify-between gap-4">
                              <span className="text-sm text-muted-foreground">Tax Base:</span>
                              <span className="text-sm text-foreground font-medium font-mono">{details.summary.taxBaseFiat}</span>
                            </div>
                            <div className="flex justify-between pt-3 border-t border-border gap-4">
                              <span className="text-sm font-medium text-foreground">Tax Due:</span>
                              <span className="text-base font-semibold text-primary font-mono">{details.summary.taxDueFiat}</span>
                            </div>
                          </>
                        ) : (
                          <p className="text-sm text-muted-foreground">Summary is not available until processing completes.</p>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="mt-6 pt-6 border-t border-border">
                    <h4 className="text-foreground mb-4">Download Artifacts</h4>
                    <div className="flex items-center gap-3 flex-wrap">
                      {detailsReportUrl ? (
                        <a
                          className="flex items-center gap-2 px-4 py-2 bg-primary hover:bg-primary-dark text-primary-foreground rounded-lg text-sm font-medium transition-all"
                          href={detailsReportUrl}
                          target="_blank"
                          rel="noreferrer"
                        >
                          <Download className="w-4 h-4" />
                          Tax Report
                        </a>
                      ) : null}
                      {detailsAuditZipUrl ? (
                        <a
                          className="flex items-center gap-2 px-4 py-2 border border-border hover:bg-muted rounded-lg text-sm font-medium transition-all"
                          href={detailsAuditZipUrl}
                          target="_blank"
                          rel="noreferrer"
                        >
                          <Download className="w-4 h-4" />
                          Audit Trail
                        </a>
                      ) : null}
                      {!detailsReportUrl && !detailsAuditZipUrl ? (
                        <p className="text-sm text-muted-foreground">Artifacts unavailable</p>
                      ) : null}
                    </div>
                  </div>

                  {detailsError ? (
                    <div className="mt-6 p-4 bg-[var(--status-failed-bg)] border border-[var(--status-failed)] rounded-lg">
                      <div className="flex items-start gap-2">
                        <XCircle className="w-5 h-5 text-[var(--status-failed)] flex-shrink-0 mt-0.5" />
                        <div>
                          <h4 className="text-sm font-medium text-foreground mb-1">Error Details</h4>
                          <p className="text-sm text-muted-foreground">{detailsError}</p>
                        </div>
                      </div>
                    </div>
                  ) : null}

                  {normalizeStatus(details.status) === 'failed' && details.lastErrorMessage ? (
                    <div className="mt-6 p-4 bg-[var(--status-failed-bg)] border border-[var(--status-failed)] rounded-lg">
                      <div className="flex items-start gap-2">
                        <XCircle className="w-5 h-5 text-[var(--status-failed)] flex-shrink-0 mt-0.5" />
                        <div>
                          <h4 className="text-sm font-medium text-foreground mb-1">Execution Error</h4>
                          <p className="text-sm text-muted-foreground">{details.lastErrorMessage}</p>
                        </div>
                      </div>
                    </div>
                  ) : null}
                </div>
              ) : null}
            </div>
          )
        })}
      </div>

      {total > jobs.length ? (
        <div className="mt-6 flex items-center justify-between gap-4 flex-wrap">
          <div className="text-sm text-muted-foreground">
            Showing <span className="font-medium text-foreground">{jobs.length}</span> report(s) from offset{' '}
            <span className="font-medium text-foreground">{offset}</span> of <span className="font-medium text-foreground">{total}</span>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => void loadList(Math.max(0, offset - DEFAULT_LIMIT))}
              disabled={isLoadingList || offset === 0}
              className="px-3 py-1.5 text-sm border border-border rounded-lg hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              Previous
            </button>
            <button
              type="button"
              onClick={() => void loadList(offset + DEFAULT_LIMIT)}
              disabled={isLoadingList || offset + DEFAULT_LIMIT >= total}
              className="px-3 py-1.5 text-sm border border-border rounded-lg hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              Next
            </button>
          </div>
        </div>
      ) : null}

      {isCreateDialogOpen ? (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => !isCreating && setIsCreateDialogOpen(false)}>
          <div
            className="bg-surface rounded-xl border border-border p-8 max-w-md w-full"
            style={{ boxShadow: 'var(--shadow-lg)' }}
            onClick={(event) => event.stopPropagation()}
          >
            <h3 className="text-foreground mb-6">Create New Tax Report</h3>
            <form className="space-y-4 mb-6" onSubmit={handleCreateReport}>
              <div>
                <label className="block text-sm text-foreground mb-2">Tax Year</label>
                <input
                  type="number"
                  min={TAX_YEAR_MIN}
                  max={TAX_YEAR_MAX}
                  value={createForm.taxYear}
                  onChange={(event) => {
                    const value = Number(event.target.value)
                    setCreateForm((prev) => ({
                      ...prev,
                      taxYear: Number.isFinite(value) ? value : prev.taxYear,
                    }))
                  }}
                  className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  disabled={isCreating}
                />
              </div>
              <div>
                <label className="block text-sm text-foreground mb-2">Jurisdiction</label>
                <select
                  value={createForm.jurisdiction}
                  onChange={(event) =>
                    setCreateForm((prev) => ({
                      ...prev,
                      jurisdiction: event.target.value as JurisdictionOption,
                    }))
                  }
                  className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  disabled={isCreating}
                >
                  {JURISDICTION_OPTIONS.map((jurisdiction) => (
                    <option key={jurisdiction} value={jurisdiction}>
                      {jurisdiction}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm text-foreground mb-2">Cost Basis Method</label>
                <select
                  value={createForm.costBasisMethod}
                  onChange={(event) =>
                    setCreateForm((prev) => ({
                      ...prev,
                      costBasisMethod: event.target.value as TaxPolicy['costBasisMethod'],
                    }))
                  }
                  className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  disabled={isCreating}
                >
                  {COST_BASIS_OPTIONS.map((method) => (
                    <option key={method} value={method}>
                      {method}
                    </option>
                  ))}
                </select>
              </div>
              <label
                className={`flex items-center justify-between gap-4 rounded-lg border border-border px-4 py-3 transition-colors ${
                  isCreating ? 'opacity-70' : 'cursor-pointer hover:bg-muted/40'
                }`}
              >
                <div className="min-w-0">
                  <div className="text-sm font-medium text-foreground">Treat crypto-to-crypto as disposal</div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    Count crypto swaps as taxable disposal events in the generated report.
                  </div>
                </div>
                <span
                  className={`relative inline-flex h-6 w-11 flex-shrink-0 rounded-full transition-colors ${
                    createForm.treatCryptoCryptoAsDisposal ? 'bg-primary' : 'bg-muted'
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={createForm.treatCryptoCryptoAsDisposal}
                    onChange={(event) =>
                      setCreateForm((prev) => ({
                        ...prev,
                        treatCryptoCryptoAsDisposal: event.target.checked,
                      }))
                    }
                    disabled={isCreating}
                    className="sr-only"
                  />
                  <span
                    className={`absolute top-0.5 h-5 w-5 rounded-full bg-white shadow-sm transition-transform ${
                      createForm.treatCryptoCryptoAsDisposal ? 'translate-x-5' : 'translate-x-0.5'
                    }`}
                  />
                </span>
              </label>
              {createError ? <p className="text-sm text-[var(--status-failed)]">{createError}</p> : null}
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={() => !isCreating && setIsCreateDialogOpen(false)}
                  className="flex-1 px-4 py-2.5 border border-border rounded-lg font-medium hover:bg-muted transition-all"
                  disabled={isCreating}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="flex-1 px-4 py-2.5 bg-primary hover:bg-primary-dark text-primary-foreground rounded-lg font-medium transition-all disabled:opacity-50"
                  disabled={isCreating}
                >
                  {isCreating ? 'Creating...' : 'Create Report'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </div>
  )
}
