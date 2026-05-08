'use client';

// Thailand province tile-map (F-01 / F-03).
//
// Each province is rendered as a coloured hexagonal tile arranged in a
// geographic grid approximating Thailand's shape. The fill colour comes from
// the leading party in the constituency ballot.
//
// A real SVG with exact province outlines can be swapped in later by replacing
// this component while keeping the same props contract.

import { useEffect, useMemo, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useUIStore } from '@/lib/store/ui';
import { useNationalSummary, useProvinces, queryKeys } from '@/lib/api/client';
import { subscribeChannel, Channels } from '@/lib/ws/centrifuge';
import type { NationalSummary, Province } from '@/lib/types/election';
import { SkeletonCard } from '@/components/shared/SkeletonCard';

// ── Geographic grid layout ─────────────────────────────────────────────────────
// [row, col, provinceId, shortLabel]
// Approximates Thailand's geography; North is top.

const GRID: [number, number, number, string][] = [
  // Northern Thailand
  [0, 2, 57, 'เชียงราย'],
  [0, 3, 58, 'น่าน'],
  [1, 1, 53, 'แม่ฮ่องสอน'],
  [1, 2, 50, 'เชียงใหม่'],
  [1, 3, 54, 'แพร่'],
  [1, 4, 55, 'น่าน'],
  [2, 2, 51, 'ลำพูน'],
  [2, 3, 52, 'ลำปาง'],
  [2, 4, 63, 'ตาก'],
  [3, 1, 56, 'พะเยา'],
  [3, 3, 64, 'สุโขทัย'],
  [3, 4, 65, 'อุตรดิตถ์'],
  // Central North
  [4, 2, 60, 'นครสวรรค์'],
  [4, 3, 66, 'พิษณุโลก'],
  [4, 4, 67, 'เพชรบูรณ์'],
  [5, 1, 61, 'อุทัยธานี'],
  [5, 2, 62, 'กำแพงเพชร'],
  [5, 3, 65, 'พิจิตร'],
  // Northeast (Isan)
  [2, 5, 43, 'เลย'],
  [2, 6, 41, 'หนองบัวลำภู'],
  [2, 7, 42, 'อุดรธานี'],
  [2, 8, 44, 'นครพนม'],
  [3, 5, 39, 'ขอนแก่น'],
  [3, 6, 40, 'สกลนคร'],
  [3, 7, 48, 'นครพนม'],
  [4, 5, 30, 'นครราชสีมา'],
  [4, 6, 45, 'มหาสารคาม'],
  [4, 7, 47, 'อำนาจเจริญ'],
  [4, 8, 49, 'มุกดาหาร'],
  [5, 5, 31, 'บุรีรัมย์'],
  [5, 6, 32, 'สุรินทร์'],
  [5, 7, 33, 'ศรีสะเกษ'],
  [5, 8, 34, 'อุบลราชธานี'],
  // Central
  [6, 1, 18, 'ชัยนาท'],
  [6, 2, 19, 'สระบุรี'],
  [6, 3, 26, 'นครนายก'],
  [6, 4, 25, 'ปราจีนบุรี'],
  [7, 1, 72, 'สุพรรณบุรี'],
  [7, 2, 14, 'พระนครศรีอยุธยา'],
  [7, 3, 13, 'ลพบุรี'],
  [7, 4, 27, 'สระแก้ว'],
  [8, 0, 71, 'กาญจนบุรี'],
  [8, 1, 73, 'นครปฐม'],
  [8, 2, 10, 'กรุงเทพมหานคร'],
  [8, 3, 11, 'สมุทรปราการ'],
  [8, 4, 20, 'ชลบุรี'],
  [8, 5, 21, 'ระยอง'],
  [8, 6, 22, 'จันทบุรี'],
  [8, 7, 23, 'ตราด'],
  [9, 1, 74, 'สมุทรสาคร'],
  [9, 2, 75, 'สมุทรสงคราม'],
  [9, 3, 76, 'เพชรบุรี'],
  // South
  [10, 2, 77, 'ประจวบคีรีขันธ์'],
  [11, 1, 85, 'ระนอง'],
  [11, 2, 70, 'ชุมพร'],
  [11, 3, 84, 'สุราษฎร์ธานี'],
  [12, 1, 82, 'กระบี่'],
  [12, 2, 86, 'นครศรีธรรมราช'],
  [12, 3, 80, 'พัทลุง'],
  [13, 0, 83, 'พังงา'],
  [13, 1, 81, 'ตรัง'],
  [13, 2, 90, 'สงขลา'],
  [14, 0, 83, 'ภูเก็ต'],
  [14, 1, 91, 'สตูล'],
  [14, 2, 92, 'ปัตตานี'],
  [15, 1, 93, 'ยะลา'],
  [15, 2, 94, 'นราธิวาส'],
];

const COLS = 10;
const ROWS = 16;
const TILE = 36; // px per tile
const GAP = 2;

interface ProvinceColors {
  [provinceId: number]: string; // hex colour of the leading party
}

function buildColors(
  summary: NationalSummary | undefined,
  provinces: Province[] | undefined
): ProvinceColors {
  if (!summary || !provinces) return {};
  // For the tile map we colour by leading national party as a rough approximation.
  // In a real implementation each province would have its own leading party colour.
  const top = summary.parties[0];
  if (!top) return {};
  return Object.fromEntries(provinces.map((p) => [p.id, top.party_color]));
}

interface Props {
  /** Initial national summary, fetched server-side. May be undefined on error. */
  initialSummary?: NationalSummary;
  /** Initial province list, fetched server-side. */
  initialProvinces?: Province[];
}

export function ThailandMap({ initialSummary, initialProvinces }: Props) {
  const queryClient = useQueryClient();
  const { selectedProvinceId, setSelectedProvinceId } = useUIStore();

  const { data: summary } = useNationalSummary({ initialData: initialSummary });
  const { data: provinces, isPending } = useProvinces({ initialData: initialProvinces });

  // Update cache when Centrifugo pushes a national update
  useEffect(() => {
    let cleanup: (() => void) | undefined;
    subscribeChannel(Channels.national, () => {
      // Invalidate; TanStack Query will refetch
      void queryClient.invalidateQueries({ queryKey: queryKeys.nationalSummary() });
    }).then((fn) => {
      cleanup = fn;
    });
    return () => cleanup?.();
  }, [queryClient]);

  const provinceColors = useMemo(
    () => buildColors(summary, provinces),
    [summary, provinces]
  );

  const handleTileClick = useCallback(
    (id: number) => {
      setSelectedProvinceId(selectedProvinceId === id ? null : id);
    },
    [selectedProvinceId, setSelectedProvinceId]
  );

  if (isPending) return <SkeletonCard rows={6} className="h-64" />;

  const svgW = COLS * (TILE + GAP);
  const svgH = ROWS * (TILE + GAP);

  return (
    <div className="overflow-auto rounded-lg border border-border bg-card p-3">
      <h2 className="text-sm font-semibold text-muted-foreground mb-2 px-1">
        Province Map — Leading Party
      </h2>
      <svg
        viewBox={`0 0 ${svgW} ${svgH}`}
        width={svgW}
        height={svgH}
        aria-label="Thailand province tile map"
        role="img"
      >
        {GRID.map(([row, col, provinceId, label]) => {
          const x = col * (TILE + GAP);
          const y = row * (TILE + GAP);
          const color = provinceColors[provinceId] ?? '#d1d5db';
          const isSelected = selectedProvinceId === provinceId;

          return (
            <g
              key={`${provinceId}-${row}-${col}`}
              onClick={() => handleTileClick(provinceId)}
              className="cursor-pointer"
              aria-label={label}
              role="button"
              tabIndex={0}
              onKeyDown={(e) => e.key === 'Enter' && handleTileClick(provinceId)}
            >
              <rect
                x={x}
                y={y}
                width={TILE}
                height={TILE}
                rx={4}
                fill={color}
                opacity={isSelected ? 1 : 0.75}
                stroke={isSelected ? '#1d4ed8' : 'transparent'}
                strokeWidth={isSelected ? 2 : 0}
                className="transition-opacity hover:opacity-100"
              />
              <text
                x={x + TILE / 2}
                y={y + TILE / 2 + 4}
                fontSize={7}
                textAnchor="middle"
                fill="#fff"
                fontWeight={500}
                style={{ pointerEvents: 'none', userSelect: 'none' }}
              >
                {label.slice(0, 5)}
              </text>
            </g>
          );
        })}
      </svg>
      <p className="text-xs text-muted-foreground mt-2 px-1">
        Click a province to view its results
      </p>
    </div>
  );
}
