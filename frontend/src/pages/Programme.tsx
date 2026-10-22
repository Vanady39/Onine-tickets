import { Link, useSearchParams } from 'react-router-dom';
import { api, EVENT_TYPES, type EventType } from '../api';
import { Empty, ErrorBox, Loading, Poster, TypeBadge } from '../components/ui';
import { EVENT_TYPE_LABEL, formatDateTime, formatDuration, formatMoney } from '../format';
import { useAsync, useTitle } from '../hooks';

function isEventType(v: string | null): v is EventType {
  return !!v && (EVENT_TYPES as string[]).includes(v);
}

/** Programme (UC01, UC02): upcoming events filtered by type and date range. */
export function Programme() {
  useTitle('Programme');
  const [params, setParams] = useSearchParams();
  const typeParam = params.get('type');
  const type: EventType | '' = isEventType(typeParam) ? typeParam : '';
  const from = params.get('from') ?? '';
  const to = params.get('to') ?? '';
  const rangeInvalid = !!from && !!to && from > to;

  const { data, error, loading, reload } = useAsync(
    () => (rangeInvalid ? Promise.resolve([]) : api.events({ type, from, to })),
    [type, from, to, rangeInvalid],
  );

  const update = (key: string, value: string) => {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    setParams(next, { replace: true });
  };

  const filtered = !!(type || from || to);

  return (
    <div className="container section">
      <section className="hero">
        <h1>What’s on</h1>
        <p>Concerts, cinema, theatre and exhibitions — pick a seat and get your e-ticket in minutes.</p>
      </section>

      <div className="filters" role="search">
        <div className="chips" role="group" aria-label="Event type">
          <button type="button" className={`chip${type === '' ? ' active' : ''}`} aria-pressed={type === ''} onClick={() => update('type', '')}>
            All
          </button>
          {EVENT_TYPES.map((t) => (
            <button
              key={t}
              type="button"
              className={`chip chip-${t}${type === t ? ' active' : ''}`}
              aria-pressed={type === t}
              onClick={() => update('type', t)}
            >
              {EVENT_TYPE_LABEL[t]}
            </button>
          ))}
        </div>
        <div className="date-range">
          <label>
            <span>From</span>
            <input type="date" value={from} max={to || undefined} onChange={(e) => update('from', e.target.value)} />
          </label>
          <label>
            <span>To</span>
            <input type="date" value={to} min={from || undefined} onChange={(e) => update('to', e.target.value)} />
          </label>
          {filtered && (
            <button type="button" className="btn btn-small btn-ghost" onClick={() => setParams({}, { replace: true })}>
              Reset
            </button>
          )}
        </div>
      </div>

      {rangeInvalid && <ErrorBox message="The start date must not be after the end date." />}
      {error && <ErrorBox message={error} onRetry={reload} />}
      {loading && !data && <Loading label="Loading the programme…" />}

      {data && !rangeInvalid && (
        data.length === 0 ? (
          <Empty title="No events found">
            <p>{filtered ? 'Try another type or a wider date range.' : 'New events will appear here soon.'}</p>
          </Empty>
        ) : (
          <ul className={`event-grid${loading ? ' is-refreshing' : ''}`}>
            {data.map((e) => (
              <li key={e.id}>
                <Link to={`/events/${e.id}`} className={`event-card type-edge-${e.type}`}>
                  <Poster type={e.type} title={e.title} imageUrl={e.imageUrl} />
                  <div className="event-card-body">
                    <div className="event-card-meta">
                      <TypeBadge type={e.type} />
                      {e.ageRating && <span className="age">{e.ageRating}</span>}
                      {e.durationMin > 0 && <span className="muted small">{formatDuration(e.durationMin)}</span>}
                    </div>
                    <h2>{e.title}</h2>
                    <p className="event-card-desc">{e.description}</p>
                    <div className="event-card-foot">
                      <span>
                        <span className="muted small">Next</span>
                        <br />
                        <strong>{e.nextSessionAt ? formatDateTime(e.nextSessionAt) : 'No sessions yet'}</strong>
                      </span>
                      {e.minPrice !== undefined && e.minPrice !== null && (
                        <span className="price-from">
                          from <strong>{formatMoney(e.minPrice)}</strong>
                        </span>
                      )}
                    </div>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        )
      )}
    </div>
  );
}
