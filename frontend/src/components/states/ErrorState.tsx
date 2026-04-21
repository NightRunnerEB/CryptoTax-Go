interface ErrorStateProps {
  title?: string
  message: string
  actionLabel?: string
  onAction?: () => void
}

export function ErrorState({ title = 'Something went wrong', message, actionLabel, onAction }: ErrorStateProps) {
  return (
    <div className="state-card state-card-error" role="alert">
      <div className="state-mark info-banner-mark" aria-hidden="true">!</div>
      <h3>{title}</h3>
      <p>{message}</p>
      {actionLabel && onAction ? (
        <button type="button" className="btn-secondary" onClick={onAction}>
          {actionLabel}
        </button>
      ) : null}
    </div>
  )
}
