// Typed client for the Online Tickets REST API (backend/internal/httpapi).
// Types mirror the JSON tags of backend/internal/domain/*.go.
// Money values are integers in kopecks; times are RFC 3339 strings.

export type Money = number;

export type EventType = 'concert' | 'cinema' | 'theatre' | 'exhibition';
export const EVENT_TYPES: EventType[] = ['concert', 'cinema', 'theatre', 'exhibition'];

export type Role = 'customer' | 'cashier' | 'manager' | 'admin';

export type HallKind = 'seated' | 'general_admission';

export interface PriceZone {
  id: number;
  name: string;
}

export interface Hall {
  id: number;
  venueId: number;
  venueName: string;
  address: string;
  name: string;
  kind: HallKind;
  capacity: number;
  zones?: PriceZone[];
}

export interface Venue {
  id: number;
  name: string;
  address: string;
  halls?: Hall[];
}

export interface ZonePrice {
  zoneId: number;
  zoneName: string;
  price: Money;
}

export type AddOnKind = 'business_lunch' | 'masterclass' | 'presentation';

export interface AddOnService {
  id: number;
  kind: AddOnKind;
  name: string;
  price: Money;
  capacity: number;
  remaining: number;
}

export interface Session {
  id: number;
  eventId: number;
  eventTitle: string;
  eventType: EventType;
  hall: Hall;
  startsAt: string;
  basePrice: Money;
  prices: ZonePrice[] | null;
  addOns: AddOnService[] | null;
  remaining: number;
}

export interface Event {
  id: number;
  title: string;
  type: EventType;
  description: string;
  durationMin: number;
  ageRating: string;
  imageUrl: string;
  sessions?: Session[];
  nextSessionAt?: string;
  minPrice?: Money;
}

export type SeatStatus = 'free' | 'held' | 'sold';

export interface SeatState {
  id: number;
  zoneId: number;
  row: number;
  number: number;
  zoneName: string;
  price: Money;
  status: SeatStatus;
}

export interface User {
  id: number;
  email: string;
  name: string;
  role: Role;
  createdAt?: string;
}

export interface Profile extends User {
  paidOrders: number;
  loyal: boolean;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export type OrderStatus = 'pending_payment' | 'booked' | 'paid' | 'cancelled' | 'expired' | 'refunded';
export type PaymentMethod = 'card' | 'cash';
export type DeliveryOption = 'e_ticket' | 'courier';

export interface Delivery {
  option: DeliveryOption;
  address?: string;
  fee: Money;
  courierRef?: string;
  status?: string;
}

export interface TicketAddOn {
  addOnId: number;
  name: string;
  price: Money;
}

export interface Ticket {
  id: number;
  code: string;
  sessionId: number;
  eventId: number;
  seatId?: number;
  price: Money;
  active: boolean;
  addOns: TicketAddOn[] | null;
  eventTitle: string;
  eventType: EventType;
  startsAt: string;
  venueName: string;
  hallName: string;
  zoneName?: string;
  row?: number;
  number?: number;
}

export interface Order {
  id: number;
  customerId: number;
  status: OrderStatus;
  paymentMethod: PaymentMethod;
  bookingCode?: string;
  bookingDeadline: string;
  holdExpiresAt?: string;
  total: Money;
  delivery: Delivery;
  tickets: Ticket[] | null;
  createdAt: string;
  paidAt?: string;
}

export interface OrderItem {
  sessionId: number;
  seatId?: number;
  addOnIds?: number[];
}

export interface OrderRequest {
  items: OrderItem[];
  paymentMethod: PaymentMethod;
  book: boolean;
  deliveryOption: DeliveryOption;
  deliveryAddress?: string;
}

export interface EventInput {
  title: string;
  type: EventType;
  description: string;
  durationMin: number;
  ageRating: string;
  imageUrl: string;
}

export interface NewSessionAddOn {
  kind: AddOnKind;
  name: string;
  price: Money;
  capacity: number;
}

export interface NewSession {
  eventId: number;
  hallId: number;
  startsAt: string;
  basePrice: Money;
  prices: { zoneId: number; price: Money }[];
  addOns: NewSessionAddOn[];
}

export interface ZoneSpec {
  name: string;
  rows: number;
  seatsPerRow: number;
}

export interface HallSpec {
  name: string;
  kind: HallKind;
  capacity: number;
  zones: ZoneSpec[];
}

export interface NewVenue {
  name: string;
  address: string;
  halls: HallSpec[];
}

export interface SessionReport {
  sessionId: number;
  startsAt: string;
  hallName: string;
  venueName: string;
  capacity: number;
  ticketsSold: number;
  revenue: Money;
  occupancy: number; // percent of capacity sold
}

export interface EventReport {
  eventId: number;
  title: string;
  ticketsSold: number;
  revenue: Money;
  sessions: SessionReport[] | null;
}

export interface EventFilter {
  type?: EventType | '';
  from?: string; // YYYY-MM-DD
  to?: string; // YYYY-MM-DD
}

// --- transport

const TOKEN_KEY = 'ot.token';

export function getToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

export function setToken(token: string | null): void {
  try {
    if (token) localStorage.setItem(TOKEN_KEY, token);
    else localStorage.removeItem(TOKEN_KEY);
  } catch {
    /* storage unavailable: the session lasts until reload */
  }
}

/** ApiError carries the HTTP status and the server's {"error": "..."} message. */
export class ApiError extends Error {
  readonly status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

let unauthorizedHandler: (() => void) | null = null;

/** Called when a request that carried a token gets 401 (token missing, invalid or expired). */
export function onUnauthorized(handler: (() => void) | null): void {
  unauthorizedHandler = handler;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  // Sign-in endpoints never carry a (possibly stale) token, so their 401 means wrong credentials.
  const token = path.startsWith('/api/auth/') ? null : getToken();
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (token) headers.Authorization = `Bearer ${token}`;

  let res: Response;
  try {
    res = await fetch(path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, 'Cannot reach the server. Check your connection and try again.');
  }

  const text = await res.text();
  let data: unknown = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = null;
    }
  }

  if (!res.ok) {
    const message =
      data && typeof data === 'object' && 'error' in data && typeof (data as { error: unknown }).error === 'string'
        ? (data as { error: string }).error
        : defaultMessage(res.status);
    if (res.status === 401 && token) {
      setToken(null);
      unauthorizedHandler?.();
    }
    throw new ApiError(res.status, message);
  }
  return data as T;
}

function defaultMessage(status: number): string {
  switch (status) {
    case 401:
      return 'Please sign in to continue.';
    case 403:
      return 'This action is not allowed for your role.';
    case 404:
      return 'Not found.';
    case 409:
      return 'Some of the selected places are no longer available.';
    default:
      return status >= 500 ? 'The server failed to process the request. Please try again later.' : `Request failed (${status}).`;
  }
}

function query(params: Record<string, string | undefined>): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v) q.set(k, v);
  const s = q.toString();
  return s ? `?${s}` : '';
}

const get = <T,>(path: string) => request<T>('GET', path);
const post = <T,>(path: string, body?: unknown) => request<T>('POST', path, body ?? {});
const put = <T,>(path: string, body: unknown) => request<T>('PUT', path, body);
const enc = encodeURIComponent;

export const api = {
  // Catalogue
  events: (f: EventFilter = {}) =>
    get<Event[] | null>(`/api/events${query({ type: f.type || undefined, from: f.from, to: f.to })}`).then((r) => r ?? []),
  event: (id: number) => get<Event>(`/api/events/${id}`),
  session: (id: number) => get<Session>(`/api/sessions/${id}`),
  seats: (id: number) => get<SeatState[] | null>(`/api/sessions/${id}/seats`).then((r) => r ?? []),

  // Accounts
  register: (name: string, email: string, password: string) =>
    post<AuthResponse>('/api/auth/register', { name, email, password }),
  login: (email: string, password: string) => post<AuthResponse>('/api/auth/login', { email, password }),
  me: () => get<Profile>('/api/me'),

  // Orders
  placeOrder: (req: OrderRequest) => post<Order>('/api/orders', req),
  orders: () => get<Order[] | null>('/api/orders').then((r) => r ?? []),
  order: (id: number) => get<Order>(`/api/orders/${id}`),
  pay: (id: number) => post<{ redirectUrl: string }>(`/api/orders/${id}/pay`),
  cancel: (id: number) => post<Order>(`/api/orders/${id}/cancel`),
  refund: (id: number) => post<Order>(`/api/orders/${id}/refund`),

  // Box office
  findBooking: (code: string) => get<Order>(`/api/boxoffice/bookings/${enc(code)}`),
  payCash: (code: string) => post<Order>(`/api/boxoffice/bookings/${enc(code)}/pay-cash`),

  // Back office
  adminEvents: () => get<Event[] | null>('/api/admin/events').then((r) => r ?? []),
  createEvent: (e: EventInput) => post<Event>('/api/admin/events', e),
  updateEvent: (id: number, e: EventInput) => put<Event>(`/api/admin/events/${id}`, e),
  report: (eventId: number) => get<EventReport>(`/api/admin/events/${eventId}/report`),
  venues: () => get<Venue[] | null>('/api/admin/venues').then((r) => r ?? []),
  createSession: (s: NewSession) => post<{ id: number }>('/api/admin/sessions', s),
  createVenue: (v: NewVenue) => post<{ id: number }>('/api/admin/venues', v),
};

export function errorMessage(err: unknown): string {
  if (err instanceof Error) return err.message;
  return 'Something went wrong.';
}
