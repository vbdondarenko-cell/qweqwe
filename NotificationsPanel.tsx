import { Bell, UserCheck, Clock, Heart, Crown, Info } from 'lucide-react';
import type { NotificationItem } from '@/types';
import { Sheet } from '@/components/ui/Sheet';

interface NotificationsPanelProps {
  open: boolean;
  onClose: () => void;
  notifications: NotificationItem[];
  onMarkAllRead: () => void;
}

const typeConfig: Record<NotificationItem['type'], { icon: typeof Bell; color: string }> = {
  join: { icon: UserCheck, color: '#22C55E' },
  approval: { icon: Bell, color: '#F59E0B' },
  reminder: { icon: Clock, color: '#3B82F6' },
  bump: { icon: Heart, color: '#FF2D35' },
  host: { icon: Crown, color: '#FF2D35' },
  system: { icon: Info, color: '#A5A9B0' },
};

export function NotificationsPanel({ open, onClose, notifications, onMarkAllRead }: NotificationsPanelProps) {
  const unreadCount = notifications.filter((n) => !n.read).length;

  return (
    <Sheet
      open={open}
      onClose={onClose}
      title="Notifications"
      action={
        unreadCount > 0 ? (
          <button
            onClick={onMarkAllRead}
            className="text-xs font-body font-semibold text-red active:scale-95 transition-transform no-select"
          >
            Mark all read
          </button>
        ) : undefined
      }
    >
      <div className="space-y-1.5 pt-2">
        {notifications.length === 0 && (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <div className="w-16 h-16 rounded-2xl bg-zone border border-border flex items-center justify-center mb-3">
              <Bell size={24} className="text-text-muted" />
            </div>
            <h3 className="font-display font-bold text-base text-text-primary mb-1">All caught up</h3>
            <p className="text-sm text-text-dimmed max-w-[220px]">No new notifications right now.</p>
          </div>
        )}

        {notifications.map((n) => {
          const config = typeConfig[n.type];
          const Icon = config.icon;

          return (
            <div
              key={n.id}
              className={`flex items-start gap-3 p-3 rounded-2xl transition-colors ${
                n.read ? 'bg-transparent' : 'bg-red/5 border border-red/15'
              }`}
            >
              <div
                className="w-10 h-10 rounded-full flex items-center justify-center shrink-0"
                style={{ backgroundColor: `${config.color}15`, border: `1px solid ${config.color}30` }}
              >
                <Icon size={16} style={{ color: config.color }} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-body font-semibold text-text-primary truncate">{n.title}</span>
                  {!n.read && <span className="w-2 h-2 rounded-full bg-red shrink-0" />}
                </div>
                <p className="text-xs text-text-dimmed mt-0.5">{n.description}</p>
                <span className="text-[10px] font-mono text-text-muted mt-1 block">{n.time}</span>
              </div>
            </div>
          );
        })}

        {/* Footer */}
        <div className="pt-4 pb-2 text-center">
          <p className="text-[10px] font-mono text-text-muted">No engagement bait. Only events that matter.</p>
        </div>
      </div>
    </Sheet>
  );
}
