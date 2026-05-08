'use client';

// Selection card for a single candidate, party, or referendum option.

interface Props {
  isSelected: boolean;
  onSelect: () => void;
  color?: string;   // hex, no # prefix
  label: string;
  sublabel?: string;
  badge?: string;
  disabled?: boolean;
}

export function BallotCard({
  isSelected,
  onSelect,
  color,
  label,
  sublabel,
  badge,
  disabled,
}: Props) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={isSelected}
      onClick={onSelect}
      disabled={disabled}
      className={[
        'relative w-full rounded-xl border-2 px-4 py-3 text-left transition-all',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
        isSelected
          ? 'border-primary bg-primary/5 shadow-sm'
          : 'border-border bg-card hover:border-primary/50 hover:bg-muted/30',
        disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
      ].join(' ')}
    >
      <div className="flex items-center gap-3">
        {/* Colour swatch / radio indicator */}
        <span
          className={[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-full border-2',
            isSelected ? 'border-primary' : 'border-muted-foreground/30',
          ].join(' ')}
          aria-hidden
        >
          {color && (
            <span
              className="h-5 w-5 rounded-full"
              style={{ background: `#${color}` }}
            />
          )}
          {!color && isSelected && (
            <span className="h-3 w-3 rounded-full bg-primary" />
          )}
        </span>

        <div className="flex flex-col min-w-0">
          <span className="text-sm font-semibold truncate">{label}</span>
          {sublabel && (
            <span className="text-xs text-muted-foreground truncate">{sublabel}</span>
          )}
        </div>

        {badge && (
          <span className="ml-auto shrink-0 rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
            {badge}
          </span>
        )}
      </div>

      {/* Selected checkmark */}
      {isSelected && (
        <span
          className="absolute right-3 top-3 flex h-5 w-5 items-center justify-center rounded-full bg-primary text-[10px] text-primary-foreground"
          aria-hidden
        >
          ✓
        </span>
      )}
    </button>
  );
}
