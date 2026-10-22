import type { AddOnKind, EventType, Money, OrderStatus, Role, Ticket } from './api';

const NBSP = ' ';

/** formatMoney renders kopecks as e.g. "1 800 ₽" (or "1 800,50 ₽" when there are kopecks). */
export function formatMoney(kopecks: Money): string {
  const negative = kopecks < 0;
  const abs = Math.abs(Math.round(kopecks));
  const roubles = Math.floor(abs / 100);
  const rest = abs % 100;
  const grouped = String(roubles).replace(/\B(?=(\d{3})+(?!\d))/g, NBSP);
  const value = rest ? `${grouped},${String(rest).padStart(2, '0')}` : grouped;
  return `${negative ? '−' : ''}${value}${NBSP}₽`;
}

/** roublesToKopecks converts a user-entered rouble amount ("1800", "1800.50") to kopecks. */
export function roublesToKopecks(value: string): number {
  const n = Number(value.replace(',', '.').replace(/\s/g, ''));
  return Number.isFinite(n) ? Math.round(n * 100) : NaN;
}

const dateTimeFmt = new Intl.DateTimeFormat('en-GB', {
  weekday: 'short',
  day: 'numeric',
  month: 'short',
  hour: '2-digit',
  minute: '2-digit',
});
const dateFmt = new Intl.DateTimeFormat('en-GB', { weekday: 'short', day: 'numeric', month: 'short' });
const longDateFmt = new Intl.DateTimeFormat('en-GB', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });
const timeFmt = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit' });
const fullFmt = new Intl.DateTimeFormat('en-GB', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
});

/** "Sat 25 Oct, 19:00" in the user's time zone. */
export function formatDateTime(iso: string | undefined | null): string {
  if (!iso) return '—';
  return dateTimeFmt.format(new Date(iso));
}

/** "Sat 25 Oct" */
export function formatDate(iso: string): string {
  return dateFmt.format(new Date(iso));
}

/** "Saturday, 25 October 2026" */
export function formatLongDate(iso: string): string {
  return longDateFmt.format(new Date(iso));
}

/** "19:00" */
export function formatTime(iso: string): string {
  return timeFmt.format(new Date(iso));
}

/** "25 Oct 2026, 19:00" — for timestamps such as order creation. */
export function formatTimestamp(iso: string | undefined | null): string {
  if (!iso) return '—';
  return fullFmt.format(new Date(iso));
}

export function formatDuration(min: number): string {
  if (!min) return '';
  const h = Math.floor(min / 60);
  const m = min % 60;
  if (!h) return `${m} min`;
  return m ? `${h} h ${m} min` : `${h} h`;
}

export const EVENT_TYPE_LABEL: Record<EventType, string> = {
  concert: 'Concert',
  cinema: 'Cinema',
  theatre: 'Theatre',
  exhibition: 'Exhibition',
};

export const ROLE_LABEL: Record<Role, string> = {
  customer: 'Customer',
  cashier: 'Cashier',
  manager: 'Event manager',
  admin: 'Administrator',
};

export const ORDER_STATUS_LABEL: Record<OrderStatus, string> = {
  pending_payment: 'Awaiting payment',
  booked: 'Booked',
  paid: 'Paid',
  cancelled: 'Cancelled',
  expired: 'Expired',
  refunded: 'Refunded',
};

export const ADD_ON_KIND_LABEL: Record<AddOnKind, string> = {
  business_lunch: 'Business lunch',
  masterclass: 'Masterclass',
  presentation: 'Presentation',
};

export function plural(n: number, one: string, many = `${one}s`): string {
  return `${n} ${n === 1 ? one : many}`;
}

/** Human-readable place of a ticket: "Parterre · Row 3, Seat 7" or "General admission". */
export function ticketPlace(t: Pick<Ticket, 'zoneName' | 'row' | 'number'>): string {
  if (t.row && t.number) return `${t.zoneName ? `${t.zoneName} · ` : ''}Row ${t.row}, Seat ${t.number}`;
  return 'General admission';
}

/** Date as YYYY-MM-DD in the local time zone (for <input type="date">). */
export function toDateInput(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

/** Converts the value of <input type="datetime-local"> (local time) to RFC 3339. */
export function localInputToRFC3339(value: string): string {
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? '' : d.toISOString();
}

/** mm:ss for a countdown. */
export function formatCountdown(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  const m = Math.floor(total / 60);
  const s = total % 60;
  return `${m}:${String(s).padStart(2, '0')}`;
}

export const HOUR = 60 * 60 * 1000;
export const BOOKING_WINDOW_MS = 24 * HOUR;
