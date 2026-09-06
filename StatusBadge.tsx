import type { SlotStatus } from '@/types';

interface StatusBadgeProps {
  status: SlotStatus;
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const styles: Record<SlotStatus, string> = {
    LIVE: 'bg-success/15 text-success border-success/30',
    OPEN: 'bg-info/15 text-info border-info/30',
    FULL: 'bg-text-muted/15 text-text-muted border-text-muted/30',
    APPROVAL: 'bg-warning/15 text-warning border-warning/30',
  };

  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[10px] font-mono font-semibold border ${styles[status]}`}
    >
      {status === 'LIVE' && <span className="w-1.5 h-1.5 rounded-full bg-success animate-bpm-beat" />}
      {status}
    </span>
  );
}
