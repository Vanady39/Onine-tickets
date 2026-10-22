import { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { api, type Order } from '../api';
import { useAuth } from '../auth';
import { Empty, ErrorBox, Loading, PageHeader, StatusBadge, TypeBadge } from '../components/ui';
import { formatDateTime, formatMoney, formatTimestamp, plural } from '../format';
import { useAsync, useTitle } from '../hooks';

const LOYAL_AFTER = 3; // BR-3: loyal = more than 3 paid orders

/** My tickets (UC07): profile with loyalty status and the purchase history. */
export function MyOrders() {
  useTitle('My tickets');
  const { user, refreshProfile } = useAuth();
  const { data: orders, error, loading, reload } = useAsync(() => api.orders(), []);
  // Loyalty depends on paid orders, which change with payments and refunds.
  useEffect(() => {
    void refreshProfile();
  }, [refreshProfile]);

  return (
    <div className="container section">
      <PageHeader title="My tickets" />
      <div className="orders-layout">
        {user && (
          <aside className="panel profile-card">
            <div className="avatar" aria-hidden="true">
              {user.name.trim().charAt(0).toUpperCase()}
            </div>
            <h2>{user.name}</h2>
            <p className="muted">{user.email}</p>
            {user.loyal ? (
              <span className="badge badge-loyal">★ Loyal customer</span>
            ) : (
              <span className="badge badge-regular">Regular customer</span>
            )}
            <dl className="details">
              <div>
                <dt>Paid orders</dt>
                <dd>{user.paidOrders}</dd>
              </div>
            </dl>
            {user.loyal ? (
              <p className="small">
                You can combine tickets for several events in one order — a season ticket.{' '}
                <Link to="/season">Open season ticket</Link>
              </p>
            ) : (
              <>
                <div className="progress" aria-label={`${user.paidOrders} of ${LOYAL_AFTER + 1} paid orders`}>
                  <span style={{ width: `${Math.min(100, (user.paidOrders / (LOYAL_AFTER + 1)) * 100)}%` }} />
                </div>
                <p className="small muted">
                  {plural(Math.max(0, LOYAL_AFTER + 1 - user.paidOrders), 'more paid order')} to become a loyal customer and unlock
                  season tickets.
                </p>
              </>
            )}
          </aside>
        )}

        <section aria-label="Order history">
          {error && <ErrorBox message={error} onRetry={reload} />}
          {loading && !orders && <Loading label="Loading your orders…" />}
          {orders &&
            (orders.length === 0 ? (
              <Empty title="No orders yet">
                <p>Your tickets and bookings will appear here.</p>
                <Link className="btn btn-primary" to="/">
                  Find an event
                </Link>
              </Empty>
            ) : (
              <ul className="order-list">
                {orders.map((o) => (
                  <li key={o.id}>
                    <OrderRow order={o} />
                  </li>
                ))}
              </ul>
            ))}
        </section>
      </div>
    </div>
  );
}

function OrderRow({ order }: { order: Order }) {
  const tickets = order.tickets ?? [];
  const events = new Map<number, { title: string; type: (typeof tickets)[number]['eventType']; startsAt: string }>();
  for (const t of tickets) {
    const prev = events.get(t.eventId);
    if (!prev || t.startsAt < prev.startsAt) events.set(t.eventId, { title: t.eventTitle, type: t.eventType, startsAt: t.startsAt });
  }
  return (
    <Link to={`/orders/${order.id}`} className="order-card">
      <div className="order-card-head">
        <strong>Order #{order.id}</strong>
        <StatusBadge status={order.status} />
        {events.size > 1 && <span className="badge badge-loyal">Season ticket</span>}
      </div>
      <ul className="order-card-events">
        {[...events.values()].map((e) => (
          <li key={e.title + e.startsAt}>
            <TypeBadge type={e.type} /> <span>{e.title}</span> <span className="muted small">{formatDateTime(e.startsAt)}</span>
          </li>
        ))}
      </ul>
      <div className="order-card-foot">
        <span className="muted small">
          {plural(tickets.length, 'ticket')} · {formatTimestamp(order.createdAt)}
          {order.status === 'booked' && order.bookingCode ? (
            <>
              {' '}
              · code <code>{order.bookingCode}</code>
            </>
          ) : null}
        </span>
        <strong>{formatMoney(order.total)}</strong>
      </div>
    </Link>
  );
}
