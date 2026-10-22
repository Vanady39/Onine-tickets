import { useMemo, useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { api, ApiError, errorMessage, type DeliveryOption, type OrderRequest } from '../api';
import { useAuth } from '../auth';
import { countByEvent, itemTotal, itemsTotal, MAX_PER_EVENT, toOrderItems, useCart, type CartItem } from '../cart';
import { Empty, ErrorBox, Notice, PageHeader, TypeBadge } from '../components/ui';
import { BOOKING_WINDOW_MS, formatDateTime, formatMoney, plural } from '../format';
import { useTitle } from '../hooks';

const COURIER_FEE = 30_000; // 300 ₽, domain.CourierFee

type PayChoice = 'card' | 'book' | 'cash';

/** Checkout (UC08, UC09, UC13–UC19): summary, payment method, delivery and confirmation. */
export function Checkout() {
  useTitle('Checkout');
  const { draft, clearDraft, clearSeason } = useCart();
  const { user, refreshProfile } = useAuth();
  const navigate = useNavigate();

  const [pay, setPay] = useState<PayChoice>('card');
  const [delivery, setDelivery] = useState<DeliveryOption>('e_ticket');
  const [address, setAddress] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<{ text: string; conflict: boolean } | null>(null);

  const items = useMemo(() => draft?.items ?? [], [draft]);
  const groups = useMemo(() => groupBySession(items), [items]);
  const earliest = useMemo(() => Math.min(...items.map((i) => new Date(i.startsAt).getTime())), [items]);

  if (!draft || items.length === 0) {
    return (
      <div className="container narrow section">
        <Empty title="Nothing to check out">
          <p>Choose seats or tickets for a session first.</p>
          <Link className="btn btn-primary" to="/">
            Browse the programme
          </Link>
        </Empty>
      </div>
    );
  }

  const deadline = earliest - BOOKING_WINDOW_MS;
  const bookingAllowed = deadline > Date.now();
  const events = countByEvent(items);
  const isSeason = events.size >= 2;
  const overLimit = [...events.values()].some((n) => n > MAX_PER_EVENT);
  const courier = delivery === 'courier';
  const fee = courier ? COURIER_FEE : 0;
  const total = itemsTotal(items) + fee;

  const choosePay = (p: PayChoice) => {
    setPay(p);
    if (p === 'cash') setDelivery('e_ticket');
  };

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (submitting) return;
    if (courier && !address.trim()) {
      setError({ text: 'Enter the delivery address for the courier.', conflict: false });
      return;
    }
    setSubmitting(true);
    setError(null);
    const req: OrderRequest = {
      items: toOrderItems(items),
      paymentMethod: pay === 'cash' ? 'cash' : 'card',
      book: pay !== 'card',
      deliveryOption: delivery,
      ...(courier ? { deliveryAddress: address.trim() } : {}),
    };
    let orderId: number;
    try {
      const order = await api.placeOrder(req);
      orderId = order.id;
    } catch (err) {
      setError({ text: errorMessage(err), conflict: err instanceof ApiError && err.status === 409 });
      setSubmitting(false);
      return;
    }
    if (draft.source === 'season') clearSeason();
    clearDraft();
    void refreshProfile();
    if (pay !== 'card') {
      navigate(`/orders/${orderId}`, { replace: true, state: { created: true } });
      return;
    }
    try {
      const { redirectUrl } = await api.pay(orderId);
      window.location.assign(redirectUrl);
    } catch (err) {
      navigate(`/orders/${orderId}`, { replace: true, state: { payError: errorMessage(err) } });
    }
  };

  return (
    <div className="container section">
      <PageHeader
        title={isSeason ? 'Season ticket checkout' : 'Checkout'}
        back={{ to: draft.backTo, label: 'Change selection' }}
      />
      <form className="checkout-layout" onSubmit={submit}>
        <div className="stack-lg">
          {isSeason && !user?.loyal && (
            <Notice kind="warning">
              Tickets for several events in one order (a season ticket) are available to loyal customers only — more than 3 paid
              orders.
            </Notice>
          )}
          {overLimit && <Notice kind="warning">At most {MAX_PER_EVENT} tickets for one event are allowed in one order.</Notice>}

          <fieldset className="panel">
            <legend className="panel-title">Payment</legend>
            <div className="options">
              <Option
                name="pay"
                checked={pay === 'card'}
                onChange={() => choosePay('card')}
                title="Pay by card now"
                text="You are sent to the secure payment page. Your seats are held for 10 minutes while you pay."
              />
              <Option
                name="pay"
                checked={pay === 'book'}
                disabled={!bookingAllowed}
                onChange={() => choosePay('book')}
                title="Book now, pay later"
                text={
                  bookingAllowed
                    ? `Get a booking code and pay online or at the box office until ${formatDateTime(new Date(deadline).toISOString())}.`
                    : 'Unavailable: booking closes 24 hours before the session.'
                }
              />
              <Option
                name="pay"
                checked={pay === 'cash'}
                disabled={!bookingAllowed}
                onChange={() => choosePay('cash')}
                title="Cash at the box office"
                text={
                  bookingAllowed
                    ? `Your tickets are booked; show the booking code at the box office and pay before ${formatDateTime(new Date(deadline).toISOString())}.`
                    : 'Unavailable: booking closes 24 hours before the session.'
                }
              />
            </div>
          </fieldset>

          <fieldset className="panel">
            <legend className="panel-title">Delivery</legend>
            <div className="options">
              <Option
                name="delivery"
                checked={delivery === 'e_ticket'}
                onChange={() => setDelivery('e_ticket')}
                title="E-ticket"
                text="Free. Tickets with QR codes appear in your account right after payment."
                extra="Free"
              />
              <Option
                name="delivery"
                checked={courier}
                disabled={pay === 'cash'}
                onChange={() => setDelivery('courier')}
                title="Courier delivery"
                text={pay === 'cash' ? 'Unavailable with cash payment: courier delivery requires a card payment.' : 'Printed tickets delivered to your address in the city after payment.'}
                extra={`+${formatMoney(COURIER_FEE)}`}
              />
            </div>
            {courier && (
              <label className="field">
                <span>Delivery address</span>
                <input
                  type="text"
                  value={address}
                  onChange={(e) => setAddress(e.target.value)}
                  placeholder="Street, house, flat, city"
                  autoComplete="street-address"
                  required
                  maxLength={300}
                />
              </label>
            )}
          </fieldset>
        </div>

        <aside className="panel summary">
          <h2>Order summary</h2>
          {groups.map((g) => (
            <div key={g.sessionId} className="summary-group">
              <div className="summary-group-head">
                <TypeBadge type={g.items[0].eventType} />
                <strong>{g.items[0].eventTitle}</strong>
                <span className="muted small">
                  {formatDateTime(g.items[0].startsAt)} · {g.items[0].venueName}, {g.items[0].hallName}
                </span>
              </div>
              <ul className="summary-list">
                {g.items.map((i) => (
                  <li key={i.key}>
                    <div>
                      <span>{i.place}</span>
                      {i.addOns.map((a) => (
                        <span key={a.id} className="muted small block">
                          + {a.name} ({formatMoney(a.price)})
                        </span>
                      ))}
                    </div>
                    <span className="summary-price">{formatMoney(itemTotal(i))}</span>
                  </li>
                ))}
              </ul>
            </div>
          ))}
          {courier && (
            <div className="summary-line">
              <span>Courier delivery</span>
              <span>{formatMoney(fee)}</span>
            </div>
          )}
          <div className="summary-total">
            <span>Total · {plural(items.length, 'ticket')}</span>
            <strong>{formatMoney(total)}</strong>
          </div>
          <p className="muted small">
            {pay === 'card'
              ? 'Seats are held for 10 minutes after you confirm. If the payment is not completed in time, they return to sale.'
              : `An unpaid booking expires 24 hours before the session (${formatDateTime(new Date(deadline).toISOString())}) and the seats return to sale.`}
          </p>
          {error && (
            <div className="stack">
              <ErrorBox message={error.text} />
              {error.conflict && (
                <Link className="btn btn-secondary btn-block" to={draft.backTo}>
                  Choose other seats
                </Link>
              )}
            </div>
          )}
          <button type="submit" className="btn btn-primary btn-block btn-large" disabled={submitting || overLimit}>
            {submitting ? 'Processing…' : pay === 'card' ? `Confirm and pay ${formatMoney(total)}` : 'Confirm booking'}
          </button>
        </aside>
      </form>
    </div>
  );
}

function Option({
  name,
  checked,
  disabled,
  onChange,
  title,
  text,
  extra,
}: {
  name: string;
  checked: boolean;
  disabled?: boolean;
  onChange: () => void;
  title: string;
  text: string;
  extra?: string;
}) {
  return (
    <label className={`option${checked ? ' selected' : ''}${disabled ? ' disabled' : ''}`}>
      <input type="radio" name={name} checked={checked} disabled={disabled} onChange={onChange} />
      <span className="option-body">
        <span className="option-title">
          {title}
          {extra && <span className="option-extra">{extra}</span>}
        </span>
        <span className="option-text">{text}</span>
      </span>
    </label>
  );
}

function groupBySession(items: CartItem[]): { sessionId: number; items: CartItem[] }[] {
  const groups: { sessionId: number; items: CartItem[] }[] = [];
  for (const i of [...items].sort((a, b) => a.startsAt.localeCompare(b.startsAt))) {
    const g = groups.find((x) => x.sessionId === i.sessionId);
    if (g) g.items.push(i);
    else groups.push({ sessionId: i.sessionId, items: [i] });
  }
  return groups;
}
