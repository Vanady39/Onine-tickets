import { useState, type FormEvent } from 'react';
import { api, errorMessage, type HallKind, type HallSpec } from '../../api';
import { Empty, ErrorBox, Loading, Notice } from '../../components/ui';
import { plural } from '../../format';
import { useAsync } from '../../hooks';

/** Venues (UC26): halls with their seat maps; administrators can create venues. */
export function Venues({ canCreate }: { canCreate: boolean }) {
  const { data: venues, error, loading, reload } = useAsync(() => api.venues(), []);
  const [creating, setCreating] = useState(false);
  const [saved, setSaved] = useState<string | null>(null);

  return (
    <div className="stack-lg">
      {saved && <Notice kind="success">{saved}</Notice>}
      {creating ? (
        <VenueForm
          onCancel={() => setCreating(false)}
          onSaved={(name) => {
            setCreating(false);
            setSaved(`Venue “${name}” created.`);
            reload();
          }}
        />
      ) : (
        <div className="toolbar">
          <h2>Venues</h2>
          {canCreate ? (
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => {
                setSaved(null);
                setCreating(true);
              }}
            >
              New venue
            </button>
          ) : (
            <span className="muted small">Only administrators can add venues.</span>
          )}
        </div>
      )}
      {error && <ErrorBox message={error} onRetry={reload} />}
      {loading && !venues && <Loading />}
      {venues &&
        (venues.length === 0 ? (
          <Empty title="No venues yet" />
        ) : (
          <div className="venue-grid">
            {venues.map((v) => (
              <article key={v.id} className="panel venue-card">
                <h3>{v.name}</h3>
                <p className="muted small">{v.address}</p>
                <ul className="hall-list">
                  {(v.halls ?? []).map((h) => (
                    <li key={h.id}>
                      <strong>{h.name}</strong>
                      <span className="muted small block">
                        {h.kind === 'seated'
                          ? `${plural(h.capacity, 'seat')} · zones: ${(h.zones ?? []).map((z) => z.name).join(', ')}`
                          : `General admission · ${plural(h.capacity, 'place')}`}
                      </span>
                    </li>
                  ))}
                </ul>
              </article>
            ))}
          </div>
        ))}
    </div>
  );
}

interface ZoneRow {
  name: string;
  rows: string;
  seatsPerRow: string;
}

interface HallRow {
  name: string;
  kind: HallKind;
  capacity: string;
  zones: ZoneRow[];
}

const newZone = (): ZoneRow => ({ name: '', rows: '2', seatsPerRow: '10' });
const newHall = (): HallRow => ({ name: '', kind: 'seated', capacity: '100', zones: [newZone()] });

function hallCapacity(h: HallRow): number {
  if (h.kind === 'general_admission') return Number(h.capacity) || 0;
  return h.zones.reduce((s, z) => s + (Number(z.rows) || 0) * (Number(z.seatsPerRow) || 0), 0);
}

function VenueForm({ onCancel, onSaved }: { onCancel: () => void; onSaved: (name: string) => void }) {
  const [name, setName] = useState('');
  const [address, setAddress] = useState('');
  const [halls, setHalls] = useState<HallRow[]>([newHall()]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateHall = (i: number, patch: Partial<HallRow>) => setHalls((hs) => hs.map((h, j) => (j === i ? { ...h, ...patch } : h)));
  const updateZone = (hi: number, zi: number, patch: Partial<ZoneRow>) =>
    setHalls((hs) => hs.map((h, j) => (j === hi ? { ...h, zones: h.zones.map((z, k) => (k === zi ? { ...z, ...patch } : z)) } : h)));

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    const specs: HallSpec[] = [];
    for (const h of halls) {
      if (!h.name.trim()) return setError('Every hall needs a name.');
      if (h.kind === 'general_admission') {
        const capacity = Number(h.capacity);
        if (!Number.isInteger(capacity) || capacity <= 0) return setError(`Hall “${h.name}”: capacity must be a positive whole number.`);
        specs.push({ name: h.name.trim(), kind: h.kind, capacity, zones: [] });
      } else {
        if (h.zones.length === 0) return setError(`Hall “${h.name}” needs at least one price zone.`);
        const zones = [];
        for (const z of h.zones) {
          const rows = Number(z.rows);
          const seatsPerRow = Number(z.seatsPerRow);
          if (!z.name.trim() || !Number.isInteger(rows) || rows <= 0 || !Number.isInteger(seatsPerRow) || seatsPerRow <= 0) {
            return setError(`Hall “${h.name}”: every zone needs a name and positive numbers of rows and seats.`);
          }
          zones.push({ name: z.name.trim(), rows, seatsPerRow });
        }
        specs.push({ name: h.name.trim(), kind: h.kind, capacity: 0, zones });
      }
    }
    setBusy(true);
    try {
      await api.createVenue({ name: name.trim(), address: address.trim(), halls: specs });
      onSaved(name.trim());
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  return (
    <form className="panel form-grid" onSubmit={submit}>
      <h2 className="span-all">New venue</h2>
      <label className="field">
        <span>Name</span>
        <input value={name} onChange={(e) => setName(e.target.value)} required maxLength={200} />
      </label>
      <label className="field">
        <span>Address</span>
        <input value={address} onChange={(e) => setAddress(e.target.value)} required maxLength={300} />
      </label>

      {halls.map((h, hi) => (
        <fieldset key={hi} className="span-all subform">
          <legend>
            Hall {hi + 1} · capacity {hallCapacity(h)}
          </legend>
          <div className="form-grid">
            <label className="field">
              <span>Hall name</span>
              <input value={h.name} onChange={(e) => updateHall(hi, { name: e.target.value })} required maxLength={200} />
            </label>
            <label className="field">
              <span>Kind</span>
              <select value={h.kind} onChange={(e) => updateHall(hi, { kind: e.target.value as HallKind })}>
                <option value="seated">Seated (seat map with zones)</option>
                <option value="general_admission">General admission (no seats)</option>
              </select>
            </label>
            {h.kind === 'general_admission' ? (
              <label className="field">
                <span>Capacity</span>
                <input type="number" min={1} step={1} value={h.capacity} onChange={(e) => updateHall(hi, { capacity: e.target.value })} required />
              </label>
            ) : (
              <div className="span-all stack">
                <p className="muted small">Zones take rows from the front: the first zone gets rows 1…n, the next zone continues. Up to 100 rows and 100 seats per row.</p>
                {h.zones.map((z, zi) => (
                  <div key={zi} className="addon-row">
                    <label className="field grow">
                      <span>Zone name</span>
                      <input value={z.name} onChange={(e) => updateZone(hi, zi, { name: e.target.value })} placeholder="e.g. Parterre" required />
                    </label>
                    <label className="field">
                      <span>Rows</span>
                      <input type="number" min={1} max={100} step={1} value={z.rows} onChange={(e) => updateZone(hi, zi, { rows: e.target.value })} required />
                    </label>
                    <label className="field">
                      <span>Seats per row</span>
                      <input
                        type="number"
                        min={1}
                        max={100}
                        step={1}
                        value={z.seatsPerRow}
                        onChange={(e) => updateZone(hi, zi, { seatsPerRow: e.target.value })}
                        required
                      />
                    </label>
                    <button
                      type="button"
                      className="icon-btn addon-remove"
                      aria-label="Remove zone"
                      disabled={h.zones.length === 1}
                      onClick={() => updateHall(hi, { zones: h.zones.filter((_, k) => k !== zi) })}
                    >
                      ×
                    </button>
                  </div>
                ))}
                <div>
                  <button type="button" className="btn btn-small btn-secondary" onClick={() => updateHall(hi, { zones: [...h.zones, newZone()] })}>
                    + Zone
                  </button>
                </div>
              </div>
            )}
          </div>
          {halls.length > 1 && (
            <button type="button" className="btn btn-small btn-danger-ghost" onClick={() => setHalls(halls.filter((_, j) => j !== hi))}>
              Remove hall
            </button>
          )}
        </fieldset>
      ))}
      <div className="span-all">
        <button type="button" className="btn btn-small btn-secondary" onClick={() => setHalls([...halls, newHall()])}>
          + Hall
        </button>
      </div>
      {error && (
        <div className="span-all">
          <ErrorBox message={error} />
        </div>
      )}
      <div className="btn-row span-all">
        <button type="submit" className="btn btn-primary" disabled={busy}>
          {busy ? 'Creating…' : 'Create venue'}
        </button>
        <button type="button" className="btn btn-ghost" onClick={onCancel} disabled={busy}>
          Cancel
        </button>
      </div>
    </form>
  );
}
