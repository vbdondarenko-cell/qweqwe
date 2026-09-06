interface AvatarProps {
  initials: string;
  color: string;
  size?: 'sm' | 'md' | 'lg' | 'xl';
}

const sizeClasses = {
  sm: 'w-6 h-6 text-[10px]',
  md: 'w-8 h-8 text-xs',
  lg: 'w-12 h-12 text-sm',
  xl: 'w-20 h-20 text-2xl',
};

export function Avatar({ initials, color, size = 'md' }: AvatarProps) {
  return (
    <div
      className={`${sizeClasses[size]} rounded-full flex items-center justify-center font-body font-bold text-text-primary no-select`}
      style={{ backgroundColor: `${color}22`, border: `1.5px solid ${color}` }}
    >
      {initials}
    </div>
  );
}
