import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth';
import { countByEvent, itemTotal, itemsTotal, MAX_PER_EVENT, useCart, type CartItem } from '../cart';
import { Empty, Notice, PageHeader, TypeBadge } from '../components/ui';
import { formatDateTime, formatMoney, plural } from '../format';
import { useTitle } from '../hooks';

/** Season ticket (UC09): a cart of tickets for several events, for loyal customers only. */
export function SeasonTicket() {
  useTitle('Season ticket');
  const { user } = useAuth();
  const { season, removeFromSeason, clearSeason, startCheckout } = useCart();
  const navigate = useNavigate();

  if (!user?.loyal) {
    return (
      <div className="container narrow section">
        <PageHeader title="Season ticket" />
        <Notice kind="info">
          Season tickets — one order with tickets for several events — are available to loyal customers, who have more than 3 paid
          orders. You have {plural(user?.paidOrders ?? 0, 'paid order')}.
        </Notice>
      </div>
    );
  }

  const byEvent = new Map<number, CartItem[]>();
  for (const i of [...season].sort((a, b) => a.startsAt.localeCompare(b.startsAt))) {
    byEvent.set(i.eventId, [...(byEvent.get(i.eventId) ?? []), i]);
  }
  const events = countByEvent(season);

  const checkout = () => {
    startCheckout({ source: 'season', items: season, backTo: '/season' });
    navigate('/checkout');
  };

  return (
    <div className="container section">
      <PageHeader
        title="Season ticket"
        subtitle="Collect tickets for several events and pay for them in one order."
        actions={
          season.length > 0 && (
            <button type="button" className="btn btn-ghost btn-small" onClick={clearSeason}>
              Clear all
            </button>
          )
        }
      />
      {season.length === 0 ? (
        <Empty title="Your season ticket is empty">
          <p>Open a session and press “Add to season ticket”. Up to {MAX_PER_EVENT} tickets per event.</p>
          <Link className="btn btn-primary" to="/">
            Browse the programme
          </Link>
        </Empty>
      ) : (
        <div className="checkout-layout">
          <div className="stack">
            {[...byEvent.entries()].map(([eventId, items]) => (
              <section key={eventId} className="panel">
                <div className="summary-group-head">
                  <TypeBadge type={items[0].eventType} />
                  <Link to={`/events/${eventId}`}>
                    <strong>{items[0].eventTitle}</strong>
                  </Link>
                  <span className="muted small">
                    {items.length}/{MAX_PER_EVENT} tickets
                  </span>
                </div>
                <ul className="summary-list">
                  {items.map((i) => (
                    <li key={i.key}>
                      <div>
                        <span>{formatDateTime(i.startsAt)}</span>
                        <span className="muted small block">
                          {i.venueName} · {i.place}
                        </span>
                        {i.addOns.map((a) => (
                          <span key={a.id} className="muted small block">
                            + {a.name}
                          </span>
                        ))}
                      </div>
                      <span className="summary-price">
                        {formatMoney(itemTotal(i))}
                        <button type="button" className="icon-btn" aria-label={`Remove ${i.place}`} onClick={() => removeFromSeason(i.key)}>
                          ×
                        </button>
                      </span>
                    </li>
                  ))}
                </ul>
              </section>
            ))}
          </div>
          <aside className="panel summary">
            <h2>Summary</h2>
            <div className="summary-line">
              <span>Events</span>
              <span>{events.size}</span>
            </div>
            <div className="summary-line">
              <span>Tickets</span>
              <span>{season.length}</span>
            </div>
            <div className="summary-total">
              <span>Total</span>
              <strong>{formatMoney(itemsTotal(season))}</strong>
            </div>
            {events.size < 2 && <p className="muted small">Add tickets for at least one more event to make it a season ticket.</p>}
            <p className="muted small">Seats are not reserved until you confirm the order at checkout.</p>
            <button type="button" className="btn btn-primary btn-block" onClick={checkout}>
              Proceed to checkout
            </button>
          </aside>
        </div>
      )}
    </div>
  );
}
