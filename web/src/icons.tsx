import type { ReactNode } from 'react'

export type IconName =
  | 'arrow-up-right'
  | 'book'
  | 'briefcase'
  | 'brand'
  | 'chart'
  | 'chevron-down'
  | 'chevron-right'
  | 'close'
  | 'compass'
  | 'home'
  | 'journal'
  | 'moon'
  | 'plus'
  | 'roadmap'
  | 'server'
  | 'sun'

interface IconProps {
  name: IconName
  size?: number
  strokeWidth?: number
  className?: string
}

function Svg({ name, size, strokeWidth, className, children }: IconProps & { children: ReactNode }) {
  return (
    <svg
      className={`ui-icon${className ? ` ${className}` : ''}`}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      {children}
    </svg>
  )
}

export function Icon({ name, size = 18, strokeWidth = 1.8, className }: IconProps) {
  const props = { name, size, strokeWidth, className }
  switch (name) {
    case 'brand':
      return <Svg {...props}><path d="m12 3 8 4.5v9L12 21l-8-4.5v-9L12 3Z" /><path d="m8 9 4 2.3L16 9M12 11.3V17" /></Svg>
    case 'home':
      return <Svg {...props}><path d="m3 10 9-7 9 7" /><path d="M5 9.5V21h14V9.5M9 21v-6h6v6" /></Svg>
    case 'roadmap':
      return <Svg {...props}><circle cx="5" cy="5" r="2.5" /><circle cx="19" cy="5" r="2.5" /><circle cx="12" cy="19" r="2.5" /><path d="M7.5 5h9M12 7.5v9" /></Svg>
    case 'book':
      return <Svg {...props}><path d="M4.5 5.5A2.5 2.5 0 0 1 7 3h12.5v17H7a2.5 2.5 0 0 0-2.5 2.5v-17Z" /><path d="M4.5 20.5A2.5 2.5 0 0 1 7 18h12.5M8 7h8M8 11h6" /></Svg>
    case 'journal':
      return <Svg {...props}><circle cx="12" cy="12" r="8.5" /><path d="M12 7v5l3.5 2M5 5l-1.5-1.5M19 5l1.5-1.5" /></Svg>
    case 'briefcase':
      return <Svg {...props}><rect x="3.5" y="7" width="17" height="12.5" rx="2" /><path d="M8 7V5.5A2 2 0 0 1 10 3.5h4a2 2 0 0 1 2 2V7M3.5 12h17M10 12v2h4v-2" /></Svg>
    case 'chart':
      return <Svg {...props}><path d="M4 19.5V4.5M4 19.5h17" /><path d="m7 15 3-3 2.5 2 5-6 2.5 2" /></Svg>
    case 'sun':
      return <Svg {...props}><circle cx="12" cy="12" r="3.5" /><path d="M12 2.5v2M12 19.5v2M4.8 4.8l1.4 1.4M17.8 17.8l1.4 1.4M2.5 12h2M19.5 12h2M4.8 19.2l1.4-1.4M17.8 6.2l1.4-1.4" /></Svg>
    case 'moon':
      return <Svg {...props}><path d="M20 15.5A8.5 8.5 0 0 1 8.5 4 8.5 8.5 0 1 0 20 15.5Z" /></Svg>
    case 'plus':
      return <Svg {...props}><path d="M12 5v14M5 12h14" /></Svg>
    case 'close':
      return <Svg {...props}><path d="m6 6 12 12M18 6 6 18" /></Svg>
    case 'chevron-down':
      return <Svg {...props}><path d="m5 9 7 7 7-7" /></Svg>
    case 'chevron-right':
      return <Svg {...props}><path d="m9 5 7 7-7 7" /></Svg>
    case 'arrow-up-right':
      return <Svg {...props}><path d="M6 18 18 6M8 6h10v10" /></Svg>
    case 'compass':
      return <Svg {...props}><circle cx="12" cy="12" r="8.5" /><path d="m15.5 8.5-2.2 4.8-4.8 2.2 2.2-4.8 4.8-2.2Z" /></Svg>
    case 'server':
      return <Svg {...props}><rect x="4" y="4" width="16" height="6" rx="1.5" /><rect x="4" y="14" width="16" height="6" rx="1.5" /><path d="M8 7h.01M8 17h.01M11 7h5M11 17h5" /></Svg>
    default:
      return <Svg {...props}><circle cx="12" cy="12" r="8.5" /><path d="m15.5 8.5-2.2 4.8-4.8 2.2 2.2-4.8 4.8-2.2Z" /></Svg>
  }
}
