import { useState } from 'react';
import type { Screen, LoadState, Slot, NotificationItem } from '@/types';
import { NOTIFICATIONS } from '@/data';
import { BottomNav } from '@/components/BottomNav';
import { StateToggle } from '@/components/StateToggle';
import { NotificationsPanel } from '@/components/NotificationsPanel';
import { Sheet } from '@/components/ui/Sheet';
import { StatusBadge } from '@/components/ui/StatusBadge';
import { Avatar } from '@/components/ui/Avatar';
import { ProgressBar } from '@/components/ui/ProgressBar';
import { ACTIVITY_MAP } from '@/data';
import { PulseScreen } from '@/screens/PulseScreen';
import { MapScreen } from '@/screens/MapScreen';
import { CreateLinkScreen } from '@/screens/CreateLinkScreen';
import { FlyScreen } from '@/screens/FlyScreen';
import { MeScreen } from '@/screens/MeScreen';

function App() {
  const [screen, setScreen] = useState<Screen>('pulse');
  const [prevScreen, setPrevScreen] = useState<Screen>('pulse');
  const [loadState, setLoadState] = useState<LoadState>('content');
  const [showCreate, setShowCreate] = useState(false);
  const [showNotifications, setShowNotifications] = useState(false);
  const [notifications, setNotifications] = useState<NotificationItem[]>(NOTIFICATIONS);
  const [detailSlot, setDetailSlot] = useState<Slot | null>(null);

  const unreadCount = notifications.filter((n) => !n.read).length;

  const handleNavigate = (s: Screen) => {
    if (s === 'create') {
      setShowCreate(true);
      return;
    }
    setPrevScreen(screen);
    setScreen(s);
  };

  const handleMarkAllRead = () => {
    setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
  };

  const handleJoin = (slot: Slot) => {
    setDetailSlot(slot);
  };

  return (
    <div className="min-h-screen bg-background flex justify-center">
      <div className="mobile-frame">
        <StateToggle state={loadState} onChange={setLoadState} />

        {screen === 'pulse' && (
          <PulseScreen
            loadState={loadState}
            onSlotClick={setDetailSlot}
            onJoin={handleJoin}
            onOpenNotifications={() => setShowNotifications(true)}
            unreadCount={unreadCount}
          />
        )}

        {screen === 'map' && <MapScreen loadState={loadState} onSlotClick={setDetailSlot} />}

        {screen === 'fly' && <FlyScreen loadState={loadState} />}

        {screen === 'me' && <MeScreen loadState={loadState} />}

        <BottomNav active={screen} onNavigate={handleNavigate} />

        {/* Create LINK overlay */}
        {showCreate && (
          <CreateLinkScreen
            onClose={() => {
              setShowCreate(false);
              setScreen(prevScreen);
            }}
            onPublish={() => {
              setShowCreate(false);
              setScreen(prevScreen);
            }}
          />
        )}

        {/* Notifications panel */}
        <NotificationsPanel
          open={showNotifications}
          onClose={() => setShowNotifications(false)}
          notifications={notifications}
          onMarkAllRead={handleMarkAllRead}
        />

        {/* Slot detail sheet */}
        <Sheet open={!!detailSlot} onClose={() => setDetailSlot(null)}>
          {detailSlot && (
            <div className="space-y-4 pt-2">
              <div className="flex items-start gap-3">
                <div className="w-14 h-14 rounded-2xl bg-zone border border-border flex items-center justify-center text-3xl">
                  {ACTIVITY_MAP[detailSlot.activity].emoji}
                </div>
                <div className="flex-1">
                  <StatusBadge status={detailSlot.status} />
                  <h3 className="font-display font-bold text-lg text-text-primary mt-1">{detailSlot.title}</h3>
                  <p className="text-xs text-text-dimmed mt-0.5">{detailSlot.location} · {detailSlot.distanceKm}km</p>
                </div>
              </div>
              <p className="text-sm text-text-dimmed leading-relaxed">{detailSlot.description}</p>
              <div className="flex items-center gap-2">
                <Avatar initials={detailSlot.organizer.initials} color={detailSlot.organizer.avatarColor} size="sm" />
                <span className="text-xs font-body text-text-dimmed">{detailSlot.organizer.name}</span>
                <span className="text-[10px] font-mono text-text-muted ml-auto">{detailSlot.organizer.reliability}% reliable</span>
              </div>
              <ProgressBar value={detailSlot.joined} max={detailSlot.capacity} showLabel />
              <div className="flex gap-1.5 flex-wrap">
                {detailSlot.tags.map((tag) => (
                  <span key={tag} className="px-2 py-0.5 rounded-md bg-zone border border-border text-[10px] font-body text-text-muted">
                    {tag}
                  </span>
                ))}
              </div>
              <button
                onClick={() => setDetailSlot(null)}
                className={`w-full py-3 rounded-2xl font-body font-bold text-sm transition-all active:scale-95 ${
                  detailSlot.accessLevel === 'approval'
                    ? 'bg-warning/15 text-warning border border-warning/30'
                    : 'bg-red text-text-primary'
                }`}
              >
                {detailSlot.accessLevel === 'approval' ? 'Request to join' : 'Join now'}
              </button>
            </div>
          )}
        </Sheet>
      </div>
    </div>
  );
}

export default App;
