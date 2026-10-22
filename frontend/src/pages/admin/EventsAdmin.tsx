import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { api, errorMessage, EVENT_TYPES, type Event, type EventInput, type EventType } from '../../api';
import { Empty, ErrorBox, Loading, Notice, TypeBadge } from '../../components/ui';
import { EVENT_TYPE_LABEL, formatDuration } from '../../format';
import { useAsync } from '../../hooks';

const EMPTY: EventInput = { title: '', type: 'concert', description: '', durationMin: 120, ageRating: '0+', imageUrl: '' };
const AGE_RATINGS = ['0+', '6+', '12+', '16+', '18+'];

/** Events (UC24): list, create and edit events. */
export function EventsAdmin({ onSchedule, onReport }: { onSchedule: (id: number) => void; onReport: (id: number) => void }) {
  const { data: events, error, loading, reload } = useAsync(() => api.adminEvents(), []);
  const [editing, setEditing] = useState<Event | 'new' | null>(null);
  const [saved, setSaved] = useState<string | null>(null);

  return (
    <div className="stack-lg">
      {saved && <Notice kind="success">{saved}</Notice>}
      {editing ? (
        <EventForm
          event={editing === 'new' ? null : editing}
          onCancel={() => setEditing(null)}
          onSaved={(e, created) => {
            setEditing(null);
            setSaved(created ? `Event “${e.title}” created. Schedule its first session next.` : `Event “${e.title}” saved.`);
            reload();
          }}
        />
      ) : (
        <div className="toolbar">
          <h2>Events</h2>
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => {
              setSaved(null);
              setEditing('new');
            }}
          >
            New event
          </button>
        </div>
      )}

      {error && <ErrorBox message={error} onRetry={reload} />}
      {loading && !events && <Loading />}
      {events &&
        (events.length === 0 ? (
          <Empty title="No events yet">
            <p>Create the first event.</p>
          </Empty>
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Event</th>
                  <th>Type</th>
                  <th>Duration</th>
                  <th>Age</th>
                  <th aria-label="Actions" />
                </tr>
              </thead>
              <tbody>
                {events.map((e) => (
                  <tr key={e.id}>
                    <td>
                      <Link to={`/events/${e.id}`}>
                        <strong>{e.title}</strong>
                      </Link>
                      <span className="muted small block clamp-1">{e.description}</span>
                    </td>
                    <td>
                      <TypeBadge type={e.type} />
                    </td>
                    <td className="nowrap">{formatDuration(e.durationMin) || '—'}</td>
                    <td>{e.ageRating}</td>
                    <td>
                      <div className="btn-row nowrap">
                        <button
                          type="button"
                          className="btn btn-small btn-ghost"
                          onClick={() => {
                            setSaved(null);
                            setEditing(e);
                            window.scrollTo({ top: 0, behavior: 'smooth' });
                          }}
                        >
                          Edit
                        </button>
                        <button type="button" className="btn btn-small btn-ghost" onClick={() => onSchedule(e.id)}>
                          Schedule
                        </button>
                        <button type="button" className="btn btn-small btn-ghost" onClick={() => onReport(e.id)}>
                          Report
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ))}
    </div>
  );
}

function EventForm({ event, onCancel, onSaved }: { event: Event | null; onCancel: () => void; onSaved: (e: Event, created: boolean) => void }) {
  const [form, setForm] = useState<EventInput>(
    event
      ? {
          title: event.title,
          type: event.type,
          description: event.description,
          durationMin: event.durationMin,
          ageRating: event.ageRating,
          imageUrl: event.imageUrl,
        }
      : EMPTY,
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const set = <K extends keyof EventInput>(k: K, v: EventInput[K]) => setForm((f) => ({ ...f, [k]: v }));

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    const body: EventInput = { ...form, title: form.title.trim(), description: form.description.trim(), imageUrl: form.imageUrl.trim() };
    try {
      const saved = event ? await api.updateEvent(event.id, body) : await api.createEvent(body);
      onSaved(saved, !event);
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  return (
    <form className="panel form-grid" onSubmit={submit}>
      <h2 className="span-all">{event ? `Edit “${event.title}”` : 'New event'}</h2>
      <label className="field span-all">
        <span>Title</span>
        <input value={form.title} onChange={(e) => set('title', e.target.value)} required maxLength={200} />
      </label>
      <label className="field">
        <span>Type</span>
        <select value={form.type} onChange={(e) => set('type', e.target.value as EventType)}>
          {EVENT_TYPES.map((t) => (
            <option key={t} value={t}>
              {EVENT_TYPE_LABEL[t]}
            </option>
          ))}
        </select>
      </label>
      <label className="field">
        <span>Duration, minutes</span>
        <input
          type="number"
          min={0}
          max={1440}
          value={form.durationMin}
          onChange={(e) => set('durationMin', Math.max(0, Math.floor(Number(e.target.value) || 0)))}
        />
        <small className="muted">0 for exhibitions without a fixed duration.</small>
      </label>
      <label className="field">
        <span>Age rating</span>
        <select value={form.ageRating} onChange={(e) => set('ageRating', e.target.value)}>
          {(AGE_RATINGS.includes(form.ageRating) ? AGE_RATINGS : [form.ageRating, ...AGE_RATINGS]).map((a) => (
            <option key={a} value={a}>
              {a}
            </option>
          ))}
        </select>
      </label>
      <label className="field">
        <span>Image URL (optional)</span>
        <input type="url" value={form.imageUrl} onChange={(e) => set('imageUrl', e.target.value)} placeholder="https://…" />
      </label>
      <label className="field span-all">
        <span>Description</span>
        <textarea rows={4} value={form.description} onChange={(e) => set('description', e.target.value)} maxLength={4000} />
      </label>
      {error && (
        <div className="span-all">
          <ErrorBox message={error} />
        </div>
      )}
      <div className="btn-row span-all">
        <button type="submit" className="btn btn-primary" disabled={busy}>
          {busy ? 'Saving…' : event ? 'Save changes' : 'Create event'}
        </button>
        <button type="button" className="btn btn-ghost" onClick={onCancel} disabled={busy}>
          Cancel
        </button>
      </div>
    </form>
  );
}
