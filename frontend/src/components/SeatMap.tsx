import { useMemo, type CSSProperties } from 'react';
import type { SeatState, ZonePrice } from '../api';
import { formatMoney } from '../format';

export const ZONE_COLORS = ['#7c3aed', '#0891b2', '#d97706', '#db2777', '#16a34a', '#2563eb', '#b45309', '#9333ea'];

/** zoneColorMap assigns each zone a stable colour in the order of the session's price list. */
export function zoneColorMap(prices: ZonePrice[], seats: SeatState[]): Map<number, string> {
  const ids: number[] = [];
  for (const p of prices) if (!ids.includes(p.zoneId)) ids.push(p.zoneId);
  for (const s of seats) if (!ids.includes(s.zoneId)) ids.push(s.zoneId);
  return new Map(ids.map((id, i) => [id, ZONE_COLORS[i % ZONE_COLORS.length]]));
}

interface Props {
  seats: SeatState[];
  selected: Set<number>;
  colors: Map<number, string>;
  stageLabel: string;
  onToggle: (seat: SeatState) => void;
}

const STATUS_TEXT = { free: 'free', held: 'reserved', sold: 'sold' } as const;

export function SeatMap({ seats, selected, colors, stageLabel, onToggle }: Props) {
  const rows = useMemo(() => {
    const byRow = new Map<number, SeatState[]>();
    for (const s of seats) {
      const list = byRow.get(s.row) ?? [];
      list.push(s);
      byRow.set(s.row, list);
    }
    return [...byRow.entries()]
      .sort(([a], [b]) => a - b)
      .map(([row, list]) => ({ row, seats: list.sort((a, b) => a.number - b.number) }));
  }, [seats]);

  let prevZone = '';
  return (
    <div className="seatmap-scroll">
      <div className="seatmap" role="group" aria-label="Seat map">
        <div className="stage">{stageLabel}</div>
        {rows.map(({ row, seats: rowSeats }) => {
          const zone = rowSeats[0]?.zoneName ?? '';
          const showZone = zone !== prevZone;
          prevZone = zone;
          return (
            <div key={row}>
              {showZone && (
                <div className="zone-divider" style={{ color: colors.get(rowSeats[0].zoneId) }}>
                  <span>{zone}</span>
                </div>
              )}
              <div className="seat-row">
                <span className="row-label" aria-hidden="true">
                  {row}
                </span>
                <div className="seats">
                  {rowSeats.map((s) => {
                    const isSelected = selected.has(s.id);
                    const available = s.status === 'free';
                    return (
                      <button
                        key={s.id}
                        type="button"
                        className={`seat seat-${s.status}${isSelected ? ' seat-selected' : ''}`}
                        style={{ '--zone': colors.get(s.zoneId) } as CSSProperties}
                        disabled={!available && !isSelected}
                        aria-pressed={isSelected}
                        aria-label={`Row ${s.row}, seat ${s.number}, ${s.zoneName}, ${formatMoney(s.price)}, ${isSelected ? 'selected' : STATUS_TEXT[s.status]}`}
                        title={`Row ${s.row}, seat ${s.number} · ${s.zoneName} · ${formatMoney(s.price)}`}
                        onClick={() => onToggle(s)}
                      >
                        <span aria-hidden="true">{s.number}</span>
                      </button>
                    );
                  })}
                </div>
                <span className="row-label row-label-end" aria-hidden="true">
                  {row}
                </span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export function SeatLegend({ prices, colors }: { prices: ZonePrice[]; colors: Map<number, string> }) {
  return (
    <div className="legend">
      <ul className="legend-zones" aria-label="Price zones">
        {prices.map((p) => (
          <li key={p.zoneId}>
            <span className="legend-swatch" style={{ background: colors.get(p.zoneId) }} />
            <span>{p.zoneName}</span>
            <strong>{formatMoney(p.price)}</strong>
          </li>
        ))}
      </ul>
      <ul className="legend-states" aria-label="Seat states">
        <li>
          <span className="seat seat-demo seat-free" style={{ '--zone': '#64748b' } as CSSProperties} /> Free
        </li>
        <li>
          <span className="seat seat-demo seat-selected" /> Selected
        </li>
        <li>
          <span className="seat seat-demo seat-held" /> Reserved
        </li>
        <li>
          <span className="seat seat-demo seat-sold" /> Sold
        </li>
      </ul>
    </div>
  );
}
