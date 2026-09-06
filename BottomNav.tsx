import { Zap, MapPin, Rocket, User } from 'lucide-react';
import type { Screen } from '@/types';

interface BottomNavProps {
  active: Screen;
  onNavigate: (screen: Screen) => void;
}

const tabs: { screen: Screen; label: string; icon: typeof Zap }[] = [
  { screen: 'pulse', label: 'Pulse', icon: Zap },
  { screen: 'map', label: 'Map', icon: MapPin },
  { screen: 'fly', label: 'Fly', icon: Rocket },
  { screen: 'me', label: 'Me', icon: User },
];

export function BottomNav({ active, onNavigate }: BottomNavProps) {
  const left = tabs.slice(0, 2);
  const right = tabs.slice(2);

  return (
    <div className="absolute bottom-0 left-0 right-0 z-40">
      <div className="glass border-t border-border px-4 pt-2 pb-5 safe-bottom">
        <div className="flex items-end justify-between max-w-[420px] mx-auto">
          {left.map((tab) => (
            <NavTab key={tab.screen} tab={tab} active={active === tab.screen} onNavigate={onNavigate} />
          ))}

          <LinkButton onClick={() => onNavigate('create')} />

          {right.map((tab) => (
            <NavTab key={tab.screen} tab={tab} active={active === tab.screen} onNavigate={onNavigate} />
          ))}
        </div>
      </div>
    </div>
  );
}

function NavTab({
  tab,
  active,
  onNavigate,
}: {
  tab: { screen: Screen; label: string; icon: typeof Zap };
  active: boolean;
  onNavigate: (screen: Screen) => void;
}) {
  const Icon = tab.icon;
  return (
    <button
      onClick={() => onNavigate(tab.screen)}
      className="flex flex-col items-center gap-1 px-3 py-1.5 transition-all duration-200 active:scale-90 no-select"
    >
      <div className={`p-1.5 rounded-xl transition-all duration-300 ${active ? 'bg-red/15' : ''}`}>
        <Icon
          size={22}
          className={`transition-all duration-300 ${active ? 'text-red' : 'text-text-muted'}`}
          strokeWidth={active ? 2.5 : 2}
        />
      </div>
      <span
        className={`text-[10px] font-body font-semibold transition-all duration-300 ${
          active ? 'text-red' : 'text-text-muted'
        }`}
      >
        {tab.label}
      </span>
      {active && <div className="w-1 h-1 rounded-full bg-red animate-scale-in" />}
    </button>
  );
}

function LinkButton({ onClick }: { onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="relative flex flex-col items-center -mt-6 no-select"
    >
      <div className="relative">
        <div className="absolute inset-0 rounded-full bg-red/30 animate-pulse-ring" />
        <div className="relative w-14 h-14 rounded-full bg-red flex items-center justify-center shadow-lg shadow-red/40 animate-glow-pulse active:scale-90 transition-transform duration-200">
          <span className="font-display font-black text-white text-sm tracking-tight">LINK</span>
        </div>
      </div>
      <span className="text-[10px] font-body font-semibold text-text-muted mt-1">Create</span>
    </button>
  );
}
