import { QRCodeSVG } from 'qrcode.react';
import type { Ticket } from '../api';
import { formatDateTime, formatMoney, ticketPlace } from '../format';
import { TypeBadge } from './ui';

/** TicketCard is an electronic ticket: the QR code encodes the unique ticket code (FR-28). */
export function TicketCard({ ticket, showQr }: { ticket: Ticket; showQr: boolean }) {
  const addOns = ticket.addOns ?? [];
  return (
    <article className={`eticket type-edge-${ticket.eventType}${ticket.active ? '' : ' eticket-inactive'}`}>
      <div className="eticket-main">
        <div className="eticket-top">
          <TypeBadge type={ticket.eventType} />
          {!ticket.active && <span className="badge status status-cancelled">Not valid</span>}
        </div>
        <h3>{ticket.eventTitle}</h3>
        <dl className="eticket-facts">
          <div>
            <dt>When</dt>
            <dd>{formatDateTime(ticket.startsAt)}</dd>
          </div>
          <div>
            <dt>Where</dt>
            <dd>
              {ticket.venueName} · {ticket.hallName}
            </dd>
          </div>
          <div>
            <dt>Place</dt>
            <dd>{ticketPlace(ticket)}</dd>
          </div>
          <div>
            <dt>Price</dt>
            <dd>{formatMoney(ticket.price)}</dd>
          </div>
        </dl>
        {addOns.length > 0 && (
          <ul className="eticket-addons">
            {addOns.map((a) => (
              <li key={a.addOnId}>
                + {a.name} <span className="muted">({formatMoney(a.price)})</span>
              </li>
            ))}
          </ul>
        )}
      </div>
      {showQr && (
        <div className="eticket-qr">
          <QRCodeSVG value={ticket.code} size={112} level="M" marginSize={0} aria-label={`QR code of ticket ${ticket.code}`} />
          <code className="ticket-code">{ticket.code}</code>
        </div>
      )}
    </article>
  );
}
