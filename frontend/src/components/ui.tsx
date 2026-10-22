import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import type { EventType, OrderStatus } from '../api';
import { EVENT_TYPE_LABEL, ORDER_STATUS_LABEL } from '../format';

export function Loading({ label = 'Loading…' }: { label?: string }) {
  return (
    <div className="loading" role="status" aria-live="polite">
      <span className="spinner" aria-hidden="true" />
      {label}
    </div>
  );
}

export function ErrorBox({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="alert alert-error" role="alert">
      <span>{message}</span>
      {onRetry && (
        <button type="button" className="btn btn-small btn-ghost" onClick={onRetry}>
          Try again
        </button>
      )}
    </div>
  );
}

export function Notice({
  kind = 'info',
  children,
}: {
  kind?: 'info' | 'success' | 'warning' | 'error';
  children: ReactNode;
}) {
  return (
    <div className={`alert alert-${kind}`} role={kind === 'error' ? 'alert' : 'status'}>
      <div>{children}</div>
    </div>
  );
}

export function TypeBadge({ type }: { type: EventType }) {
  return <span className={`badge type-${type}`}>{EVENT_TYPE_LABEL[type]}</span>;
}

export function StatusBadge({ status }: { status: OrderStatus }) {
  return <span className={`badge status status-${status}`}>{ORDER_STATUS_LABEL[status]}</span>;
}

export function Empty({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="empty">
      <h3>{title}</h3>
      {children}
    </div>
  );
}

export function PageHeader({ title, subtitle, back, actions }: { title: string; subtitle?: ReactNode; back?: { to: string; label: string }; actions?: ReactNode }) {
  return (
    <div className="page-header">
      {back && (
        <Link className="back-link" to={back.to}>
          ← {back.label}
        </Link>
      )}
      <div className="page-header-row">
        <div>
          <h1>{title}</h1>
          {subtitle && <div className="page-subtitle">{subtitle}</div>}
        </div>
        {actions && <div className="page-actions">{actions}</div>}
      </div>
    </div>
  );
}

/** Poster is a typographic event cover: the event image if any, otherwise a type-coloured gradient. */
export function Poster({ type, title, imageUrl, size = 'card' }: { type: EventType; title: string; imageUrl?: string; size?: 'card' | 'hero' }) {
  return (
    <div className={`poster poster-${size} type-bg-${type}`} aria-hidden="true">
      {imageUrl ? (
        <img src={imageUrl} alt="" loading="lazy" />
      ) : (
        <span className="poster-mark">{title.trim().charAt(0).toUpperCase()}</span>
      )}
    </div>
  );
}
