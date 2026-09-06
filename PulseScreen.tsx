import { useState } from 'react';
import { Search, Bell, Coffee, Dumbbell, UtensilsCrossed, Layers, Zap } from 'lucide-react';
import type { Slot, TimeFilter, CategoryFilter, LoadState } from '@/types';
import { SLOTS, ACTIVITY_MAP } from '@/data';
import { Chip } from '@/components/ui/Chip';
import { SlotCard } from '@/components/SlotCard';
import { SlotCardSkeleton } from '@/components/ui/Skeleton';
import { EmptyState, ErrorState } from '@/components/StateToggle';

interface PulseScreenProps {
  loadState: LoadState;
  onSlotClick: (slot: Slot) => void;
  onJoin: (slot: Slot) => void;
  onOpenNotifications: () => void;
  unreadCount: number;
}

const timeFilters: { key: TimeFilter; label: string }[] = [
  { key: 'now', label: 'Now' },
  { key: 'tonight', label: 'Tonight' },
  { key: 'tomorrow', label: 'Tomorrow' },
  { key: 'all', label: 'All' },
];

const categoryFilters: { key: CategoryFilter; label: string; icon: typeof Coffee }[] = [
  { key: 'all', label: 'All', icon: Layers },
  { key: 'social', label: 'Social', icon: Coffee },
  { key: 'active', label: 'Active', icon: Dumbbell },
  { key: 'food', label: 'Food', icon: UtensilsCrossed },
];

export function PulseScreen({ loadState, onSlotClick, onJoin, onOpenNotifications, unreadCount }: PulseScreenProps) {
  const [timeFilter, setTimeFilter] = useState<TimeFilter>('now');
  const [categoryFilter, setCategoryFilter] = useState<CategoryFilter>('all');
  const [search, setSearch] = useState('');

  const happeningNow = SLOTS.filter((s) => s.happeningNow);

  const filtered = SLOTS.filter((s) => {
    if (categoryFilter !== 'all' && ACTIVITY_MAP[s.activity].category !== categoryFilter) return false;
    if (search) {
      const q = search.toLowerCase();
      return (
        s.title.toLowerCase().includes(q) ||
        s.location.toLowerCase().includes(q) ||
        s.tags.some((t) => t.includes(q))
      );
    }
    return true;
  });

  return (
    <div className="pb-28">
      {/* Header */}
      <div className="px-5 pt-14 pb-3 sticky top-0 z-30 glass border-b border-border">
        <div className="flex items-center justify-between mb-3">
          <div>
            <div className="flex items-center gap-2">
              <MapPinIcon />
              <span className="font-body font-semibold text-sm text-text-primary">Kyiv · Podil</span>
            </div>
            <div className="flex items-center gap-2 mt-1">
              <div className="relative">
                <div className="w-2 h-2 rounded-full bg-red animate-bpm-beat" />
                <div className="absolute inset-0 w-2 h-2 rounded-full bg-red/40 animate-pulse-ring" />
              </div>
              <span className="font-mono text-xs text-text-dimmed">
                City BPM <span className="text-red font-semibold">87</span>
              </span>
              <span className="text-[10px] text-text-muted">· High activity</span>
            </div>
          </div>
          <button
            onClick={onOpenNotifications}
            className="relative w-10 h-10 rounded-xl bg-elevated border border-border flex items-center justify-center active:scale-90 transition-transform"
          >
            <Bell size={18} className="text-text-dimmed" />
            {unreadCount > 0 && (
              <span className="absolute -top-1 -right-1 w-5 h-5 rounded-full bg-red text-text-primary text-[10px] font-mono font-bold flex items-center justify-center">
                {unreadCount}
              </span>
            )}
          </button>
        </div>

        {/* Search */}
        <div className="relative">
          <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search activities, places..."
            className="w-full bg-elevated border border-border rounded-xl pl-10 pr-4 py-2.5 text-sm font-body text-text-primary placeholder:text-text-muted focus:outline-none focus:border-red/50 transition-colors"
          />
        </div>
      </div>

      {/* Filters */}
      <div className="px-5 pt-3 pb-2 space-y-2.5">
        <div className="flex gap-2 overflow-x-auto">
          {timeFilters.map((f) => (
            <Chip key={f.key} active={timeFilter === f.key} onClick={() => setTimeFilter(f.key)}>
              {f.label}
            </Chip>
          ))}
        </div>
        <div className="flex gap-2 overflow-x-auto">
          {categoryFilters.map((f) => {
            const Icon = f.icon;
            return (
              <Chip
                key={f.key}
                active={categoryFilter === f.key}
                onClick={() => setCategoryFilter(f.key)}
                icon={<Icon size={14} />}
              >
                {f.label}
              </Chip>
            );
          })}
        </div>
      </div>

      {/* Content */}
      <div className="px-5 pt-2 space-y-3">
        {loadState === 'loading' && (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <SlotCardSkeleton key={i} />
            ))}
          </div>
        )}

        {loadState === 'error' && (
          <ErrorState message="Couldn't load nearby activities. Check your connection." onRetry={() => {}} />
        )}

        {loadState === 'empty' && (
          <EmptyState
            title="Quiet around here"
            subtitle="No activities nearby right now. Be the first to start something — tap LINK to create."
            icon={<Zap size={32} className="text-text-muted" />}
          />
        )}

        {loadState === 'content' && (
          <>
            {/* Happening Now banner */}
            {happeningNow.length > 0 && timeFilter === 'now' && (
              <div
                onClick={() => onSlotClick(happeningNow[0])}
                className="bg-success/10 border border-success/25 rounded-2xl p-4 flex items-center gap-3 cursor-pointer active:scale-[0.98] transition-transform animate-fade-slide-up"
              >
                <div className="relative">
                  <div className="w-10 h-10 rounded-xl bg-success/20 flex items-center justify-center">
                    <Zap size={18} className="text-success" />
                  </div>
                  <div className="absolute -top-0.5 -right-0.5 w-3 h-3 rounded-full bg-success border-2 border-background">
                    <div className="w-full h-full rounded-full bg-success animate-bpm-beat" />
                  </div>
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-[10px] font-mono font-semibold text-success uppercase tracking-wide">Happening Now</div>
                  <div className="font-display font-bold text-sm text-text-primary truncate">{happeningNow[0].title}</div>
                  <div className="text-xs text-text-dimmed truncate">{happeningNow[0].location} · {happeningNow[0].joined}/{happeningNow[0].capacity} going</div>
                </div>
              </div>
            )}

            {filtered.map((slot) => (
              <SlotCard key={slot.id} slot={slot} onJoin={onJoin} onClick={onSlotClick} />
            ))}

            {filtered.length === 0 && (
              <EmptyState
                title="No matches"
                subtitle="Try a different filter or search term."
                icon={<Search size={28} className="text-text-muted" />}
              />
            )}

            <div className="text-center py-4">
              <span className="text-[10px] font-mono text-text-muted">DEMO DATA · LinkUp Prototype v1</span>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function MapPinIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#FF2D35" strokeWidth="2">
      <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0118 0z" />
      <circle cx="12" cy="10" r="3" />
    </svg>
  );
}
