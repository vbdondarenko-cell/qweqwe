import { useState } from 'react';
import { X, Minus, Plus, Globe, Lock, Users, Check, ChevronRight } from 'lucide-react';
import { ACTIVITIES } from '@/data';
import type { ActivityType, AccessLevel, Visibility } from '@/types';
import { Button } from '@/components/ui/Button';

interface CreateLinkScreenProps {
  onClose: () => void;
  onPublish: () => void;
}

export function CreateLinkScreen({ onClose, onPublish }: CreateLinkScreenProps) {
  const [step, setStep] = useState(1);
  const [activity, setActivity] = useState<ActivityType | null>(null);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [location, setLocation] = useState('');
  const [capacity, setCapacity] = useState(6);
  const [accessLevel, setAccessLevel] = useState<AccessLevel>('instant');
  const [visibility, setVisibility] = useState<Visibility>('public');

  const canContinueStep1 = activity !== null;
  const canContinueStep2 = title.trim().length > 0 && location.trim().length > 0;

  const handlePublish = () => {
    onPublish();
  };

  return (
    <div className="fixed inset-0 z-50 bg-background animate-fade-in">
      <div className="mobile-frame flex flex-col h-full mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between px-5 pt-14 pb-3 border-b border-border">
          <button
            onClick={onClose}
            className="w-9 h-9 rounded-xl bg-elevated border border-border flex items-center justify-center active:scale-90 transition-transform"
          >
            <X size={18} className="text-text-dimmed" />
          </button>
          <div className="text-center">
            <div className="font-display font-bold text-sm text-text-primary">Create LINK</div>
            <div className="font-mono text-[10px] text-text-muted">Step {step} of 3</div>
          </div>
          <div className="w-9" />
        </div>

        {/* Progress bar */}
        <div className="px-5 py-2">
          <div className="flex gap-1.5">
            {[1, 2, 3].map((s) => (
              <div
                key={s}
                className={`flex-1 h-1 rounded-full transition-all duration-300 ${
                  s <= step ? 'bg-red' : 'bg-zone'
                }`}
              />
            ))}
          </div>
        </div>

        {/* Steps */}
        <div className="flex-1 overflow-y-auto px-5 py-4">
          {step === 1 && (
            <div className="space-y-4 animate-fade-slide-up">
              <div>
                <h2 className="font-display font-bold text-xl text-text-primary mb-1">What's happening?</h2>
                <p className="text-sm text-text-dimmed">Pick an activity to get started.</p>
              </div>

              <div className="grid grid-cols-4 gap-2">
                {ACTIVITIES.map((a) => (
                  <button
                    key={a.type}
                    onClick={() => setActivity(a.type)}
                    className={`aspect-square rounded-2xl border flex flex-col items-center justify-center gap-1 transition-all active:scale-90 no-select ${
                      activity === a.type
                        ? 'bg-red/15 border-red'
                        : 'bg-elevated border-border hover:border-border/60'
                    }`}
                  >
                    <span className="text-2xl">{a.emoji}</span>
                    <span className={`text-[9px] font-body font-semibold ${activity === a.type ? 'text-red' : 'text-text-dimmed'}`}>
                      {a.label}
                    </span>
                  </button>
                ))}
              </div>

              {activity && (
                <div className="space-y-3 animate-fade-slide-up">
                  <div>
                    <label className="text-xs font-body font-semibold text-text-dimmed mb-1.5 block">
                      Title <span className="text-text-muted">({title.length}/60)</span>
                    </label>
                    <input
                      value={title}
                      onChange={(e) => setTitle(e.target.value.slice(0, 60))}
                      placeholder="e.g. Morning Coffee at Green Hills"
                      className="w-full bg-elevated border border-border rounded-xl px-4 py-3 text-sm font-body text-text-primary placeholder:text-text-muted focus:outline-none focus:border-red/50 transition-colors"
                    />
                  </div>
                  <div>
                    <label className="text-xs font-body font-semibold text-text-dimmed mb-1.5 block">Description</label>
                    <textarea
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                      placeholder="Tell people what to expect..."
                      rows={3}
                      className="w-full bg-elevated border border-border rounded-xl px-4 py-3 text-sm font-body text-text-primary placeholder:text-text-muted focus:outline-none focus:border-red/50 transition-colors resize-none"
                    />
                  </div>
                </div>
              )}
            </div>
          )}

          {step === 2 && (
            <div className="space-y-5 animate-fade-slide-up">
              <div>
                <h2 className="font-display font-bold text-xl text-text-primary mb-1">Where & when?</h2>
                <p className="text-sm text-text-dimmed">Set the details for your LinkUp.</p>
              </div>

              <div>
                <label className="text-xs font-body font-semibold text-text-dimmed mb-1.5 block">Location</label>
                <input
                  value={location}
                  onChange={(e) => setLocation(e.target.value)}
                  placeholder="e.g. Green Hills Coffee, Podil"
                  className="w-full bg-elevated border border-border rounded-xl px-4 py-3 text-sm font-body text-text-primary placeholder:text-text-muted focus:outline-none focus:border-red/50 transition-colors"
                />
              </div>

              <div>
                <label className="text-xs font-body font-semibold text-text-dimmed mb-2 block">Capacity</label>
                <div className="flex items-center gap-4">
                  <button
                    onClick={() => setCapacity((c) => Math.max(2, c - 1))}
                    className="w-12 h-12 rounded-2xl bg-elevated border border-border flex items-center justify-center active:scale-90 transition-transform"
                  >
                    <Minus size={20} className="text-text-dimmed" />
                  </button>
                  <div className="flex-1 text-center">
                    <span className="font-mono font-bold text-3xl text-text-primary tabular-nums">{capacity}</span>
                    <span className="text-sm text-text-muted ml-1">people</span>
                  </div>
                  <button
                    onClick={() => setCapacity((c) => Math.min(50, c + 1))}
                    className="w-12 h-12 rounded-2xl bg-elevated border border-border flex items-center justify-center active:scale-90 transition-transform"
                  >
                    <Plus size={20} className="text-text-dimmed" />
                  </button>
                </div>
              </div>

              <div>
                <label className="text-xs font-body font-semibold text-text-dimmed mb-2 block">Access level</label>
                <div className="space-y-2">
                  <AccessOption
                    active={accessLevel === 'instant'}
                    onClick={() => setAccessLevel('instant')}
                    icon={<Users size={18} />}
                    title="Instant Join"
                    subtitle="Anyone can join immediately"
                  />
                  <AccessOption
                    active={accessLevel === 'approval'}
                    onClick={() => setAccessLevel('approval')}
                    icon={<Check size={18} />}
                    title="Approval Required"
                    subtitle="You approve each request manually"
                  />
                </div>
              </div>

              <div>
                <label className="text-xs font-body font-semibold text-text-dimmed mb-2 block">Visibility</label>
                <div className="space-y-2">
                  <AccessOption
                    active={visibility === 'public'}
                    onClick={() => setVisibility('public')}
                    icon={<Globe size={18} />}
                    title="Public"
                    subtitle="Visible to everyone nearby"
                  />
                  <AccessOption
                    active={visibility === 'friends'}
                    onClick={() => setVisibility('friends')}
                    icon={<Users size={18} />}
                    title="Friends Only"
                    subtitle="Only your connections can see it"
                  />
                  <AccessOption
                    active={visibility === 'private'}
                    onClick={() => setVisibility('private')}
                    icon={<Lock size={18} />}
                    title="Private"
                    subtitle="Invite-only, not listed anywhere"
                  />
                </div>
              </div>
            </div>
          )}

          {step === 3 && (
            <div className="space-y-4 animate-fade-slide-up">
              <div>
                <h2 className="font-display font-bold text-xl text-text-primary mb-1">Preview</h2>
                <p className="text-sm text-text-dimmed">Review before publishing.</p>
              </div>

              <div className="bg-elevated border border-border rounded-2xl p-4 space-y-3">
                <div className="flex items-start gap-3">
                  <div className="w-14 h-14 rounded-2xl bg-zone border border-border flex items-center justify-center text-3xl">
                    {activity ? ACTIVITIES.find((a) => a.type === activity)?.emoji : '🎯'}
                  </div>
                  <div className="flex-1">
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[10px] font-mono font-semibold border bg-info/15 text-info border-info/30">
                      OPEN
                    </span>
                    <h3 className="font-display font-bold text-base text-text-primary mt-1">
                      {title || 'Untitled LinkUp'}
                    </h3>
                    <p className="text-xs text-text-dimmed">{location || 'No location set'}</p>
                  </div>
                </div>
                {description && (
                  <p className="text-sm text-text-dimmed leading-relaxed">{description}</p>
                )}
                <div className="flex items-center gap-3 text-xs">
                  <span className="font-mono text-text-dimmed">0/{capacity} going</span>
                  <span className="text-text-muted">·</span>
                  <span className="text-text-dimmed">{accessLevel === 'instant' ? 'Instant join' : 'Approval required'}</span>
                  <span className="text-text-muted">·</span>
                  <span className="text-text-dimmed capitalize">{visibility}</span>
                </div>
              </div>

              <div className="bg-zone/50 border border-border rounded-xl p-3 flex items-start gap-2">
                <Check size={16} className="text-success shrink-0 mt-0.5" />
                <p className="text-xs text-text-dimmed">
                  This is a demo prototype. No real slot will be created — the data stays in your browser.
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-5 pb-6 pt-3 border-t border-border safe-bottom">
          <div className="flex gap-2">
            {step > 1 && (
              <Button variant="secondary" onClick={() => setStep((s) => s - 1)}>
                Back
              </Button>
            )}
            {step < 3 ? (
              <Button
                fullWidth
                disabled={(step === 1 && !canContinueStep1) || (step === 2 && !canContinueStep2)}
                onClick={() => setStep((s) => s + 1)}
                className={`flex items-center justify-center gap-1 ${
                  (step === 1 && !canContinueStep1) || (step === 2 && !canContinueStep2)
                    ? 'opacity-40 cursor-not-allowed bg-elevated text-text-muted border border-border'
                    : ''
                }`}
              >
                Continue <ChevronRight size={16} />
              </Button>
            ) : (
              <Button fullWidth onClick={handlePublish}>
                Publish LinkUp
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function AccessOption({
  active,
  onClick,
  icon,
  title,
  subtitle,
}: {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  title: string;
  subtitle: string;
}) {
  return (
    <button
      onClick={onClick}
      className={`w-full flex items-center gap-3 p-3 rounded-2xl border transition-all active:scale-[0.98] ${
        active ? 'bg-red/10 border-red/50' : 'bg-elevated border-border'
      }`}
    >
      <div className={`w-10 h-10 rounded-xl flex items-center justify-center ${active ? 'bg-red/20 text-red' : 'bg-zone text-text-dimmed'}`}>
        {icon}
      </div>
      <div className="flex-1 text-left">
        <div className={`text-sm font-body font-semibold ${active ? 'text-text-primary' : 'text-text-dimmed'}`}>{title}</div>
        <div className="text-xs text-text-muted">{subtitle}</div>
      </div>
      {active && <Check size={18} className="text-red" />}
    </button>
  );
}
