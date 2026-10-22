import { useEffect, useState } from 'react';
import { Link, useLocation, useParams, useSearchParams } from 'react-router-dom';
import { api, errorMessage, type Order } from '../api';
import { useAuth } from '../auth';
import { TicketCard } from '../components/TicketCard';
import { ErrorBox, Loading, Notice, PageHeader, StatusBadge } from '../components/ui';
import { formatCountdown, formatDateTime, formatMoney, formatTimestamp, plural } from '../format';
import { useAsync, useNow, useTitle } from '../hooks';

/** Order page (UC15–UC21, UC28): payment, booking code, e-tickets, cancellation and refund. */
export function OrderPage() {
  const id = Number(useParams().id);
  useTitle(`Order #${id}`);
  const [params, setParams] = useSearchParams();
  const location = useLocation();
  const navState = (location.state ?? {}) as { created?: boolean; payError?: string };
  const payment = params.get('payment');
  const { refreshProfile } = useAuth();

  const { data: order, error, loading, reload, setData } = useAsync(() => api.order(id), [id]);
  const [busy, setBusy] = useState<'' | 'pay' | 'cancel' | 'refund'>('');
  const [actionError, setActionError] = useState<string | null>(navState.payError ?? null);
  const [actionDone, setActionDone] = useState<string | null>(null);

  // After returning from the provider the confirmation may arrive a moment later: poll briefly.
  const [polls, setPolls] = useState(0);
  useEffect(() => {
    if (payment !== 'succeeded' || !order || (order.status !== 'pending_payment' && order.status !== 'booked') || polls >= 5) return;
    const t = window.setTimeout(() => {
      setPolls((p) => p + 1);
      reload();
    }, 1500);
    return () => window.clearTimeout(t);
  }, [payment, order, polls, reload]);

  // When the seat hold runs out, reload so the page shows the expired order.
  const holdExpiresAt = order?.status === 'pending_payment' ? order.holdExpiresAt : undefined;
  useEffect(() => {
    if (!holdExpiresAt) return;
    const wait = new Date(holdExpiresAt).getTime() - Date.now() + 2000;
    if (wait > 2_147_000_000) return;
    const t = window.setTimeout(reload, Math.max(wait, 0));
    return () => window.clearTimeout(t);
  }, [holdExpiresAt, reload]);

  useEffect(() => {
    if (payment === 'succeeded' && order?.status === 'paid') void refreshProfile();
  }, [payment, order?.status, refreshProfile]);

  if (loading && !order) return <Loading label="Loading your order…" />;
  if (error && !order) {
    return (
      <div className="container narrow section">
        <ErrorBox message={error === 'not found' ? 'Order not found.' : error} onRetry={reload} />
        <Link to="/orders">← My tickets</Link>
      </div>
    );
  }
  if (!order) return null;

  const startPayment = async () => {
    setBusy('pay');
    setActionError(null);
    try {
      const { redirectUrl } = await api.pay(order.id);
      window.location.assign(redirectUrl);
    } catch (err) {
      setActionError(errorMessage(err));
      setBusy('');
      reload();
    }
  };

  const runAction = async (kind: 'cancel' | 'refund') => {
    const question =
      kind === 'cancel'
        ? 'Cancel this order? The seats will be released.'
        : `Request a refund of ${formatMoney(order.total)}? All tickets of this order will become invalid.`;
    if (!window.confirm(question)) return;
    setBusy(kind);
    setActionError(null);
    try {
      const updated = await (kind === 'cancel' ? api.cancel(order.id) : api.refund(order.id));
      setData(updated);
      setActionDone(kind === 'cancel' ? 'The order was cancelled and the seats were released.' : 'The refund was issued to your card.');
      if (kind === 'refund') void refreshProfile();
    } catch (err) {
      setActionError(errorMessage(err));
      reload();
    } finally {
      setBusy('');
    }
  };

  const dismissPayment = () => {
    const next = new URLSearchParams(params);
    next.delete('payment');
    setParams(next, { replace: true });
  };

  return (
    <div className="container section">
      <PageHeader
        title={`Order #${order.id}`}
        back={{ to: '/orders', label: 'My tickets' }}
        subtitle={
          <>
            <StatusBadge status={order.status} /> <span className="muted">Created {formatTimestamp(order.createdAt)}</span>
          </>
        }
        actions={
          <button type="button" className="btn btn-ghost btn-small" onClick={reload} disabled={loading}>
            {loading ? 'Refreshing…' : 'Refresh'}
          </button>
        }
      />

      {payment === 'succeeded' && (
        <PaymentFeedback order={order} onClose={dismissPayment} />
      )}
      {payment === 'declined' && (
        <Notice kind="error">
          The payment was declined and no money was taken.{' '}
          {order.status === 'pending_payment' || order.status === 'booked' ? 'You can try again below.' : ''}
        </Notice>
      )}
      {navState.created && !payment && order.status === 'booked' && (
        <Notice kind="success">Your tickets are booked. Keep the booking code below.</Notice>
      )}
      {actionError && <ErrorBox message={actionError} />}
      {actionDone && <Notice kind="success">{actionDone}</Notice>}
      {error && <ErrorBox message={error} onRetry={reload} />}

      <div className="order-layout">
        <div className="stack-lg">
          <StatusPanel order={order} busy={busy} onPay={startPayment} onCancel={() => runAction('cancel')} onRefund={() => runAction('refund')} />

          <section>
            <h2 className="section-title">{order.status === 'paid' ? 'Your e-tickets' : 'Tickets'}</h2>
            <div className="ticket-list">
              {(order.tickets ?? []).map((t) => (
                <TicketCard key={t.id} ticket={t} showQr={order.status === 'paid' && t.active} />
              ))}
            </div>
          </section>
        </div>

        <aside className="panel summary">
          <h2>Details</h2>
          <dl className="details">
            <div>
              <dt>Status</dt>
              <dd>
                <StatusBadge status={order.status} />
              </dd>
            </div>
            <div>
              <dt>Payment</dt>
              <dd>{order.paymentMethod === 'card' ? 'Bank card' : 'Cash at the box office'}</dd>
            </div>
            <div>
              <dt>Delivery</dt>
              <dd>
                {order.delivery.option === 'courier' ? (
                  <>
                    Courier · {order.delivery.address}
                    {order.delivery.status && <span className="muted small block">Delivery {order.delivery.status}{order.delivery.courierRef ? ` (${order.delivery.courierRef})` : ''}</span>}
                  </>
                ) : (
                  'E-ticket'
                )}
              </dd>
            </div>
            {order.paidAt && (
              <div>
                <dt>Paid</dt>
                <dd>{formatTimestamp(order.paidAt)}</dd>
              </div>
            )}
            <div>
              <dt>Tickets</dt>
              <dd>{plural(order.tickets?.length ?? 0, 'ticket')}</dd>
            </div>
            {order.delivery.fee > 0 && (
              <div>
                <dt>Delivery fee</dt>
                <dd>{formatMoney(order.delivery.fee)}</dd>
              </div>
            )}
          </dl>
          <div className="summary-total">
            <span>Total</span>
            <strong>{formatMoney(order.total)}</strong>
          </div>
        </aside>
      </div>
    </div>
  );
}

function PaymentFeedback({ order, onClose }: { order: Order; onClose: () => void }) {
  if (order.status === 'paid') {
    return (
      <Notice kind="success">
        Payment received — thank you! {order.delivery.option === 'courier' ? 'The courier has been requested.' : 'Your e-tickets are below.'}{' '}
        <button type="button" className="link-btn" onClick={onClose}>
          Dismiss
        </button>
      </Notice>
    );
  }
  if (order.status === 'pending_payment' || order.status === 'booked') {
    return <Notice kind="info">Waiting for the payment provider to confirm your payment…</Notice>;
  }
  return <Notice kind="warning">The payment arrived after the order had {order.status === 'expired' ? 'expired' : 'closed'}; it will be refunded automatically.</Notice>;
}

function StatusPanel({
  order,
  busy,
  onPay,
  onCancel,
  onRefund,
}: {
  order: Order;
  busy: string;
  onPay: () => void;
  onCancel: () => void;
  onRefund: () => void;
}) {
  const now = useNow(1000);

  switch (order.status) {
    case 'pending_payment': {
      const left = order.holdExpiresAt ? new Date(order.holdExpiresAt).getTime() - now : 0;
      const expired = left <= 0;
      return (
        <section className="panel status-panel">
          <h2>Complete your payment</h2>
          {expired ? (
            <p>The 10-minute seat hold has run out. The seats are being returned to sale.</p>
          ) : (
            <>
              <p>Your seats are held while you pay.</p>
              <div className={`countdown${left < 60_000 ? ' urgent' : ''}`} aria-live="off">
                <span className="countdown-value">{formatCountdown(left)}</span>
                <span className="muted small">left to pay (until {formatDateTime(order.holdExpiresAt)})</span>
              </div>
            </>
          )}
          <div className="btn-row">
            <button type="button" className="btn btn-primary" onClick={onPay} disabled={!!busy || expired}>
              {busy === 'pay' ? 'Redirecting…' : `Pay now ${formatMoney(order.total)}`}
            </button>
            <button type="button" className="btn btn-ghost" onClick={onCancel} disabled={!!busy}>
              {busy === 'cancel' ? 'Cancelling…' : 'Cancel order'}
            </button>
          </div>
        </section>
      );
    }
    case 'booked': {
      const deadlinePassed = new Date(order.bookingDeadline).getTime() <= now;
      const courier = order.delivery.option === 'courier';
      return (
        <section className="panel status-panel">
          <h2>Your booking</h2>
          <div className="booking-code">
            <span className="muted small">Booking code</span>
            <code>{order.bookingCode}</code>
          </div>
          <p>
            Pay before <strong>{formatDateTime(order.bookingDeadline)}</strong> —{' '}
            {courier
              ? 'online by card (courier delivery requires a card payment).'
              : 'online by card, or in cash at the box office by showing this code.'}{' '}
            After the deadline the booking expires and the seats return to sale.
          </p>
          {deadlinePassed && <Notice kind="warning">The booking deadline has passed.</Notice>}
          <div className="btn-row">
            <button type="button" className="btn btn-primary" onClick={onPay} disabled={!!busy || deadlinePassed}>
              {busy === 'pay' ? 'Redirecting…' : `Pay online ${formatMoney(order.total)}`}
            </button>
            <button type="button" className="btn btn-danger-ghost" onClick={onCancel} disabled={!!busy}>
              {busy === 'cancel' ? 'Cancelling…' : 'Cancel booking'}
            </button>
          </div>
        </section>
      );
    }
    case 'paid': {
      const refundable = order.paymentMethod === 'card' && new Date(order.bookingDeadline).getTime() > now;
      return (
        <section className="panel status-panel">
          <h2>Paid</h2>
          <p>
            {order.delivery.option === 'courier'
              ? 'Your tickets will be delivered by courier. The e-tickets below are also valid at the entrance.'
              : 'Show the QR code of each ticket at the entrance — on your phone or printed.'}
          </p>
          {order.paymentMethod === 'card' && (
            <div className="btn-row">
              <button type="button" className="btn btn-danger-ghost" onClick={onRefund} disabled={!!busy || !refundable}>
                {busy === 'refund' ? 'Refunding…' : 'Request refund'}
              </button>
              <span className="muted small">
                {refundable
                  ? `Refunds are possible until ${formatDateTime(order.bookingDeadline)} (24 h before the session).`
                  : 'Refunds close 24 hours before the session.'}
              </span>
            </div>
          )}
          {order.paymentMethod === 'cash' && <p className="muted small">Cash payments are refunded at the box office only.</p>}
        </section>
      );
    }
    case 'cancelled':
      return (
        <section className="panel status-panel">
          <h2>Cancelled</h2>
          <p>This order was cancelled and its seats were released.</p>
        </section>
      );
    case 'expired':
      return (
        <section className="panel status-panel">
          <h2>Expired</h2>
          <p>This order was not paid in time, so its seats were returned to sale.</p>
        </section>
      );
    case 'refunded':
      return (
        <section className="panel status-panel">
          <h2>Refunded</h2>
          <p>The money was returned to your card. The tickets are no longer valid.</p>
        </section>
      );
  }
}
