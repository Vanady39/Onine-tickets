import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import type { EventType, Money, OrderItem } from './api';

/** MAX_PER_EVENT mirrors BR-2: at most 3 tickets for one event in one order. */
export const MAX_PER_EVENT = 3;

/** CartItem is one ticket the customer is about to order, with what is needed to display it. */
export interface CartItem {
  key: string;
  sessionId: number;
  seatId?: number;
  eventId: number;
  eventTitle: string;
  eventType: EventType;
  startsAt: string;
  venueName: string;
  hallName: string;
  /** "Parterre · Row 3, Seat 7" or "General admission". */
  place: string;
  price: Money;
  addOns: { id: number; name: string; price: Money }[];
}

export type CheckoutSource = 'single' | 'season';

export interface CheckoutDraft {
  source: CheckoutSource;
  items: CartItem[];
  /** Page to go back to for changing the selection. */
  backTo: string;
}

interface CartState {
  draft: CheckoutDraft | null;
  startCheckout: (draft: CheckoutDraft) => void;
  clearDraft: () => void;
  season: CartItem[];
  /** Adds tickets to the season-ticket cart; returns an error message if a rule is broken. */
  addToSeason: (items: CartItem[]) => string | null;
  removeFromSeason: (key: string) => void;
  clearSeason: () => void;
}

const DRAFT_KEY = 'ot.checkout';
const SEASON_KEY = 'ot.season';

function load<T>(storage: () => Storage, key: string, fallback: T): T {
  try {
    const raw = storage().getItem(key);
    return raw ? (JSON.parse(raw) as T) : fallback;
  } catch {
    return fallback;
  }
}

function save(storage: () => Storage, key: string, value: unknown): void {
  try {
    if (value === null) storage().removeItem(key);
    else storage().setItem(key, JSON.stringify(value));
  } catch {
    /* storage unavailable or full: state lives in memory only */
  }
}

const session = () => sessionStorage;
const local = () => localStorage;

export function itemTotal(item: CartItem): Money {
  return item.price + item.addOns.reduce((s, a) => s + a.price, 0);
}

export function itemsTotal(items: CartItem[]): Money {
  return items.reduce((s, i) => s + itemTotal(i), 0);
}

export function toOrderItems(items: CartItem[]): OrderItem[] {
  return items.map((i) => ({
    sessionId: i.sessionId,
    ...(i.seatId ? { seatId: i.seatId } : {}),
    ...(i.addOns.length ? { addOnIds: i.addOns.map((a) => a.id) } : {}),
  }));
}

/** countByEvent returns how many tickets of each event the items contain. */
export function countByEvent(items: CartItem[]): Map<number, number> {
  const m = new Map<number, number>();
  for (const i of items) m.set(i.eventId, (m.get(i.eventId) ?? 0) + 1);
  return m;
}

const CartContext = createContext<CartState | null>(null);

export function CartProvider({ children }: { children: ReactNode }) {
  // The checkout draft survives the sign-in redirect but not the browser tab.
  const [draft, setDraft] = useState<CheckoutDraft | null>(() => load(session, DRAFT_KEY, null));
  const [season, setSeason] = useState<CartItem[]>(() => load(local, SEASON_KEY, []));

  useEffect(() => {
    save(session, DRAFT_KEY, draft);
  }, [draft]);
  useEffect(() => {
    save(local, SEASON_KEY, season);
  }, [season]);

  const startCheckout = useCallback((d: CheckoutDraft) => setDraft(d), []);
  const clearDraft = useCallback(() => setDraft(null), []);

  const addToSeason = useCallback(
    (items: CartItem[]) => {
      const keys = new Set(season.map((i) => i.key));
      const fresh = items.filter((i) => !keys.has(i.key));
      if (!fresh.length) return 'These tickets are already in your season ticket.';
      const counts = countByEvent([...season, ...fresh]);
      for (const [eventId, n] of counts) {
        if (n > MAX_PER_EVENT) {
          const title = fresh.find((i) => i.eventId === eventId)?.eventTitle ?? 'this event';
          return `A season ticket can hold at most ${MAX_PER_EVENT} tickets for ${title}.`;
        }
      }
      setSeason([...season, ...fresh]);
      return null;
    },
    [season],
  );

  const removeFromSeason = useCallback((key: string) => setSeason((s) => s.filter((i) => i.key !== key)), []);
  const clearSeason = useCallback(() => setSeason([]), []);

  const value = useMemo(
    () => ({ draft, startCheckout, clearDraft, season, addToSeason, removeFromSeason, clearSeason }),
    [draft, startCheckout, clearDraft, season, addToSeason, removeFromSeason, clearSeason],
  );
  return <CartContext.Provider value={value}>{children}</CartContext.Provider>;
}

export function useCart(): CartState {
  const ctx = useContext(CartContext);
  if (!ctx) throw new Error('useCart must be used inside <CartProvider>');
  return ctx;
}
