import { useCallback, useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import {
  listSupportedExchanges,
  type ParseSuccessResponse,
  uploadExchangeCsv,
} from '../api/ledgerService'
import { useAuth } from '../auth/AuthContext'
import { PageHeader } from '../components/layout/PageHeader'
import { useNotifications } from '../components/notifications/NotificationProvider'
import { EmptyState } from '../components/states/EmptyState'
import { ErrorState } from '../components/states/ErrorState'
import { LoadingState } from '../components/states/LoadingState'
import { toErrorMessage } from '../utils/errors'

type ExchangeListState = 'idle' | 'loading' | 'ready' | 'empty' | 'error'

interface UploadResult {
  response: ParseSuccessResponse
  exchange: string
  fileName: string
  finishedAt: string
}

function formatDateTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return date.toLocaleString()
}

export function ImportsPage() {
  const { session } = useAuth()
  const notifications = useNotifications()

  const [exchanges, setExchanges] = useState<string[]>([])
  const [exchangeListState, setExchangeListState] = useState<ExchangeListState>('idle')
  const [exchangeError, setExchangeError] = useState<string | null>(null)

  const [selectedExchangeId, setSelectedExchangeId] = useState('')
  const [file, setFile] = useState<File | null>(null)

  const [isUploading, setIsUploading] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [lastSuccess, setLastSuccess] = useState<UploadResult | null>(null)

  const selectedExchange = useMemo(() => {
    return exchanges.find((exchange) => exchange === selectedExchangeId) ?? null
  }, [exchanges, selectedExchangeId])

  const loadSupportedExchanges = useCallback(async (): Promise<void> => {
    if (!session) {
      return
    }

    setExchangeListState('loading')
    setExchangeError(null)

    try {
      const result = await listSupportedExchanges()
      setExchanges(result)

      if (result.length === 0) {
        setSelectedExchangeId('')
        setExchangeListState('empty')
        return
      }

      setSelectedExchangeId(result[0])
      setExchangeListState('ready')
    } catch (error) {
      setExchangeError(toErrorMessage(error, 'Unable to load supported exchanges.'))
      setExchanges([])
      setSelectedExchangeId('')
      setExchangeListState('error')
    }
  }, [session])

  useEffect(() => {
    void loadSupportedExchanges()
  }, [loadSupportedExchanges])

  const handleSubmit = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault()

    if (!session || !selectedExchange || !file) {
      return
    }

    setIsUploading(true)
    setUploadError(null)
    setLastSuccess(null)

    try {
      const response = await uploadExchangeCsv({
        file,
        exchange: selectedExchange,
      })

      const successResult: UploadResult = {
        response,
        exchange: selectedExchange,
        fileName: file.name,
        finishedAt: new Date().toISOString(),
      }

      setLastSuccess(successResult)
      notifications.success('CSV processed successfully', `${selectedExchange.toUpperCase()}: ${response.status}`)
      setFile(null)
    } catch (error) {
      const message = toErrorMessage(error, 'CSV upload failed.')
      setUploadError(message)
      notifications.error('CSV upload failed', message)
    } finally {
      setIsUploading(false)
    }
  }

  return (
    <section className="stack-lg">
      <PageHeader
        title="CSV Imports"
        description="Select an exchange and upload a CSV statement for synchronous ledger processing."
      />

      {exchangeListState === 'loading' ? <LoadingState label="Loading supported exchanges from ledger-svc..." /> : null}

      {exchangeListState === 'error' && exchangeError ? (
        <ErrorState message={exchangeError} actionLabel="Retry" onAction={() => void loadSupportedExchanges()} />
      ) : null}

      {exchangeListState === 'empty' ? (
        <EmptyState
          title="No exchanges available"
          description="Ledger-svc returned an empty supported exchanges list."
        />
      ) : null}

      {exchangeListState === 'ready' ? (
        <article className="card">
          <form onSubmit={handleSubmit} className="form-grid">
            <label>
              Exchange
              <select
                value={selectedExchangeId}
                onChange={(event) => setSelectedExchangeId(event.target.value)}
                disabled={isUploading}
                required
              >
                {exchanges.map((exchange) => (
                  <option key={exchange} value={exchange}>
                    {exchange.toUpperCase()}
                  </option>
                ))}
              </select>
            </label>

            <label>
              CSV file
              <input
                key={lastSuccess ? `uploaded-${lastSuccess.finishedAt}` : 'upload-file'}
                type="file"
                accept=".csv,text/csv"
                onChange={(event) => setFile(event.target.files?.[0] ?? null)}
                disabled={isUploading}
                required
              />
            </label>

            <p className="hint-text">Accepted format: CSV file uploaded as multipart form field `file`.</p>

            <div className="actions-row">
              <button type="submit" className="btn-primary" disabled={isUploading || !file || !selectedExchange}>
                {isUploading ? 'Processing...' : 'Upload and process'}
              </button>
            </div>
          </form>

          {isUploading ? (
            <div className="upload-progress" role="status" aria-live="polite">
              <span className="state-spinner" aria-hidden="true" />
              <p>
                Processing in progress. Ledger import may take several seconds. Please wait and do not close this page.
              </p>
            </div>
          ) : null}
        </article>
      ) : null}

      {uploadError ? <ErrorState message={uploadError} /> : null}

      {lastSuccess ? (
        <article className="card success-card">
          <h3>Upload completed</h3>
          <dl className="details-grid">
            <dt>Exchange</dt>
            <dd>{lastSuccess.exchange.toUpperCase()}</dd>
            <dt>File</dt>
            <dd>{lastSuccess.fileName}</dd>
            <dt>Status</dt>
            <dd>{lastSuccess.response.status}</dd>
            <dt>Finished at</dt>
            <dd>{formatDateTime(lastSuccess.finishedAt)}</dd>
          </dl>
        </article>
      ) : null}
    </section>
  )
}
