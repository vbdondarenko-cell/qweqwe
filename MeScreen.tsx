import { useState } from 'react';
import {
  Shield, Ghost, Bell, Globe, Accessibility, Database,
  Crown, Gift, Palette, FileText, LogOut, ChevronRight,
  Award, TrendingUp, MapPin, Heart, CheckCircle2, XCircle,
} from 'lucide-react';
import type { MeTab, LoadState } from '@/types';
import { USER_PROFILE, CITY_BADGES, BUMP_RECORDS, MONTH_ACTIVITY } from '@/data';
import { Avatar } from '@/components/ui/Avatar';
import { ProgressBar } from '@/components/ui/ProgressBar';
import { Chip } from '@/components/ui/Chip';
import { EmptyState, ErrorState } from '@/components/StateToggle';
import { SlotCardSkeleton } from '@/components/ui/Skeleton';

interface MeScreenProps {
  loadState: LoadState;
}

const meTabs: { key: MeTab; label: string }[] = [
  { key: 'profile', label: 'Profile' },
  { key: 'passport', label: 'Passport' },
  { key: 'settings', label: 'Settings' },
];

export function MeScreen({ loadState }: MeScreenProps) {
  const [tab, setTab] = useState<MeTab>('profile');

  return (
    <div className="pb-28">
      {/* Header */}
      <div className="px-5 pt-14 pb-3 sticky top-0 z-30 glass border-b border-border">
        <h1 className="font-display font-black text-2xl text-text-primary mb-3">Me</h1>
        <div className="flex gap-2">
          {meTabs.map((t) => (
            <Chip key={t.key} active={tab === t.key} onClick={() => setTab(t.key)}>
              {t.label}
            </Chip>
          ))}
        </div>
      </div>

      <div className="px-5 pt-4">
        {loadState === 'loading' && (
          <div className="space-y-3">
            <SlotCardSkeleton />
            <SlotCardSkeleton />
            <SlotCardSkeleton />
          </div>
        )}

        {loadState === 'error' && <ErrorState message="Couldn't load your profile." onRetry={() => {}} />}

        {loadState === 'empty' && <EmptyState title="Not signed in" subtitle="Sign in to see your profile." />}

        {loadState === 'content' && tab === 'profile' && <ProfileTab />}
        {loadState === 'content' && tab === 'passport' && <PassportTab />}
        {loadState === 'content' && tab === 'settings' && <SettingsTab />}
      </div>
    </div>
  );
}

function ProfileTab() {
  const u = USER_PROFILE;

  return (
    <div className="space-y-4 animate-fade-slide-up">
      {/* Profile header */}
      <div className="flex items-center gap-4">
        <Avatar initials={u.initials} color={u.avatarColor} size="xl" />
        <div>
          <h2 className="font-display font-bold text-xl text-text-primary">{u.name}</h2>
          <p className="text-sm text-text-dimmed">{u.username}</p>
          <div className="flex items-center gap-1 mt-1">
            <MapPin size={12} className="text-red" />
            <span className="text-xs text-text-dimmed">{u.city} · {u.area}</span>
          </div>
        </div>
      </div>

      {/* City badges */}
      <div className="flex gap-2 flex-wrap">
        {CITY_BADGES.slice(0, 3).map((c) => (
          <span
            key={c.city}
            className="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-elevated border border-border text-xs font-body text-text-dimmed"
          >
            {c.flag} {c.city}
          </span>
        ))}
      </div>

      {/* Reliability */}
      <div className="bg-elevated border border-border rounded-2xl p-4 space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <TrendingUp size={16} className="text-red" />
            <span className="font-body font-semibold text-sm text-text-primary">Reliability</span>
          </div>
          <span className="font-mono font-bold text-lg text-text-primary tabular-nums">{u.reliability}%</span>
        </div>
        <ProgressBar value={u.reliability} max={100} color="gradient" />
        <div className="grid grid-cols-2 gap-2 pt-1">
          <Metric icon={<CheckCircle2 size={14} className="text-success" />} label="Showed up" value={u.metrics.showedUp} />
          <Metric icon={<Award size={14} className="text-red" />} label="Hosted" value={u.metrics.hosted} />
          <Metric icon={<Heart size={14} className="text-info" />} label="BUMP verified" value={u.metrics.bumpVerified} />
          <Metric icon={<XCircle size={14} className="text-text-muted" />} label="No-show" value={u.metrics.noShow} />
        </div>
      </div>

      {/* BUMP Vault */}
      <div className="bg-elevated border border-border rounded-2xl p-4 space-y-3">
        <div className="flex items-center gap-2">
          <Heart size={16} className="text-red" />
          <span className="font-body font-semibold text-sm text-text-primary">BUMP Vault</span>
          <span className="ml-auto text-[10px] font-mono text-text-muted">{BUMP_RECORDS.length} verified</span>
        </div>
        <div className="space-y-2">
          {BUMP_RECORDS.map((b) => (
            <div key={b.id} className="flex items-center gap-3 py-1.5">
              <div className="w-8 h-8 rounded-lg bg-red/10 border border-red/20 flex items-center justify-center">
                <Heart size={14} className="text-red" />
              </div>
              <div className="flex-1">
                <div className="text-sm font-body font-semibold text-text-primary">{b.name}</div>
                <div className="text-xs text-text-muted">{b.event}</div>
              </div>
              <div className="text-right">
                <div className="text-[10px] font-mono text-text-muted">{b.date}</div>
                {b.verified && <CheckCircle2 size={12} className="text-success ml-auto mt-0.5" />}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* My LinkUps */}
      <div className="bg-elevated border border-border rounded-2xl p-4 space-y-2">
        <div className="flex items-center gap-2 mb-1">
          <Award size={16} className="text-red" />
          <span className="font-body font-semibold text-sm text-text-primary">My LinkUps</span>
        </div>
        <div className="flex items-center justify-between py-2">
          <span className="text-sm text-text-dimmed">Morning Espresso Run</span>
          <span className="text-[10px] font-mono text-success">LIVE · 4/6</span>
        </div>
        <div className="flex items-center justify-between py-2">
          <span className="text-sm text-text-dimmed">Cocktail Crawl · 3 Bars</span>
          <span className="text-[10px] font-mono text-info">OPEN · 6/8</span>
        </div>
        <div className="flex items-center justify-between py-2">
          <span className="text-sm text-text-dimmed">Evening Riverside Run</span>
          <span className="text-[10px] font-mono text-info">OPEN · 7/12</span>
        </div>
      </div>
    </div>
  );
}

function PassportTab() {
  const u = USER_PROFILE;

  const maxActivity = Math.max(...MONTH_ACTIVITY.map((m) => m.value));

  return (
    <div className="space-y-4 animate-fade-slide-up">
      {/* Metrics */}
      <div className="grid grid-cols-3 gap-2">
        <MetricCard value={u.passport.meetups} label="Meetups" />
        <MetricCard value={u.passport.cities} label="Cities" />
        <MetricCard value={u.passport.hosted} label="Hosted" />
      </div>

      {/* Cities explored */}
      <div className="bg-elevated border border-border rounded-2xl p-4 space-y-3">
        <div className="flex items-center gap-2">
          <Globe size={16} className="text-red" />
          <span className="font-body font-semibold text-sm text-text-primary">Cities Explored</span>
        </div>
        <div className="space-y-2">
          {CITY_BADGES.map((c) => (
            <div key={c.city} className="flex items-center gap-3 py-1.5">
              <span className="text-xl">{c.flag}</span>
              <div className="flex-1">
                <div className="text-sm font-body font-semibold text-text-primary">{c.city}</div>
                <div className="text-xs text-text-muted">{c.country}</div>
              </div>
              <span className="font-mono text-xs text-text-dimmed">{c.visits} visits</span>
            </div>
          ))}
        </div>
      </div>

      {/* Interests */}
      <div className="bg-elevated border border-border rounded-2xl p-4 space-y-3">
        <span className="font-body font-semibold text-sm text-text-primary">Interests</span>
        <div className="flex gap-2 flex-wrap">
          {u.interests.map((interest) => (
            <Chip key={interest} active>
              {interest}
            </Chip>
          ))}
        </div>
      </div>

      {/* Activity chart */}
      <div className="bg-elevated border border-border rounded-2xl p-4 space-y-3">
        <span className="font-body font-semibold text-sm text-text-primary">Activity · 6 months</span>
        <div className="flex items-end justify-between gap-2 h-32 pt-2">
          {MONTH_ACTIVITY.map((m) => (
            <div key={m.month} className="flex-1 flex flex-col items-center gap-1.5">
              <div className="w-full flex items-end justify-center" style={{ height: '100%' }}>
                <div
                  className="w-full max-w-[28px] rounded-t-lg bg-gradient-to-t from-red-deep to-red transition-all duration-500"
                  style={{ height: `${(m.value / maxActivity) * 100}%` }}
                />
              </div>
              <span className="text-[10px] font-mono text-text-muted">{m.month}</span>
              <span className="text-[10px] font-mono font-bold text-text-dimmed">{m.value}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function SettingsTab() {
  return (
    <div className="space-y-4 animate-fade-slide-up">
      {/* Privacy & Safety */}
      <SettingsSection title="Privacy & Safety">
        <SettingsItem icon={<Shield size={18} />} label="Privacy Center" />
        <SettingsItem icon={<Shield size={18} />} label="Safety Center" />
        <SettingsItem icon={<Heart size={18} />} label="Guardian" badge="NEW" />
        <SettingsItem icon={<Ghost size={18} />} label="Ghost Mode" toggle />
      </SettingsSection>

      {/* Account */}
      <SettingsSection title="Account">
        <SettingsItem icon={<Bell size={18} />} label="Notifications" />
        <SettingsItem icon={<Globe size={18} />} label="Language" value="English" />
        <SettingsItem icon={<Accessibility size={18} />} label="Accessibility" />
        <SettingsItem icon={<Database size={18} />} label="Data & Privacy" />
      </SettingsSection>

      {/* LinkUp+ */}
      <SettingsSection title="LinkUp+">
        <SettingsItem icon={<Crown size={18} />} label="Upgrade to LinkUp+" highlight />
        <SettingsItem icon={<Gift size={18} />} label="Rewarded Free Day" badge="FREE" />
      </SettingsSection>

      {/* App */}
      <SettingsSection title="App">
        <SettingsItem icon={<Palette size={18} />} label="Themes" value="OLED Dark" />
        <SettingsItem icon={<FileText size={18} />} label="Legal" />
        <SettingsItem icon={<FileText size={18} />} label="Version" value="1.0.0-prototype" />
        <SettingsItem icon={<LogOut size={18} />} label="Log out" danger />
      </SettingsSection>

      <div className="text-center py-4">
        <span className="text-[10px] font-mono text-text-muted">DEMO DATA · LinkUp Prototype v1</span>
      </div>
    </div>
  );
}

function Metric({ icon, label, value }: { icon: React.ReactNode; label: string; value: number }) {
  return (
    <div className="flex items-center gap-2 bg-zone/50 rounded-xl px-3 py-2">
      {icon}
      <div>
        <div className="font-mono font-bold text-sm text-text-primary tabular-nums">{value}</div>
        <div className="text-[10px] text-text-muted">{label}</div>
      </div>
    </div>
  );
}

function MetricCard({ value, label }: { value: number; label: string }) {
  return (
    <div className="bg-elevated border border-border rounded-2xl p-4 text-center">
      <div className="font-display font-black text-2xl text-text-primary">{value}</div>
      <div className="text-xs text-text-muted mt-0.5">{label}</div>
    </div>
  );
}

function SettingsSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="text-[10px] font-mono font-semibold text-text-muted uppercase tracking-wide px-1 mb-1.5">{title}</div>
      <div className="bg-elevated border border-border rounded-2xl overflow-hidden">{children}</div>
    </div>
  );
}

function SettingsItem({
  icon,
  label,
  value,
  badge,
  toggle,
  highlight,
  danger,
}: {
  icon: React.ReactNode;
  label: string;
  value?: string;
  badge?: string;
  toggle?: boolean;
  highlight?: boolean;
  danger?: boolean;
}) {
  return (
    <button
      className={`w-full flex items-center gap-3 px-4 py-3 border-b border-border/50 last:border-0 active:bg-zone transition-colors no-select ${
        danger ? 'text-critical' : highlight ? 'text-red' : 'text-text-dimmed'
      }`}
    >
      <span className={danger ? 'text-critical' : highlight ? 'text-red' : 'text-text-muted'}>{icon}</span>
      <span className={`flex-1 text-left text-sm font-body ${danger ? 'text-critical' : highlight ? 'text-red' : 'text-text-primary'}`}>
        {label}
      </span>
      {badge && (
        <span className="px-1.5 py-0.5 rounded-md bg-red/15 text-red text-[9px] font-mono font-bold border border-red/30">
          {badge}
        </span>
      )}
      {value && <span className="text-xs text-text-muted">{value}</span>}
      {toggle && (
        <div className="w-9 h-5 rounded-full bg-red flex items-center justify-end pr-0.5">
          <div className="w-4 h-4 rounded-full bg-text-primary" />
        </div>
      )}
      {!toggle && !value && !badge && <ChevronRight size={16} className="text-text-muted" />}
    </button>
  );
}
