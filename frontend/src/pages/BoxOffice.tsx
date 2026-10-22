import { useState, type FormEvent } from 'react';
import { api, ApiError, errorMessage, type Order } from '../api';
import { ErrorBox, Notice, PageHeader, StatusBadge, TypeBadge } from '../components/ui';
import { formatDateTime, formatMoney, plural, ticketPlace } from '../format';
import { useTitle } from '../hooks';

/** Box office (UC23, UC17): find a booking by its code and accept the cash payment. */
export function BoxOffice() {
  useTitle('Box office');
  const [code, setCode] = useState('');
  const [order, setOrder] = useState<Order | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [busy, setBusy] = useState<'' | 'find' | 'pay'>('');

  const find = async (e: FormEvent) => {
    e.preventDefault();
    const c = code.trim().toUpperCase();
    if (!c) return;
    setBusy('find');
    setError(null);
    setSuccess(null);
    setOrder(null);
    try {
      setOrder(await api.findBooking(c));
    } catch (err) {
      setError(err instanceof ApiError && err.status === 404 ? `No booking with code ${c}.` : errorMessage(err));
    } finally {
      setBusy('');
    }
  };

  const accept = async () => {
    if (!order?.bookingCode) return;
    if (!window.confirm(`Confirm that ${formatMoney(order.total)} was received in cash?`)) return;
    setBusy('pay');
    setError(null);
    try {
      const paid = await api.payCash(order.bookingCode);
      setOrder(paid);
      setSuccess(`Cash payment of ${formatMoney(paid.total)} recorded. Order #${paid.id} is paid.`);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy('');
    }
  };

  const courier = order?.delivery.option === 'courier';
  const deadlinePassed = order ? new Date(order.bookingDeadline).getTime() <= Date.now() : false;
  const canAccept = order?.status === 'booked' && !courier && !deadlinePassed;

  return (
    <div className="container narrow section">
      <PageHeader title="Box office" subtitle="Find a booking by its code and accept cash." />
      <form className="panel lookup" onSubmit={find}>
        <label className="field">
          <span>Booking code</span>
          <input
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            placeholder="e.g. K7M2P9QX"
            maxLength={16}
            autoComplete="off"
            spellCheck={false}
            className="code-input"
            required
            autoFocus
          />
        </label>
        <button type="submit" className="btn btn-primary" disabled={busy !== '' || !code.trim()}>
          {busy === 'find' ? 'Searching…' : 'Find booking'}
        </button>
      </form>

      {error && <ErrorBox message={error} />}
      {success && <Notice kind="success">{success}</Notice>}

      {order && (
        <section className="panel stack">
          <div className="page-header-row">
            <div>
              <h2>
                Order #{order.id} · <code>{order.bookingCode}</code>
              </h2>
              <p className="muted small">Pay before {formatDateTime(order.bookingDeadline)}</p>
            </div>
            <StatusBadge status={order.status} />
          </div>
          <ul className="summary-list">
            {(order.tickets ?? []).map((t) => (
              <li key={t.id}>
                <div>
                  <span className="block">
                    <TypeBadge type={t.eventType} /> <strong>{t.eventTitle}</strong>
                  </span>
                  <span className="muted small block">
                    {formatDateTime(t.startsAt)} · {t.venueName}, {t.hallName} · {ticketPlace(t)}
                  </span>
                  {(t.addOns ?? []).map((a) => (
                    <span key={a.addOnId} className="muted small block">
                      + {a.name} ({formatMoney(a.price)})
                    </span>
                  ))}
                </div>
                <span className="summary-price">{formatMoney(t.price + (t.addOns ?? []).reduce((s, a) => s + a.price, 0))}</span>
              </li>
            ))}
          </ul>
          <div className="summary-total">
            <span>To pay · {plural(order.tickets?.length ?? 0, 'ticket')}</span>
            <strong>{formatMoney(order.total)}</strong>
          </div>
          {order.status !== 'booked' && <Notice kind="info">This order is not an active booking, so it cannot be paid here.</Notice>}
          {order.status === 'booked' && courier && (
            <Notice kind="warning">This booking has courier delivery, which requires a card payment. The customer must pay online.</Notice>
          )}
          {order.status === 'booked' && deadlinePassed && <Notice kind="warning">The booking deadline has passed.</Notice>}
          <button type="button" className="btn btn-primary btn-large" onClick={accept} disabled={!canAccept || busy !== ''}>
            {busy === 'pay' ? 'Recording…' : `Accept cash ${formatMoney(order.total)}`}
          </button>
        </section>
      )}
    </div>
  );
}
