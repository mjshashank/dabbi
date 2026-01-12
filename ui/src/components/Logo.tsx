interface LogoProps {
  size?: number
  className?: string
}

export default function Logo({ size = 32, className = '' }: LogoProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      {/* Two stacked rounded rectangles - clean VM/container symbol */}
      <rect x="4" y="4" width="24" height="10" rx="2" fill="currentColor" />
      <rect x="4" y="18" width="24" height="10" rx="2" fill="currentColor" fillOpacity="0.5" />
    </svg>
  )
}
