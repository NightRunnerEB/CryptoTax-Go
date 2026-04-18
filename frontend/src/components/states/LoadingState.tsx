interface LoadingStateProps {
  label?: string
}

export function LoadingState({ label = 'Loading data...' }: LoadingStateProps) {
  return (
    <div className="state-card state-card-quiet" role="status" aria-live="polite">
      <div className="state-mark upload-hero-mark" aria-hidden="true">
        <span className="state-spinner" />
      </div>
      <h3>Loading</h3>
      <p>{label}</p>
    </div>
  )
}
