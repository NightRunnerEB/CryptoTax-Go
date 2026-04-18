interface EmptyStateProps {
  title: string
  description?: string
}

export function EmptyState({ title, description }: EmptyStateProps) {
  return (
    <div className="state-card state-card-quiet" role="status" aria-live="polite">
      <div className="state-mark upload-hero-mark" aria-hidden="true">•</div>
      <h3>{title}</h3>
      {description ? <p>{description}</p> : null}
    </div>
  )
}
