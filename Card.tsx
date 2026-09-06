import { type ReactNode } from 'react';

interface CardProps {
  children: ReactNode;
  className?: string;
  onClick?: () => void;
}

export function Card({ children, className = '', onClick }: CardProps) {
  return (
    <div
      onClick={onClick}
      className={`bg-elevated border border-border rounded-2xl transition-all duration-200 ${onClick ? 'cursor-pointer active:scale-[0.98] hover:border-border/80' : ''} ${className}`}
    >
      {children}
    </div>
  );
}
