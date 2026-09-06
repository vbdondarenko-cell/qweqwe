interface ProgressBarProps {
  value: number;
  max: number;
  className?: string;
  color?: 'red' | 'green' | 'gradient';
  showLabel?: boolean;
}

export function ProgressBar({ value, max, className = '', color = 'red', showLabel }: ProgressBarProps) {
  const pct = Math.min(100, (value / max) * 100);

  const barColor =
    color === 'green'
      ? 'bg-success'
      : color === 'gradient'
        ? 'bg-gradient-to-r from-red via-red-signal to-success'
        : 'bg-red';

  return (
    <div className={`flex items-center gap-2 ${className}`}>
      <div className="flex-1 h-1.5 bg-zone rounded-full overflow-hidden">
        <div
          className={`h-full ${barColor} rounded-full transition-all duration-500 ease-out`}
          style={{ width: `${pct}%` }}
        />
      </div>
      {showLabel && (
        <span className="font-mono text-xs text-text-dimmed tabular-nums whitespace-nowrap">
          {value}/{max}
        </span>
      )}
    </div>
  );
}
