/* eslint-disable react-refresh/only-export-components */
import {
  createContext,
  type PropsWithChildren,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
} from 'react'

export type NotificationVariant = 'success' | 'error' | 'info'

interface NotificationItem {
  id: number
  title: string
  description?: string
  variant: NotificationVariant
}

interface NotifyInput {
  title: string
  description?: string
  variant?: NotificationVariant
  durationMs?: number
}

interface NotificationContextValue {
  notify: (input: NotifyInput) => void
  success: (title: string, description?: string) => void
  error: (title: string, description?: string) => void
  info: (title: string, description?: string) => void
}

const NotificationContext = createContext<NotificationContextValue | undefined>(undefined)

export function NotificationProvider({ children }: PropsWithChildren) {
  const [items, setItems] = useState<NotificationItem[]>([])
  const nextId = useRef(1)

  const remove = useCallback((id: number) => {
    setItems((prev) => prev.filter((item) => item.id !== id))
  }, [])

  const notify = useCallback(
    (input: NotifyInput) => {
      const id = nextId.current
      nextId.current += 1

      const variant = input.variant ?? 'info'
      const item: NotificationItem = {
        id,
        title: input.title,
        description: input.description,
        variant,
      }

      setItems((prev) => [item, ...prev].slice(0, 5))

      const timeoutMs = input.durationMs ?? 4200
      window.setTimeout(() => remove(id), timeoutMs)
    },
    [remove],
  )

  const value = useMemo<NotificationContextValue>(() => {
    return {
      notify,
      success: (title, description) => notify({ title, description, variant: 'success' }),
      error: (title, description) => notify({ title, description, variant: 'error' }),
      info: (title, description) => notify({ title, description, variant: 'info' }),
    }
  }, [notify])

  return (
    <NotificationContext.Provider value={value}>
      {children}
      <div className="toast-region" role="status" aria-live="polite">
        {items.map((item) => (
          <article key={item.id} className={`toast toast-${item.variant}`}>
            <div className="toast-head">
              <strong>{item.title}</strong>
              <button type="button" onClick={() => remove(item.id)} aria-label="Dismiss notification">
                ×
              </button>
            </div>
            {item.description ? <p>{item.description}</p> : null}
          </article>
        ))}
      </div>
    </NotificationContext.Provider>
  )
}

export function useNotifications(): NotificationContextValue {
  const context = useContext(NotificationContext)

  if (!context) {
    throw new Error('useNotifications must be used within NotificationProvider')
  }

  return context
}
