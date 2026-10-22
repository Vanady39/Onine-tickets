import { Link, useParams } from 'react-router-dom';
import { api, type Session } from '../api';
import { ErrorBox, Loading, Poster, TypeBadge, Empty } from '../components/ui';
import { formatDate, formatDuration, formatMoney, formatTime, plural } from '../format';
import { useAsync, useTitle } from '../hooks';

export function priceRange(s: Session): string {
  if (s.hall.kind === 'general_admission') return formatMoney(s.basePrice);
  const prices = (s.prices ?? []).map((p) => p.price);
  if (!prices.length) return '—';
  const min = Math.min(...prices);
  const max = Math.max(...prices);
  return min === max ? formatMoney(min) : `${formatMoney(min)} – ${formatMoney(max)}`;
}

/** Event page (UC03): description and the schedule with venues and prices. */
export function EventPage() {
  const id = Number(useParams().id);
  const { data: event, error, loading, reload } = useAsync(() => api.event(id), [id]);
  useTitle(event?.title);

  if (loading && !event) return <Loading />;
  if (error) {
    return (
      <div className="container section">
        <ErrorBox message={error === 'not found' ? 'This event does not exist.' : error} onRetry={reload} />
        <Link to="/">← Back to the programme</Link>
      </div>
    );
  }
  if (!event) return null;

  const now = Date.now();
  const sessions = [...(event.sessions ?? [])].sort((a, b) => a.startsAt.localeCompare(b.startsAt));
  const upcoming = sessions.filter((s) => new Date(s.startsAt).getTime() > now);

  return (
    <div className="container section">
      <Link className="back-link" to="/">
        ← Programme
      </Link>
      <article className={`event-hero type-edge-${event.type}`}>
        <Poster type={event.type} title={event.title} imageUrl={event.imageUrl} size="hero" />
        <div className="event-hero-body">
          <div className="event-card-meta">
            <TypeBadge type={event.type} />
            {event.ageRating && <span className="age">{event.ageRating}</span>}
          </div>
          <h1>{event.title}</h1>
          <p className="lead">{event.description}</p>
          <dl className="facts">
            {event.durationMin > 0 && (
              <div>
                <dt>Duration</dt>
                <dd>{formatDuration(event.durationMin)}</dd>
              </div>
            )}
            <div>
              <dt>Age rating</dt>
              <dd>{event.ageRating || '—'}</dd>
            </div>
            <div>
              <dt>Sessions</dt>
              <dd>{upcoming.length}</dd>
            </div>
          </dl>
        </div>
      </article>

      <h2 className="section-title">Schedule</h2>
      {upcoming.length === 0 ? (
        <Empty title="No upcoming sessions">
          <p>Check back later — new dates are added regularly.</p>
        </Empty>
      ) : (
        <ul className="session-list">
          {upcoming.map((s) => {
            const ga = s.hall.kind === 'general_admission';
            const soldOut = s.remaining <= 0;
            return (
              <li key={s.id} className="session-row">
                <div className="session-when">
                  <span className="session-date">{formatDate(s.startsAt)}</span>
                  <span className="session-time">{formatTime(s.startsAt)}</span>
                </div>
                <div className="session-where">
                  <strong>{s.hall.venueName}</strong>
                  <span className="muted">
                    {s.hall.name} · {s.hall.address}
                  </span>
                </div>
                <div className="session-price">
                  <strong>{priceRange(s)}</strong>
                  <span className={`muted small${s.remaining < 10 && !soldOut ? ' warn-text' : ''}`}>
                    {soldOut ? 'Sold out' : `${plural(s.remaining, ga ? 'place' : 'seat')} left`}
                  </span>
                </div>
                <div className="session-cta">
                  {soldOut ? (
                    <span className="btn btn-disabled" aria-disabled="true">
                      Sold out
                    </span>
                  ) : (
                    <Link className="btn btn-primary" to={`/sessions/${s.id}`}>
                      {ga ? 'Get tickets' : 'Choose seats'}
                    </Link>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
