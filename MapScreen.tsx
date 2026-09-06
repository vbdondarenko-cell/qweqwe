import { useState } from 'react';
import { Filter, Flame, Settings, ChevronRight } from 'lucide-react';
import type { Slot, MapPin, LoadState } from '@/types';
import { MAP_PINS, SLOTS, ACTIVITY_MAP } from '@/data';
import { Sheet } from '@/components/ui/Sheet';
import { StatusBadge } from '@/components/ui/StatusBadge';
import { ProgressBar } from '@/components/ui/ProgressBar';
import { Avatar } from '@/components/ui/Avatar';
import { EmptyState, ErrorState } from '@/components/StateToggle';

interface MapScreenProps {
  loadState: LoadState;
  onSlotClick: (slot: Slot) => void;
}

export function MapScreen({ loadState, onSlotClick }: MapScreenProps) {
  const [selectedPin, setSelectedPin] = useState<MapPin | null>(null);

  const selectedSlot = selectedPin ? SLOTS.find((s) => s.id === selectedPin.slotId) : null;
  const activeCount = MAP_PINS.filter((p) => p.status === 'LIVE' || p.status === 'OPEN').length;

  const pinColor = (activity: string): string => {
    const colors: Record<string, string> = {
      coffee: '#F59E0B',
      running: '#22C55E',
      food: '#EF4444',
      drinks: '#FF2D35',
      photography: '#3B82F6',
      chess: '#A5A9B0',
      cycling: '#22C55E',
      yoga: '#3B82F6',
    };
    return colors[activity] ?? '#FF2D35';
  };

  return (
    <div className="relative h-screen pb-28">
      {/* Stylized dark map */}
      {loadState === 'content' && (
        <div className="absolute inset-0 overflow-hidden">
          <div className="absolute inset-0 bg-[#08090B]">
            {/* Grid lines */}
            <svg className="absolute inset-0 w-full h-full opacity-[0.07]" preserveAspectRatio="none">
              <defs>
                <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
                  <path d="M 40 0 L 0 0 0 40" fill="none" stroke="#F7F8FA" strokeWidth="0.5" />
                </pattern>
              </defs>
              <rect width="100%" height="100%" fill="url(#grid)" />
            </svg>

            {/* Stylized roads */}
            <svg className="absolute inset-0 w-full h-full" viewBox="0 0 100 100" preserveAspectRatio="none">
              <path d="M0,35 Q30,30 50,40 T100,38" stroke="#1A1C20" strokeWidth="2" fill="none" />
              <path d="M0,60 Q25,65 50,55 T100,62" stroke="#1A1C20" strokeWidth="2" fill="none" />
              <path d="M20,0 Q22,30 35,50 T28,100" stroke="#1A1C20" strokeWidth="2" fill="none" />
              <path d="M65,0 Q70,25 60,50 T72,100" stroke="#1A1C20" strokeWidth="2" fill="none" />
              <path d="M0,35 Q30,30 50,40 T100,38" stroke="#24262B" strokeWidth="0.5" fill="none" strokeDasharray="2,2" />
              {/* River */}
              <path d="M0,75 Q30,72 50,78 T100,74" stroke="#0F1A2E" strokeWidth="4" fill="none" opacity="0.5" />
              {/* Park area */}
              <ellipse cx="48" cy="68" rx="8" ry="5" fill="#0D1A14" opacity="0.4" />
            </svg>

            {/* Pins */}
            {MAP_PINS.map((pin) => {
              const color = pinColor(pin.activity);
              const activity = ACTIVITY_MAP[pin.activity];
              return (
                <button
                  key={pin.id}
                  onClick={() => setSelectedPin(pin)}
                  className="absolute -translate-x-1/2 -translate-y-1/2 active:scale-90 transition-transform no-select"
                  style={{ left: `${pin.x}%`, top: `${pin.y}%` }}
                >
                  <div
                    className={`relative flex flex-col items-center ${selectedPin?.id === pin.id ? 'scale-110' : ''} transition-transform duration-200`}
                  >
                    {pin.status === 'LIVE' && (
                      <div
                        className="absolute -inset-1 rounded-full animate-pulse-ring"
                        style={{ backgroundColor: `${color}40` }}
                      />
                    )}
                    <div
                      className="relative w-10 h-10 rounded-full flex items-center justify-center text-lg border-2"
                      style={{
                        backgroundColor: `${color}25`,
                        borderColor: color,
                        boxShadow: `0 0 12px ${color}40`,
                      }}
                    >
                      {activity.emoji}
                    </div>
                    <div
                      className="mt-0.5 px-1.5 py-0.5 rounded-md text-[9px] font-mono font-bold border"
                      style={{
                        backgroundColor: `${color}15`,
                        borderColor: `${color}40`,
                        color: color,
                      }}
                    >
                      {pin.joined}/{pin.capacity}
                    </div>
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      )}

      {/* Top area pill */}
      <div className="absolute top-14 left-1/2 -translate-x-1/2 z-20">
        <div className="glass border border-border rounded-full px-4 py-2 flex items-center gap-2">
          <div className="w-2 h-2 rounded-full bg-red animate-bpm-beat" />
          <span className="font-body font-semibold text-xs text-text-primary">Podil · Kyiv</span>
        </div>
      </div>

      {/* Right control column */}
      <div className="absolute right-4 top-24 z-20 flex flex-col gap-2">
        {[
          { icon: Filter, label: 'Filter' },
          { icon: Flame, label: 'Hot' },
          { icon: Settings, label: 'Settings' },
        ].map((btn) => {
          const Icon = btn.icon;
          return (
            <button
              key={btn.label}
              className="w-10 h-10 rounded-xl glass border border-border flex items-center justify-center active:scale-90 transition-transform"
            >
              <Icon size={18} className="text-text-dimmed" />
            </button>
          );
        })}
      </div>

      {/* Bottom panel */}
      {loadState === 'content' && (
        <div className="absolute bottom-24 left-0 right-0 z-20 px-5">
          <div className="glass border border-border rounded-2xl px-4 py-3 flex items-center justify-between">
            <div>
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-success animate-bpm-beat" />
                <span className="font-mono text-xs font-semibold text-success">ACTIVE</span>
              </div>
              <div className="font-display font-bold text-sm text-text-primary mt-0.5">
                {activeCount} active LinkUps
              </div>
            </div>
            <button className="flex items-center gap-1 text-xs font-body font-semibold text-text-dimmed active:scale-95 transition-transform">
              List view <ChevronRight size={14} />
            </button>
          </div>
        </div>
      )}

      {/* Load states */}
      {loadState === 'loading' && (
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="w-8 h-8 border-2 border-border border-t-red rounded-full animate-spin" />
        </div>
      )}
      {loadState === 'empty' && (
        <div className="absolute inset-0 flex items-center justify-center">
          <EmptyState title="No map data" subtitle="No activities to display on the map." />
        </div>
      )}
      {loadState === 'error' && (
        <div className="absolute inset-0 flex items-center justify-center">
          <ErrorState message="Couldn't load the map." onRetry={() => {}} />
        </div>
      )}

      {/* Pin detail sheet */}
      <Sheet open={!!selectedPin} onClose={() => setSelectedPin(null)}>
        {selectedSlot && (
          <div className="space-y-4 pt-2">
            <div className="flex items-start gap-3">
              <div className="w-14 h-14 rounded-2xl bg-zone border border-border flex items-center justify-center text-3xl">
                {ACTIVITY_MAP[selectedSlot.activity].emoji}
              </div>
              <div className="flex-1">
                <StatusBadge status={selectedSlot.status} />
                <h3 className="font-display font-bold text-lg text-text-primary mt-1">{selectedSlot.title}</h3>
                <p className="text-xs text-text-dimmed mt-0.5">{selectedSlot.location} · {selectedSlot.distanceKm}km</p>
              </div>
            </div>
            <p className="text-sm text-text-dimmed leading-relaxed">{selectedSlot.description}</p>
            <div className="flex items-center gap-2">
              <Avatar initials={selectedSlot.organizer.initials} color={selectedSlot.organizer.avatarColor} size="sm" />
              <span className="text-xs font-body text-text-dimmed">{selectedSlot.organizer.name}</span>
            </div>
            <ProgressBar value={selectedSlot.joined} max={selectedSlot.capacity} showLabel />
            <div className="flex gap-1.5 flex-wrap">
              {selectedSlot.tags.map((tag) => (
                <span key={tag} className="px-2 py-0.5 rounded-md bg-zone border border-border text-[10px] font-body text-text-muted">
                  {tag}
                </span>
              ))}
            </div>
            <button
              onClick={() => {
                onSlotClick(selectedSlot);
                setSelectedPin(null);
              }}
              className={`w-full py-3 rounded-2xl font-body font-bold text-sm transition-all active:scale-95 ${
                selectedSlot.accessLevel === 'approval'
                  ? 'bg-warning/15 text-warning border border-warning/30'
                  : 'bg-red text-text-primary'
              }`}
            >
              {selectedSlot.accessLevel === 'approval' ? 'Request to join' : 'Join now'}
            </button>
          </div>
        )}
      </Sheet>
    </div>
  );
}
