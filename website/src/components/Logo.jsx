export function LogoMark({ className = 'w-8 h-8' }) {
  return (
    <svg className={className} viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="lm-g1" x1="4" y1="4" x2="28" y2="28" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stopColor="#c4b5fd"/>
          <stop offset="100%" stopColor="#7c3aed"/>
        </linearGradient>
        <linearGradient id="lm-bg" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stopColor="#13101f"/>
          <stop offset="100%" stopColor="#0d0a1a"/>
        </linearGradient>
      </defs>
      <rect width="32" height="32" rx="7" fill="url(#lm-bg)"/>
      <rect x="0.5" y="0.5" width="31" height="31" rx="6.5" stroke="url(#lm-g1)" strokeOpacity="0.4"/>
      {/* Flowing drift curves */}
      <path d="M8 20 C12 20, 13 12, 16 12 C19 12, 20 20, 24 20" stroke="url(#lm-g1)" strokeWidth="2.5" strokeLinecap="round" fill="none"/>
      <path d="M8 16 C12 16, 13 8, 16 8 C19 8, 20 16, 24 16" stroke="url(#lm-g1)" strokeWidth="2.5" strokeLinecap="round" fill="none" strokeOpacity="0.4"/>
      {/* Time dot */}
      <circle cx="24" cy="20" r="2" fill="url(#lm-g1)"/>
    </svg>
  )
}

export function LogoFull({ className = 'h-8' }) {
  return (
    <svg className={className} viewBox="0 0 150 48" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="lf-g1" x1="6" y1="6" x2="42" y2="42" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stopColor="#c4b5fd"/>
          <stop offset="100%" stopColor="#7c3aed"/>
        </linearGradient>
        <linearGradient id="lf-bg" x1="0" y1="0" x2="48" y2="48" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stopColor="#13101f"/>
          <stop offset="100%" stopColor="#0d0a1a"/>
        </linearGradient>
        <linearGradient id="lf-text" x1="60" y1="0" x2="150" y2="0" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stopColor="#ffffff"/>
          <stop offset="60%" stopColor="#e8e8f0"/>
          <stop offset="100%" stopColor="#a0a0b8"/>
        </linearGradient>
      </defs>
      <rect width="48" height="48" rx="11" fill="url(#lf-bg)"/>
      <rect x="0.75" y="0.75" width="46.5" height="46.5" rx="10.25" stroke="url(#lf-g1)" strokeOpacity="0.4"/>
      {/* Flowing drift curves */}
      <path d="M11 30 C17 30, 19 18, 24 18 C29 18, 31 30, 37 30" stroke="url(#lf-g1)" strokeWidth="3.5" strokeLinecap="round" fill="none"/>
      <path d="M11 24 C17 24, 19 12, 24 12 C29 12, 31 24, 37 24" stroke="url(#lf-g1)" strokeWidth="3.5" strokeLinecap="round" fill="none" strokeOpacity="0.4"/>
      {/* Time dot */}
      <circle cx="37" cy="30" r="3" fill="url(#lf-g1)"/>
      <text x="60" y="31" fontFamily="Inter, system-ui, sans-serif" fontWeight="700" fontSize="19" fill="url(#lf-text)" letterSpacing="-0.3">Drift</text>
    </svg>
  )
}
