import { type ButtonHTMLAttributes, type ReactNode } from 'react';

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'success';
type Size = 'sm' | 'md' | 'lg';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
  children: ReactNode;
  fullWidth?: boolean;
}

const variantClasses: Record<Variant, string> = {
  primary: 'bg-red text-text-primary hover:bg-red-signal active:bg-red-deep',
  secondary: 'bg-elevated text-text-primary border border-border hover:bg-zone active:bg-border',
  ghost: 'bg-transparent text-text-dimmed hover:bg-elevated hover:text-text-primary',
  danger: 'bg-critical/15 text-critical border border-critical/30 hover:bg-critical/25',
  success: 'bg-success/15 text-success border border-success/30 hover:bg-success/25',
};

const sizeClasses: Record<Size, string> = {
  sm: 'px-3 py-1.5 text-xs rounded-lg',
  md: 'px-4 py-2.5 text-sm rounded-xl',
  lg: 'px-5 py-3.5 text-base rounded-2xl',
};

export function Button({
  variant = 'primary',
  size = 'md',
  children,
  fullWidth,
  className = '',
  ...props
}: ButtonProps) {
  return (
    <button
      className={`font-body font-semibold transition-all duration-200 active:scale-95 no-select ${variantClasses[variant]} ${sizeClasses[size]} ${fullWidth ? 'w-full' : ''} ${className}`}
      {...props}
    >
      {children}
    </button>
  );
}
