import { useState, useEffect } from 'react';
import { Rocket, MapPin, Check, X } from 'lucide-react';
import type { FlyTab, LoadState } from '@/types';
import { FLASH_DROPS } from '@/data';
import { Chip } from '@/components/ui/Chip';
import { ProgressBar } from '@/components/ui/ProgressBar';
import { EmptyState, ErrorState } from '@/components/StateToggle';
import { SlotCardSkeleton } from '@/components/ui/Skeleton';

interface FlyScreenProps {
  loadState: LoadState;
}

const flyTabs: { key: FlyTab; label: string }[] = [
  { key: 'now', label: 'Now' },
  { key: 'travel', label: 'Travel' },
  { key: 'motion', label: 'Motion' },
];

export function FlyScreen({ loadState }: FlyScreenProps) {
  const [tab, setTab] = useState<FlyTab>('now');
  const [drops, setDrops] = useState(FLASH_DROPS);
  const [passed, setPassed] = useState<string[]>([]);
  const [joined, setJoined] = useState<string[]>([]);

  useEffect(() => {
    const interval = setInterval(() => {
      setDrops((prev) =>
        prev.map((d) => (d.secondsLeft > 0 ? { ...d, secondsLeft: d.secondsLeft - 1 } : d))
      );
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  const formatTime = (seconds: number): string => {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  };

  const availableDrops = drops.filter((d) => !passed.includes(d.id) && !joined.includes(d.id));
  const joinedCount = joined.length;
  const passedCount = passed.length;
  const availableCount = availableDrops.length;

  const handlePass = (id: string) => setPassed((p) => [...p, id]);
  const handleJoin = (id: string) => setJoined((j) => [...j, id]);

  return (
    <div className="pb-28">
      {/* Header */}
      <div className="px-5 pt-14 pb-3 sticky top-0 z-30 glass border-b border-border">
        <div className="flex items-center gap-2 mb-3">
          <Rocket size={20} className="text-red" />
          <h1 className="font-display font-black text-2xl text-text-primary">Fly Now</h1>
          <span className="ml-auto px-2.5 py-1 rounded-lg bg-red/15 border border-red/30 text-[10px] font-mono font-bold text-red">
            {availableCount + joinedCount} LIVE
          </span>
        </div>
        <p className="text-sm text-text-dimmed">Flash drops · Kyiv · right now</p>

        {/* Counters */}
        <div className="flex gap-2 mt-3">
          <Counter label="Joined" value={joinedCount} color="text-success" />
          <Counter label="Passed" value={passedCount} color="text-text-muted" />
          <Counter label="Available" value={availableCount} color="text-red" />
        </div>
      </div>

      {/* Tabs */}
      <div className="px-5 pt-3 pb-2">
        <div className="flex gap-2">
          {flyTabs.map((t) => (
            <Chip key={t.key} active={tab === t.key} onClick={() => setTab(t.key)}>
              {t.label}
            </Chip>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="px-5 pt-2 space-y-3">
        {loadState === 'loading' && (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <SlotCardSkeleton key={i} />
            ))}
          </div>
        )}

        {loadState === 'error' && <ErrorState message="Couldn't load flash drops." onRetry={() => {}} />}

        {loadState === 'empty' && (
          <EmptyState
            title="No flash drops"
            subtitle="No live drops in your area right now. Check back soon."
            icon={<Rocket size={28} className="text-text-muted" />}
          />
        )}

        {loadState === 'content' && (
          <>
            {availableDrops.map((drop) => (
              <FlashCard
                key={drop.id}
                drop={drop}
                formatTime={formatTime}
                onPass={() => handlePass(drop.id)}
                onJoin={() => handleJoin(drop.id)}
              />
            ))}

            {joined.length > 0 && (
              <div className="pt-2">
                <div className="text-[10px] font-mono text-success font-semibold uppercase tracking-wide mb-2">Joined</div>
                {drops
                  .filter((d) => joined.includes(d.id))
                  .map((drop) => (
                    <div
                      key={drop.id}
                      className="bg-success/10 border border-success/25 rounded-2xl p-4 flex items-center gap-3 mb-2 animate-fade-slide-up"
                    >
                      <div className="w-10 h-10 rounded-xl bg-success/20 flex items-center justify-center text-xl">
                        {drop.emoji}
                      </div>
                      <div className="flex-1">
                        <div className="font-display font-bold text-sm text-text-primary">{drop.title}</div>
                        <div className="text-xs text-text-dimmed">{drop.location}</div>
                      </div>
                      <Check size={20} className="text-success" />
                    </div>
                  ))}
              </div>
            )}

            {availableDrops.length === 0 && joined.length === 0 && (
              <EmptyState
                title="All caught up"
                subtitle="You've seen all current flash drops. New ones appear throughout the day."
                icon={<Rocket size={28} className="text-text-muted" />}
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

function Counter({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="flex-1 bg-elevated border border-border rounded-xl px-3 py-2 text-center">
      <div className={`font-mono font-bold text-xl tabular-nums ${color}`}>{value}</div>
      <div className="text-[10px] font-body text-text-muted">{label}</div>
    </div>
  );
}

function FlashCard({
  drop,
  formatTime,
  onPass,
  onJoin,
}: {
  drop: (typeof FLASH_DROPS)[number];
  formatTime: (s: number) => string;
  onPass: () => void;
  onJoin: () => void;
}) {
  const isUrgent = drop.secondsLeft < 300;

  return (
    <div className="bg-elevated border border-border rounded-2xl overflow-hidden animate-fade-slide-up">
      <div className="p-4 space-y-3">
        <div className="flex items-start gap-3">
          <div className="w-12 h-12 rounded-xl bg-zone border border-border flex items-center justify-center text-2xl shrink-0">
            {drop.emoji}
          </div>
          <div className="flex-1 min-w-0">
            <h3 className="font-display font-bold text-base text-text-primary truncate">{drop.title}</h3>
            <p className="flex items-center gap-1 text-xs text-text-dimmed mt-0.5">
              <MapPin size={12} className="shrink-0" />
              <span className="truncate">{drop.location}</span>
              <span className="text-text-muted">· {drop.distanceKm}km</span>
            </p>
          </div>
          <div
            className={`px-2.5 py-1.5 rounded-xl border font-mono font-bold text-sm tabular-nums ${
              isUrgent
                ? 'bg-critical/15 text-critical border-critical/30 animate-bpm-beat'
                : 'bg-zone text-text-dimmed border-border'
            }`}
          >
            {formatTime(drop.secondsLeft)}
          </div>
        </div>

        <div className="flex gap-1.5 flex-wrap">
          {drop.tags.map((tag) => (
            <span key={tag} className="px-2 py-0.5 rounded-md bg-zone border border-border text-[10px] font-body text-text-muted">
              {tag}
            </span>
          ))}
        </div>

        <ProgressBar value={drop.joined} max={drop.capacity} showLabel color={drop.joined >= drop.capacity ? 'red' : 'red'} />

        <div className="flex gap-2">
          <button
            onClick={onPass}
            className="flex-1 py-2.5 rounded-xl bg-zone border border-border text-sm font-body font-semibold text-text-dimmed flex items-center justify-center gap-1.5 active:scale-95 transition-transform no-select"
          >
            <X size={16} /> Pass
          </button>
          <button
            onClick={onJoin}
            className="flex-1 py-2.5 rounded-xl bg-red text-sm font-body font-bold text-text-primary flex items-center justify-center gap-1.5 active:scale-95 transition-transform no-select"
          >
            <Rocket size={16} /> Join
          </button>
        </div>
      </div>
    </div>
  );
}
