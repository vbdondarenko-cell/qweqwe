import { type Slot } from '@/types';
import { ACTIVITY_MAP } from '@/data';
import { Avatar } from '@/components/ui/Avatar';
import { StatusBadge } from '@/components/ui/StatusBadge';
import { ProgressBar } from '@/components/ui/ProgressBar';
import { MapPin, Clock } from 'lucide-react';

interface SlotCardProps {
  slot: Slot;
  onJoin?: (slot: Slot) => void;
  onClick?: (slot: Slot) => void;
}

export function SlotCard({ slot, onJoin, onClick }: SlotCardProps) {
  const activity = ACTIVITY_MAP[slot.activity];

  return (
    <div
      onClick={() => onClick?.(slot)}
      className="bg-elevated border border-border rounded-2xl p-4 space-y-3 transition-all duration-200 hover:border-border/60 active:scale-[0.98] cursor-pointer animate-fade-slide-up"
    >
      <div className="flex items-start gap-3">
        <div className="w-12 h-12 rounded-xl bg-zone border border-border flex items-center justify-center text-2xl shrink-0">
          {activity.emoji}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <StatusBadge status={slot.status} />
            <span className="flex items-center gap-1 text-[11px] font-mono text-text-muted">
              <Clock size={11} />
              {slot.timeLabel}
            </span>
          </div>
          <h3 className="font-display font-bold text-base text-text-primary leading-tight truncate">
            {slot.title}
          </h3>
          <p className="flex items-center gap-1 text-xs text-text-dimmed mt-0.5">
            <MapPin size={12} className="shrink-0" />
            <span className="truncate">{slot.location}</span>
            <span className="text-text-muted">· {slot.distanceKm}km</span>
          </p>
        </div>
      </div>

      <p className="text-sm text-text-dimmed leading-relaxed line-clamp-2">{slot.description}</p>

      <div className="flex items-center gap-2">
        <Avatar initials={slot.organizer.initials} color={slot.organizer.avatarColor} size="sm" />
        <span className="text-xs font-body text-text-dimmed">{slot.organizer.name}</span>
        <span className="text-[10px] font-mono text-text-muted ml-auto">{slot.organizer.reliability}% reliable</span>
      </div>

      <ProgressBar value={slot.joined} max={slot.capacity} showLabel color={slot.joined >= slot.capacity ? 'red' : 'red'} />

      <div className="flex items-center justify-between gap-2">
        <div className="flex gap-1.5 flex-wrap">
          {slot.tags.map((tag) => (
            <span
              key={tag}
              className="px-2 py-0.5 rounded-md bg-zone border border-border text-[10px] font-body text-text-muted"
            >
              {tag}
            </span>
          ))}
        </div>
        <button
          onClick={(e) => {
            e.stopPropagation();
            onJoin?.(slot);
          }}
          className={`shrink-0 px-3.5 py-2 rounded-xl text-xs font-body font-bold transition-all active:scale-95 no-select ${
            slot.accessLevel === 'approval'
              ? 'bg-warning/15 text-warning border border-warning/30'
              : 'bg-red text-text-primary'
          }`}
        >
          {slot.accessLevel === 'approval' ? 'Request to join' : 'Join now'}
        </button>
      </div>
    </div>
  );
}
