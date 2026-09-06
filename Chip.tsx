import { type ReactNode } from 'react';

interface ChipProps {
  active?: boolean;
  onClick?: () => void;
  children: ReactNode;
  icon?: ReactNode;
  className?: string;
}

export function Chip({ active, onClick, children, icon, className = '' }: ChipProps) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-xs font-body font-semibold transition-all duration-200 active:scale-95 no-select whitespace-nowrap ${
        active
          ? 'bg-red text-text-primary'
          : 'bg-elevated text-text-dimmed border border-border hover:bg-zone hover:text-text-primary'
      } ${className}`}
    >
      {icon}
      {children}
    </button>
  );
}
