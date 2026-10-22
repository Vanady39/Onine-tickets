import { api } from '../../api';
import { Empty, ErrorBox, Loading } from '../../components/ui';
import { EVENT_TYPE_LABEL, formatDateTime, formatMoney } from '../../format';
import { useAsync } from '../../hooks';

/** Sales report (UC27): tickets sold, revenue and occupancy per session and per event. */
export function Reports({ eventId, onSelect }: { eventId?: number; onSelect: (id: number) => void }) {
  const events = useAsync(() => api.adminEvents(), []);
  const selected = eventId ?? events.data?.[0]?.id;
  const report = useAsync(() => (selected ? api.report(selected) : Promise.resolve(undefined)), [selected]);

  if (events.loading && !events.data) return <Loading />;
  if (events.error) return <ErrorBox message={events.error} onRetry={events.reload} />;
  if (!events.data?.length) return <Empty title="No events yet" />;

  const r = report.data;
  const sessions = r?.sessions ?? [];
  const capacity = sessions.reduce((s, x) => s + x.capacity, 0);
  const occupancy = capacity ? (r!.ticketsSold / capacity) * 100 : 0;

  return (
    <div className="stack-lg">
      <div className="toolbar">
        <label className="field inline-field">
          <span>Event</span>
          <select value={selected} onChange={(e) => onSelect(Number(e.target.value))}>
            {events.data.map((e) => (
              <option key={e.id} value={e.id}>
                {e.title} ({EVENT_TYPE_LABEL[e.type]})
              </option>
            ))}
          </select>
        </label>
        <button type="button" className="btn btn-small btn-ghost" onClick={report.reload} disabled={report.loading}>
          {report.loading ? 'Refreshing…' : 'Refresh'}
        </button>
      </div>

      {report.error && <ErrorBox message={report.error} onRetry={report.reload} />}
      {report.loading && !r && <Loading label="Building the report…" />}
      {r && (
        <>
          <div className="stats">
            <div className="stat">
              <span className="stat-label">Tickets sold</span>
              <span className="stat-value">{r.ticketsSold}</span>
            </div>
            <div className="stat">
              <span className="stat-label">Revenue</span>
              <span className="stat-value">{formatMoney(r.revenue)}</span>
            </div>
            <div className="stat">
              <span className="stat-label">Occupancy</span>
              <span className="stat-value">{occupancy.toFixed(1)}%</span>
            </div>
            <div className="stat">
              <span className="stat-label">Sessions</span>
              <span className="stat-value">{sessions.length}</span>
            </div>
          </div>
          {sessions.length === 0 ? (
            <Empty title="No sessions scheduled for this event" />
          ) : (
            <div className="table-wrap">
              <table className="table report-table">
                <thead>
                  <tr>
                    <th>Session</th>
                    <th>Venue</th>
                    <th className="num">Sold</th>
                    <th className="num">Revenue</th>
                    <th>Occupancy</th>
                  </tr>
                </thead>
                <tbody>
                  {sessions.map((s) => (
                    <tr key={s.sessionId}>
                      <td className="nowrap">{formatDateTime(s.startsAt)}</td>
                      <td>
                        {s.venueName}
                        <span className="muted small block">{s.hallName}</span>
                      </td>
                      <td className="num nowrap">
                        {s.ticketsSold} / {s.capacity}
                      </td>
                      <td className="num nowrap">{formatMoney(s.revenue)}</td>
                      <td>
                        <div className="bar" role="img" aria-label={`${s.occupancy.toFixed(1)}% occupied`}>
                          <span style={{ width: `${Math.min(100, s.occupancy)}%` }} />
                        </div>
                        <span className="small">{s.occupancy.toFixed(1)}%</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          <p className="muted small">Only paid tickets are counted; refunded orders are excluded.</p>
        </>
      )}
    </div>
  );
}
