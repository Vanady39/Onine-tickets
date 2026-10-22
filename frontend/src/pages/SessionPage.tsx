import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api, type SeatState, type Session } from '../api';
import { signInPath, useAuth } from '../auth';
import { itemsTotal, MAX_PER_EVENT, useCart, type CartItem } from '../cart';
import { SeatLegend, SeatMap, zoneColorMap } from '../components/SeatMap';
import { ErrorBox, Loading, Notice, TypeBadge } from '../components/ui';
import { ADD_ON_KIND_LABEL, BOOKING_WINDOW_MS, formatDateTime, formatMoney, plural, ROLE_LABEL } from '../format';
import { useAsync, useTitle } from '../hooks';

interface Loaded {
  session: Session;
  seats: SeatState[];
}

/** Session page (UC04, UC10, UC11, UC12): choose seats or general-admission tickets with add-ons. */
export function SessionPage() {
  const id = Number(useParams().id);
  const { data, error, loading, reload } = useAsync<Loaded>(async () => {
    const session = await api.session(id);
    const seats = session.hall.kind === 'seated' ? await api.seats(id) : [];
    return { session, seats };
  }, [id]);
  useTitle(data?.session.eventTitle);

  // Keep the seat map fresh while the customer is choosing (other buyers hold seats too).
  useEffect(() => {
    if (data?.session.hall.kind !== 'seated') return;
    const t = window.setInterval(reload, 30_000);
    return () => window.clearInterval(t);
  }, [data?.session.hall.kind, reload]);

  if (loading && !data) return <Loading label="Loading the session…" />;
  if (error && !data) {
    return (
      <div className="container section">
        <ErrorBox message={error === 'not found' ? 'This session does not exist.' : error} onRetry={reload} />
        <Link to="/">← Back to the programme</Link>
      </div>
    );
  }
  if (!data) return null;
  return <SessionView key={data.session.id} session={data.session} seats={data.seats} reload={reload} refreshing={loading} error={error} />;
}

function SessionView({
  session,
  seats,
  reload,
  refreshing,
  error,
}: Loaded & { reload: () => void; refreshing: boolean; error: string | null }) {
  const { user } = useAuth();
  const cart = useCart();
  const navigate = useNavigate();
  const seated = session.hall.kind === 'seated';
  const startsAt = new Date(session.startsAt).getTime();
  const started = startsAt <= Date.now();
  const bookingClosed = startsAt - Date.now() <= BOOKING_WINDOW_MS;

  const [message, setMessage] = useState<{ kind: 'info' | 'success' | 'warning'; text: string; link?: boolean } | null>(null);

  // --- seated selection
  const [selected, setSelected] = useState<number[]>([]);
  const colors = useMemo(() => zoneColorMap(session.prices ?? [], seats), [session.prices, seats]);
  const seatById = useMemo(() => new Map(seats.map((s) => [s.id, s])), [seats]);

  // Drop selected seats that someone else took meanwhile.
  useEffect(() => {
    setSelected((sel) => {
      const still = sel.filter((sid) => seatById.get(sid)?.status === 'free');
      if (still.length !== sel.length) {
        setMessage({ kind: 'warning', text: 'Some of your selected seats were just taken by another customer.' });
      }
      return still.length === sel.length ? sel : still;
    });
  }, [seatById]);

  const toggleSeat = (s: SeatState) => {
    setMessage(null);
    if (selected.includes(s.id)) {
      setSelected(selected.filter((x) => x !== s.id));
      return;
    }
    if (s.status !== 'free') return;
    if (selected.length >= MAX_PER_EVENT) {
      setMessage({ kind: 'info', text: `You can choose at most ${MAX_PER_EVENT} tickets for one event in one order.` });
      return;
    }
    setSelected([...selected, s.id]);
  };

  // --- general admission
  const addOns = session.addOns ?? [];
  const maxQty = Math.max(0, Math.min(MAX_PER_EVENT, session.remaining));
  const [qty, setQty] = useState(Math.min(1, maxQty));
  const [ticketAddOns, setTicketAddOns] = useState<number[][]>([[], [], []]);
  const usedAddOn = (addOnId: number) => ticketAddOns.slice(0, qty).filter((list) => list.includes(addOnId)).length;
  const toggleAddOn = (ticket: number, addOnId: number) => {
    setTicketAddOns((all) =>
      all.map((list, i) => (i !== ticket ? list : list.includes(addOnId) ? list.filter((x) => x !== addOnId) : [...list, addOnId])),
    );
  };

  // --- cart items for the current selection
  const items: CartItem[] = useMemo(() => {
    const base = {
      sessionId: session.id,
      eventId: session.eventId,
      eventTitle: session.eventTitle,
      eventType: session.eventType,
      startsAt: session.startsAt,
      venueName: session.hall.venueName,
      hallName: session.hall.name,
    };
    if (seated) {
      return selected
        .map((sid) => seatById.get(sid))
        .filter((s): s is SeatState => !!s)
        .sort((a, b) => a.row - b.row || a.number - b.number)
        .map((s) => ({
          ...base,
          key: `s${session.id}-seat${s.id}`,
          seatId: s.id,
          place: `${s.zoneName} · Row ${s.row}, Seat ${s.number}`,
          price: s.price,
          addOns: [],
        }));
    }
    const stamp = Date.now().toString(36);
    return Array.from({ length: qty }, (_, i) => ({
      ...base,
      key: `s${session.id}-ga-${stamp}-${i}`,
      place: 'General admission',
      price: session.basePrice,
      addOns: addOns.filter((a) => ticketAddOns[i].includes(a.id)).map((a) => ({ id: a.id, name: a.name, price: a.price })),
    }));
  }, [session, seated, selected, seatById, qty, ticketAddOns, addOns]);

  const total = itemsTotal(items);
  const isCustomer = !user || user.role === 'customer';
  const canSeason = user?.role === 'customer' && user.loyal;

  const checkout = () => {
    cart.startCheckout({ source: 'single', items, backTo: `/sessions/${session.id}` });
    navigate(user ? '/checkout' : signInPath('/checkout'));
  };

  const addToSeason = () => {
    const err = cart.addToSeason(items);
    if (err) {
      setMessage({ kind: 'warning', text: err });
      return;
    }
    setSelected([]);
    setTicketAddOns([[], [], []]);
    setMessage({ kind: 'success', text: `${plural(items.length, 'ticket')} added to your season ticket.`, link: true });
  };

  return (
    <div className="container section">
      <Link className="back-link" to={`/events/${session.eventId}`}>
        ← {session.eventTitle}
      </Link>
      <div className={`session-head type-edge-${session.eventType}`}>
        <div>
          <TypeBadge type={session.eventType} />
          <h1>{session.eventTitle}</h1>
          <p className="session-head-meta">
            <strong>{formatDateTime(session.startsAt)}</strong>
            <span>
              {session.hall.venueName} · {session.hall.name}
            </span>
            <span className="muted">{session.hall.address}</span>
          </p>
        </div>
        <button type="button" className="btn btn-ghost btn-small" onClick={reload} disabled={refreshing}>
          {refreshing ? 'Refreshing…' : seated ? 'Refresh seats' : 'Refresh'}
        </button>
      </div>

      {error && <ErrorBox message={error} onRetry={reload} />}
      {started && <Notice kind="warning">This session has already started — tickets are no longer sold.</Notice>}
      {!started && bookingClosed && (
        <Notice kind="info">This session starts within 24 hours: booking is closed, but you can still buy tickets by card.</Notice>
      )}
      {user && !isCustomer && (
        <Notice kind="info">Tickets are bought with a customer account. You are signed in as {ROLE_LABEL[user.role]}.</Notice>
      )}

      <div className="session-layout">
        <section className="panel" aria-label={seated ? 'Choose seats' : 'Choose tickets'}>
          {seated ? (
            <>
              <SeatLegend prices={session.prices ?? []} colors={colors} />
              {seats.length === 0 ? (
                <p className="muted">The seat map is not available.</p>
              ) : (
                <SeatMap
                  seats={seats}
                  selected={new Set(selected)}
                  colors={colors}
                  stageLabel={session.eventType === 'cinema' ? 'Screen' : 'Stage'}
                  onToggle={started ? () => undefined : toggleSeat}
                />
              )}
            </>
          ) : (
            <GeneralAdmission
              session={session}
              qty={qty}
              maxQty={maxQty}
              setQty={setQty}
              ticketAddOns={ticketAddOns}
              usedAddOn={usedAddOn}
              toggleAddOn={toggleAddOn}
              disabled={started}
            />
          )}
        </section>

        <aside className="panel summary" aria-label="Your selection">
          <h2>Your tickets</h2>
          {items.length === 0 ? (
            <p className="muted">{seated ? 'Tap free seats on the map to select up to 3.' : 'Choose the number of tickets.'}</p>
          ) : (
            <ul className="summary-list">
              {items.map((i) => (
                <li key={i.key}>
                  <div>
                    <span>{i.place}</span>
                    {i.addOns.map((a) => (
                      <span key={a.id} className="muted small block">
                        + {a.name}
                      </span>
                    ))}
                  </div>
                  <span className="summary-price">
                    {formatMoney(i.price + i.addOns.reduce((s, a) => s + a.price, 0))}
                    {seated && (
                      <button
                        type="button"
                        className="icon-btn"
                        aria-label={`Remove ${i.place}`}
                        onClick={() => setSelected(selected.filter((x) => x !== i.seatId))}
                      >
                        ×
                      </button>
                    )}
                  </span>
                </li>
              ))}
            </ul>
          )}
          <div className="summary-total">
            <span>Total</span>
            <strong>{formatMoney(total)}</strong>
          </div>
          <p className="muted small">
            {items.length}/{MAX_PER_EVENT} tickets for this event per order.
          </p>
          {message && (
            <Notice kind={message.kind}>
              {message.text}{' '}
              {message.link && (
                <Link to="/season">
                  <strong>Open season ticket</strong>
                </Link>
              )}
            </Notice>
          )}
          <div className="stack">
            <button type="button" className="btn btn-primary btn-block" disabled={!items.length || started || !isCustomer} onClick={checkout}>
              {user ? 'Proceed to checkout' : 'Sign in and check out'}
            </button>
            {canSeason && (
              <button type="button" className="btn btn-secondary btn-block" disabled={!items.length || started} onClick={addToSeason}>
                Add to season ticket
              </button>
            )}
          </div>
          {canSeason && cart.season.length > 0 && (
            <p className="small">
              Season ticket: <Link to="/season">{plural(cart.season.length, 'ticket')}</Link>
            </p>
          )}
        </aside>
      </div>
    </div>
  );
}

function GeneralAdmission({
  session,
  qty,
  maxQty,
  setQty,
  ticketAddOns,
  usedAddOn,
  toggleAddOn,
  disabled,
}: {
  session: Session;
  qty: number;
  maxQty: number;
  setQty: (n: number) => void;
  ticketAddOns: number[][];
  usedAddOn: (id: number) => number;
  toggleAddOn: (ticket: number, addOnId: number) => void;
  disabled: boolean;
}) {
  const addOns = session.addOns ?? [];
  if (maxQty === 0) return <Notice kind="warning">This session is sold out.</Notice>;
  return (
    <div className="ga">
      <div className="ga-head">
        <div>
          <h2>General admission</h2>
          <p className="muted">
            {formatMoney(session.basePrice)} per ticket · {plural(session.remaining, 'place')} left
          </p>
        </div>
        <div className="stepper" role="group" aria-label="Number of tickets">
          <button type="button" onClick={() => setQty(Math.max(1, qty - 1))} disabled={disabled || qty <= 1} aria-label="Fewer tickets">
            −
          </button>
          <output aria-live="polite">{qty}</output>
          <button type="button" onClick={() => setQty(Math.min(maxQty, qty + 1))} disabled={disabled || qty >= maxQty} aria-label="More tickets">
            +
          </button>
        </div>
      </div>
      {addOns.length > 0 && (
        <>
          <h3 className="ga-sub">Add-on services</h3>
          <p className="muted small">Optional extras for each ticket, limited places.</p>
          <div className="ga-tickets">
            {Array.from({ length: qty }, (_, i) => (
              <fieldset key={i} className="ga-ticket" disabled={disabled}>
                <legend>Ticket {i + 1}</legend>
                {addOns.map((a) => {
                  const checked = ticketAddOns[i].includes(a.id);
                  const left = a.remaining - usedAddOn(a.id);
                  const full = !checked && left <= 0;
                  return (
                    <label key={a.id} className={`check${full ? ' check-disabled' : ''}`}>
                      <input type="checkbox" checked={checked} disabled={full} onChange={() => toggleAddOn(i, a.id)} />
                      <span className="check-body">
                        <span>
                          <strong>{a.name}</strong>
                          <span className="muted small block">
                            {ADD_ON_KIND_LABEL[a.kind]} · {a.remaining > 0 ? `${a.remaining} of ${a.capacity} left` : 'fully booked'}
                          </span>
                        </span>
                        <span className="nowrap">+{formatMoney(a.price)}</span>
                      </span>
                    </label>
                  );
                })}
              </fieldset>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
