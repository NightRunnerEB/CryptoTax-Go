interface LoadingStateProps {
  label?: string
}

export function LoadingState({ label = 'Loading data...' }: LoadingStateProps) {
  return (
    <div className="state-card" role="status" aria-live="polite">
      <div className="state-spinner" aria-hidden="true" />
      <p>{label}</p>
    </div>
  )
}
