import { type LoadState } from '@/types';
import { Loader2, AlertCircle } from 'lucide-react';

interface StateToggleProps {
  state: LoadState;
  onChange: (state: LoadState) => void;
}

const states: LoadState[] = ['loading', 'content', 'empty', 'error'];

export function StateToggle({ state, onChange }: StateToggleProps) {
  return (
    <div className="fixed top-2 right-2 z-[60] flex items-center gap-1 glass border border-border rounded-full px-1 py-1">
      <Loader2 size={10} className="text-text-muted" />
      {states.map((s) => (
        <button
          key={s}
          onClick={() => onChange(s)}
          className={`px-2 py-0.5 rounded-full text-[9px] font-mono font-semibold transition-all no-select ${
            state === s ? 'bg-red text-text-primary' : 'text-text-muted hover:text-text-dimmed'
          }`}
        >
          {s}
        </button>
      ))}
    </div>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-20 px-6 text-center animate-fade-in">
      <div className="w-16 h-16 rounded-2xl bg-critical/10 border border-critical/20 flex items-center justify-center mb-4">
        <AlertCircle size={28} className="text-critical" />
      </div>
      <h3 className="font-display font-bold text-lg text-text-primary mb-1">Something went wrong</h3>
      <p className="text-sm text-text-dimmed mb-4">{message}</p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="px-4 py-2 rounded-xl bg-elevated border border-border text-sm font-body font-semibold text-text-primary active:scale-95 transition-transform"
        >
          Try again
        </button>
      )}
    </div>
  );
}

export function EmptyState({ title, subtitle, icon }: { title: string; subtitle: string; icon?: React.ReactNode }) {
  return (
    <div className="flex flex-col items-center justify-center py-20 px-6 text-center animate-fade-in">
      <div className="w-20 h-20 rounded-3xl bg-zone border border-border flex items-center justify-center mb-4 text-text-muted">
        {icon ?? (
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
            <circle cx="12" cy="12" r="10" />
            <path d="M8 12h8" strokeLinecap="round" />
          </svg>
        )}
      </div>
      <h3 className="font-display font-bold text-lg text-text-primary mb-1">{title}</h3>
      <p className="text-sm text-text-dimmed max-w-[240px]">{subtitle}</p>
    </div>
  );
}
