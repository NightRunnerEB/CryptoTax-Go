import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ChangeEvent } from 'react'
import { AlertCircle, CheckCircle2, FileSpreadsheet, LoaderCircle, RefreshCw, Upload } from 'lucide-react'
import { listSupportedExchanges, uploadExchangeCsv } from '../api/ledgerService'
import { useAuth } from '../auth/AuthContext'
import { useNotifications } from '../components/notifications/NotificationProvider'
import { toErrorMessage } from '../utils/errors'

type ExchangeListState = 'idle' | 'loading' | 'ready' | 'empty' | 'error'
type UploadState = 'idle' | 'uploading' | 'success' | 'error'
type UploadStatus = 'success' | 'failed'

interface UploadResult {
  id: string
  exchange: string
  fileName: string
  status: UploadStatus
  timestamp: string
}

function formatExchangeLabel(value: string): string {
  if (!value) {
    return value
  }

  return value
    .split(/[_\s-]+/)
    .filter(Boolean)
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1).toLowerCase())
    .join(' ')
}

function formatTimestamp(value: Date): string {
  const year = value.getFullYear()
  const month = String(value.getMonth() + 1).padStart(2, '0')
  const day = String(value.getDate()).padStart(2, '0')
  const hours = String(value.getHours()).padStart(2, '0')
  const minutes = String(value.getMinutes()).padStart(2, '0')
  const seconds = String(value.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

export function CSVImports() {
  const { session } = useAuth()
  const notifications = useNotifications()

  const [exchanges, setExchanges] = useState<string[]>([])
  const [exchangeListState, setExchangeListState] = useState<ExchangeListState>('idle')
  const [exchangeError, setExchangeError] = useState<string | null>(null)

  const [selectedExchange, setSelectedExchange] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [uploadState, setUploadState] = useState<UploadState>('idle')
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [uploadHistory, setUploadHistory] = useState<UploadResult[]>([])

  const canUpload = selectedExchange !== '' && selectedFile !== null && uploadState !== 'uploading'

  const loadSupportedExchanges = useCallback(async (): Promise<void> => {
    if (!session) {
      return
    }

    setExchangeListState('loading')
    setExchangeError(null)

    try {
      const result = await listSupportedExchanges()
      setExchanges(result)
      setSelectedExchange('')
      setExchangeListState(result.length > 0 ? 'ready' : 'empty')
    } catch (error) {
      setExchanges([])
      setSelectedExchange('')
      setExchangeError(toErrorMessage(error, 'Unable to load supported exchanges.'))
      setExchangeListState('error')
    }
  }, [session])

  useEffect(() => {
    void loadSupportedExchanges()
  }, [loadSupportedExchanges])

  const handleFileSelect = (event: ChangeEvent<HTMLInputElement>): void => {
    if (event.target.files && event.target.files[0]) {
      setSelectedFile(event.target.files[0])
      setUploadError(null)
      if (uploadState !== 'uploading') {
        setUploadState('idle')
      }
      return
    }

    setSelectedFile(null)
  }

  const handleUpload = async (): Promise<void> => {
    if (!session || !selectedExchange || !selectedFile) {
      return
    }

    setUploadState('uploading')
    setUploadError(null)

    try {
      await uploadExchangeCsv({
        file: selectedFile,
        exchange: selectedExchange,
      })

      const newUpload: UploadResult = {
        id: `${Date.now()}`,
        exchange: formatExchangeLabel(selectedExchange),
        fileName: selectedFile.name,
        status: 'success',
        timestamp: formatTimestamp(new Date()),
      }

      setUploadHistory((prev) => [newUpload, ...prev].slice(0, 5))
      setUploadState('success')
      notifications.success('CSV processed successfully', `${formatExchangeLabel(selectedExchange)}: success`)

      window.setTimeout(() => {
        setUploadState('idle')
        setSelectedFile(null)
        setSelectedExchange('')
      }, 2000)
    } catch (error) {
      const message = toErrorMessage(error, 'CSV upload failed.')
      setUploadState('error')
      setUploadError(message)
      notifications.error('CSV upload failed', message)
    }
  }

  const uploadButtonLabel = useMemo(() => {
    if (uploadState === 'uploading') {
      return 'Uploading...'
    }
    if (uploadState === 'success') {
      return 'Uploaded Successfully'
    }
    return 'Upload Statement'
  }, [uploadState])

  return (
    <div className="max-w-5xl">
      <div className="mb-8 flex flex-col gap-2">
        <h2 className="text-foreground">CSV Imports</h2>
        <p className="text-muted-foreground text-sm">Upload exchange statements to import and normalize transaction data</p>
      </div>

      <div className="bg-surface rounded-xl border border-border p-8 mb-8" style={{ boxShadow: 'var(--shadow-md)' }}>
        <h3 className="text-foreground mb-0">Upload Statement</h3>

        <div className="space-y-6 mt-5">
          <div>
            <label className="block text-foreground mb-2" htmlFor="csv-imports-exchange">
              Exchange
            </label>
            <select
              id="csv-imports-exchange"
              value={selectedExchange}
              onChange={(event) => setSelectedExchange(event.target.value)}
              disabled={exchangeListState !== 'ready' || uploadState === 'uploading'}
              className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all disabled:opacity-50"
            >
              <option value="">{exchangeListState === 'loading' ? 'Loading exchanges...' : 'Select exchange...'}</option>
              {exchanges.map((exchange) => (
                <option key={exchange} value={exchange}>
                  {formatExchangeLabel(exchange)}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-foreground mb-2" htmlFor="csv-file-upload">
              CSV File
            </label>
            <div className="relative">
              <input
                id="csv-file-upload"
                type="file"
                accept=".csv,text/csv"
                onChange={handleFileSelect}
                disabled={uploadState === 'uploading' || exchangeListState !== 'ready'}
                className="hidden"
              />
              <label
                htmlFor="csv-file-upload"
                className={`flex items-center justify-center gap-3 w-full px-6 py-12 bg-input-background border-2 border-dashed rounded-lg transition-all ${
                  uploadState === 'uploading' || exchangeListState !== 'ready'
                    ? 'border-input-border opacity-50 cursor-not-allowed'
                    : 'border-input-border cursor-pointer hover:border-primary hover:bg-surface-secondary'
                }`}
              >
                {selectedFile ? (
                  <>
                    <FileSpreadsheet className="w-6 h-6 text-primary" />
                    <span className="text-foreground font-medium">{selectedFile.name}</span>
                  </>
                ) : (
                  <>
                    <Upload className="w-6 h-6 text-muted-foreground" />
                    <span className="text-muted-foreground">Click to select a CSV file or drag and drop</span>
                  </>
                )}
              </label>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <button
              type="button"
              onClick={() => void handleUpload()}
              disabled={!canUpload}
              className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-lg px-6 py-2.5 bg-primary hover:bg-primary-dark text-primary-foreground text-sm font-medium transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {uploadState === 'uploading' ? (
                <>
                  <LoaderCircle className="w-4 h-4 animate-spin" />
                  {uploadButtonLabel}
                </>
              ) : uploadState === 'success' ? (
                <>
                  <CheckCircle2 className="w-4 h-4" />
                  {uploadButtonLabel}
                </>
              ) : (
                <>
                  <Upload className="w-4 h-4" />
                  {uploadButtonLabel}
                </>
              )}
            </button>

            {uploadState === 'uploading' ? (
              <div className="flex-1">
                <div className="h-2 bg-muted rounded-full overflow-hidden">
                  <div className="h-full bg-primary animate-pulse" style={{ width: '60%' }} />
                </div>
              </div>
            ) : null}
          </div>
        </div>
      </div>

      {exchangeListState === 'error' && exchangeError ? (
        <div className="bg-surface rounded-xl border border-[var(--status-failed)]/30 p-5 mb-6" style={{ boxShadow: 'var(--shadow-sm)' }}>
          <div className="flex items-start gap-3">
            <div className="w-10 h-10 rounded-lg bg-[var(--status-failed-bg)] flex items-center justify-center shrink-0">
              <AlertCircle className="w-5 h-5 text-[var(--status-failed)]" />
            </div>
            <div className="flex-1">
              <h4 className="text-foreground mb-1">Unable to load exchanges</h4>
              <p className="text-sm text-muted-foreground mb-4">{exchangeError}</p>
              <button
                type="button"
                className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-lg px-4 py-2 bg-surface-secondary hover:bg-muted text-foreground text-sm font-medium transition-all"
                onClick={() => void loadSupportedExchanges()}
              >
                <RefreshCw className="w-4 h-4" />
                Retry
              </button>
            </div>
          </div>
        </div>
      ) : null}

      {exchangeListState === 'empty' ? (
        <div className="bg-surface rounded-xl border border-border p-5 mb-6" style={{ boxShadow: 'var(--shadow-sm)' }}>
          <h4 className="text-foreground mb-1">No exchanges available</h4>
          <p className="text-sm text-muted-foreground">Ledger service returned an empty list of supported exchanges.</p>
        </div>
      ) : null}

      {uploadError ? (
        <div className="bg-surface rounded-xl border border-[var(--status-failed)]/30 p-5 mb-6" style={{ boxShadow: 'var(--shadow-sm)' }}>
          <div className="flex items-start gap-3">
            <div className="w-10 h-10 rounded-lg bg-[var(--status-failed-bg)] flex items-center justify-center shrink-0">
              <AlertCircle className="w-5 h-5 text-[var(--status-failed)]" />
            </div>
            <div>
              <h4 className="text-foreground mb-1">Upload failed</h4>
              <p className="text-sm text-muted-foreground">{uploadError}</p>
            </div>
          </div>
        </div>
      ) : null}

      <div>
        <h3 className="text-foreground">Recent Uploads</h3>

        {uploadHistory.length === 0 ? (
          <div className="mt-4 bg-surface rounded-xl border border-border p-12 text-center" style={{ boxShadow: 'var(--shadow-sm)' }}>
            <FileSpreadsheet className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
            <p className="text-muted-foreground">No uploads yet</p>
            <p className="text-sm text-muted-foreground mt-1">Upload your first exchange statement to get started</p>
          </div>
        ) : (
          <div className="mt-4 space-y-3">
            {uploadHistory.map((upload) => (
              <div
                key={upload.id}
                className="bg-surface rounded-lg border border-border p-5 flex items-center justify-between hover:border-primary/30 transition-all gap-4"
                style={{ boxShadow: 'var(--shadow-sm)' }}
              >
                <div className="flex items-center gap-4 min-w-0">
                  {upload.status === 'success' ? (
                    <div className="w-10 h-10 rounded-lg bg-[var(--status-success-bg)] flex items-center justify-center shrink-0">
                      <CheckCircle2 className="w-5 h-5 text-[var(--status-success)]" />
                    </div>
                  ) : (
                    <div className="w-10 h-10 rounded-lg bg-[var(--status-failed-bg)] flex items-center justify-center shrink-0">
                      <AlertCircle className="w-5 h-5 text-[var(--status-failed)]" />
                    </div>
                  )}

                  <div className="min-w-0">
                    <div className="flex items-center gap-3 mb-1">
                      <h4 className="text-foreground">{upload.exchange}</h4>
                      <span className="text-xs px-2 py-0.5 rounded-full bg-muted text-muted-foreground font-medium">{upload.status}</span>
                    </div>
                    <p className="text-sm text-muted-foreground truncate">{upload.fileName}</p>
                  </div>
                </div>

                <div className="text-right shrink-0">
                  <p className="text-sm text-muted-foreground">{upload.timestamp}</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
