import { useMemo, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { api, errorMessage, type AddOnKind, type Event, type Hall, type NewSession, type Venue } from '../../api';
import { ErrorBox, Loading, Notice } from '../../components/ui';
import { ADD_ON_KIND_LABEL, EVENT_TYPE_LABEL, formatDateTime, localInputToRFC3339, roublesToKopecks } from '../../format';
import { useAsync } from '../../hooks';

interface AddOnRow {
  kind: AddOnKind;
  name: string;
  price: string;
  capacity: string;
}

const ADD_ON_KINDS: AddOnKind[] = ['business_lunch', 'masterclass', 'presentation'];

/** Schedule session (UC25): hall, date/time and prices per zone, or base price and add-ons. */
export function SessionForm({ initialEventId }: { initialEventId?: number }) {
  const { data, error, loading, reload } = useAsync(async () => {
    const [events, venues] = await Promise.all([api.adminEvents(), api.venues()]);
    return { events, venues };
  }, []);

  if (loading && !data) return <Loading />;
  if (error && !data) return <ErrorBox message={error} onRetry={reload} />;
  if (!data) return null;
  return <Form events={data.events} venues={data.venues} initialEventId={initialEventId} />;
}

function Form({ events, venues, initialEventId }: { events: Event[]; venues: Venue[]; initialEventId?: number }) {
  const halls = useMemo(() => venues.flatMap((v) => (v.halls ?? []).map((h) => ({ ...h, venueName: v.name }))), [venues]);
  const [eventId, setEventId] = useState<number>(
    initialEventId && events.some((e) => e.id === initialEventId) ? initialEventId : events[0]?.id ?? 0,
  );
  const [hallId, setHallId] = useState<number>(0);
  const [startsAt, setStartsAt] = useState('');
  const [zonePrices, setZonePrices] = useState<Record<number, string>>({});
  const [basePrice, setBasePrice] = useState('');
  const [addOns, setAddOns] = useState<AddOnRow[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [created, setCreated] = useState<{ id: number; label: string } | null>(null);

  const event = events.find((e) => e.id === eventId);
  const hall: Hall | undefined = halls.find((h) => h.id === hallId);
  const seated = hall?.kind === 'seated';
  const exhibition = event?.type === 'exhibition';

  const updateAddOn = (i: number, patch: Partial<AddOnRow>) => setAddOns((rows) => rows.map((r, j) => (j === i ? { ...r, ...patch } : r)));

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setCreated(null);
    if (!event || !hall) {
      setError('Choose an event and a hall.');
      return;
    }
    const when = localInputToRFC3339(startsAt);
    if (!when) {
      setError('Enter the date and start time.');
      return;
    }
    if (new Date(when).getTime() <= Date.now()) {
      setError('The session must start in the future.');
      return;
    }
    const body: NewSession = { eventId: event.id, hallId: hall.id, startsAt: when, basePrice: 0, prices: [], addOns: [] };
    if (seated) {
      for (const z of hall.zones ?? []) {
        const price = roublesToKopecks(zonePrices[z.id] ?? '');
        if (!(price > 0)) {
          setError(`Enter a positive price for zone “${z.name}”.`);
          return;
        }
        body.prices.push({ zoneId: z.id, price });
      }
    } else {
      body.basePrice = roublesToKopecks(basePrice);
      if (!(body.basePrice > 0)) {
        setError('Enter a positive ticket price.');
        return;
      }
      if (exhibition) {
        for (const a of addOns) {
          const price = roublesToKopecks(a.price);
          const capacity = Number(a.capacity);
          if (!a.name.trim() || !(price >= 0) || !Number.isInteger(capacity) || capacity <= 0) {
            setError('Each add-on needs a name, a price of 0 or more, and a positive capacity.');
            return;
          }
          body.addOns.push({ kind: a.kind, name: a.name.trim(), price, capacity });
        }
      }
    }
    setBusy(true);
    try {
      const res = await api.createSession(body);
      setCreated({ id: res.id, label: `${event.title}, ${formatDateTime(when)} — ${hall.venueName}, ${hall.name}` });
      setStartsAt('');
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  if (events.length === 0) return <Notice kind="info">Create an event first.</Notice>;

  return (
    <form className="panel form-grid" onSubmit={submit}>
      <h2 className="span-all">Schedule a session</h2>
      <label className="field">
        <span>Event</span>
        <select value={eventId} onChange={(e) => setEventId(Number(e.target.value))} required>
          {events.map((e) => (
            <option key={e.id} value={e.id}>
              {e.title} ({EVENT_TYPE_LABEL[e.type]})
            </option>
          ))}
        </select>
      </label>
      <label className="field">
        <span>Venue and hall</span>
        <select
          value={hallId}
          onChange={(e) => {
            setHallId(Number(e.target.value));
            setZonePrices({});
          }}
          required
        >
          <option value={0} disabled>
            Choose a hall…
          </option>
          {venues.map((v) => (
            <optgroup key={v.id} label={v.name}>
              {(v.halls ?? []).map((h) => (
                <option key={h.id} value={h.id}>
                  {h.name} — {h.kind === 'seated' ? `${h.capacity} seats` : `general admission, ${h.capacity} places`}
                </option>
              ))}
            </optgroup>
          ))}
        </select>
      </label>
      <label className="field">
        <span>Date and start time</span>
        <input type="datetime-local" value={startsAt} onChange={(e) => setStartsAt(e.target.value)} required />
        <small className="muted">In your local time zone.</small>
      </label>

      {hall && seated && (
        <fieldset className="span-all subform">
          <legend>Prices per zone, ₽</legend>
          <div className="form-grid">
            {(hall.zones ?? []).map((z) => (
              <label key={z.id} className="field">
                <span>{z.name}</span>
                <input
                  type="number"
                  min={1}
                  step="0.01"
                  inputMode="decimal"
                  value={zonePrices[z.id] ?? ''}
                  onChange={(e) => setZonePrices((p) => ({ ...p, [z.id]: e.target.value }))}
                  required
                />
              </label>
            ))}
          </div>
        </fieldset>
      )}

      {hall && !seated && (
        <fieldset className="span-all subform">
          <legend>General admission</legend>
          <label className="field">
            <span>Ticket price, ₽</span>
            <input type="number" min={1} step="0.01" inputMode="decimal" value={basePrice} onChange={(e) => setBasePrice(e.target.value)} required />
          </label>
          {exhibition ? (
            <div className="stack">
              <h3 className="small-title">Add-on services</h3>
              {addOns.length === 0 && <p className="muted small">No add-ons. Business lunches, masterclasses and presentations are optional.</p>}
              {addOns.map((a, i) => (
                <div key={i} className="addon-row">
                  <label className="field">
                    <span>Kind</span>
                    <select value={a.kind} onChange={(e) => updateAddOn(i, { kind: e.target.value as AddOnKind })}>
                      {ADD_ON_KINDS.map((k) => (
                        <option key={k} value={k}>
                          {ADD_ON_KIND_LABEL[k]}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="field grow">
                    <span>Name</span>
                    <input value={a.name} onChange={(e) => updateAddOn(i, { name: e.target.value })} required maxLength={200} />
                  </label>
                  <label className="field">
                    <span>Price, ₽</span>
                    <input type="number" min={0} step="0.01" value={a.price} onChange={(e) => updateAddOn(i, { price: e.target.value })} required />
                  </label>
                  <label className="field">
                    <span>Capacity</span>
                    <input type="number" min={1} step={1} value={a.capacity} onChange={(e) => updateAddOn(i, { capacity: e.target.value })} required />
                  </label>
                  <button type="button" className="icon-btn addon-remove" aria-label="Remove add-on" onClick={() => setAddOns(addOns.filter((_, j) => j !== i))}>
                    ×
                  </button>
                </div>
              ))}
              <div>
                <button
                  type="button"
                  className="btn btn-small btn-secondary"
                  onClick={() => setAddOns([...addOns, { kind: 'business_lunch', name: '', price: '', capacity: '' }])}
                >
                  + Add-on service
                </button>
              </div>
            </div>
          ) : (
            <p className="muted small">Add-on services are available for exhibitions only.</p>
          )}
        </fieldset>
      )}

      {error && (
        <div className="span-all">
          <ErrorBox message={error} />
        </div>
      )}
      {created && (
        <div className="span-all">
          <Notice kind="success">
            Session scheduled: {created.label}. <Link to={`/sessions/${created.id}`}>Open the session page</Link>
          </Notice>
        </div>
      )}
      <div className="span-all">
        <button type="submit" className="btn btn-primary" disabled={busy}>
          {busy ? 'Scheduling…' : 'Schedule session'}
        </button>
      </div>
    </form>
  );
}
