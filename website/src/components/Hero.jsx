import { useState, useEffect, useRef, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { ArrowRight, Copy, Check, Star, Github } from 'lucide-react'

const GITHUB_REPO = 'https://github.com/itsakash-real/nimbi'
const GITHUB_PROFILE = 'https://github.com/itsakash-real'

const INSTALL_TABS = [
  { id: 'one-liner', label: 'One-liner', cmd: 'curl -sSL https://raw.githubusercontent.com/itsakash-real/nimbi/main/install.sh | bash' },
  { id: 'brew', label: 'brew', cmd: 'brew install itsakash-real/tap/rw' },
  { id: 'go', label: 'Go', cmd: 'go install github.com/itsakash-real/nimbi/cmd/rw@latest' },
]

function QuickStart() {
  const [tab, setTab] = useState('one-liner')
  const [copied, setCopied] = useState(false)
  const active = INSTALL_TABS.find((t) => t.id === tab)

  const copy = async () => {
    await navigator.clipboard.writeText(active.cmd)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="w-full max-w-xl mx-auto">
      <div className="rounded-xl border border-[rgba(56,189,248,0.3)] overflow-hidden bg-[rgba(56,189,248,0.03)] hover:border-[rgba(56,189,248,0.5)] transition-all duration-200">
        {/* Terminal chrome */}
        <div className="flex items-center gap-2 px-4 py-2.5 bg-[#0d0d0d] border-b border-border">
          <div className="flex gap-1.5">
            <div className="w-2.5 h-2.5 rounded-full bg-[#ff5f57]" />
            <div className="w-2.5 h-2.5 rounded-full bg-[#febc2e]" />
            <div className="w-2.5 h-2.5 rounded-full bg-[#28c840]" />
          </div>
          <div className="flex gap-1 ml-3">
            {INSTALL_TABS.map((t) => (
              <button
                key={t.id}
                onClick={() => { setTab(t.id); setCopied(false) }}
                className={`px-2.5 py-0.5 rounded text-[11px] font-mono transition-all duration-150 ${
                  tab === t.id
                    ? 'bg-[rgba(56,189,248,0.2)] text-[#38bdf8]'
                    : 'text-text-muted hover:text-text-secondary'
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>
        </div>

        {/* Command */}
        <div className="flex items-center bg-[#080808] px-5 py-4">
          <div className="flex-1 font-mono text-[17px] text-text-secondary overflow-x-auto whitespace-nowrap">
            <span className="text-[#38bdf8] mr-2 select-none">$</span>
            {active.cmd}
          </div>
          <button
            onClick={copy}
            className="ml-3 p-1.5 rounded-md text-text-muted hover:text-text hover:bg-white/[0.05] transition-colors shrink-0"
            title="Copy"
          >
            {copied ? <Check size={14} className="text-success" /> : <Copy size={14} />}
          </button>
        </div>
      </div>

      <p className="text-center text-[14px] text-text-muted mt-3">
        Works on macOS, Windows & Linux.{' '}
        <a href={`${GITHUB_REPO}/releases`} target="_blank" rel="noopener noreferrer" className="underline underline-offset-2 hover:text-text-secondary transition-colors">
          Download binary
        </a>{' '}
        for Windows.
      </p>
    </div>
  )
}

/* Timeline visualization */
function Timeline() {
  const dots = [
    { label: 'auth', type: 'past' },
    { label: 'refactor', type: 'past' },
    { label: 'tests', type: 'past' },
    { label: 'HEAD', type: 'head' },
  ]

  return (
    <div className="flex items-center justify-center gap-0 mt-8 mb-2">
      {dots.map((dot, i) => (
        <div key={dot.label} className="flex items-center">
          <div className="flex flex-col items-center">
            <div
              className={`rounded-full ${
                dot.type === 'head'
                  ? 'w-3.5 h-3.5 bg-[#38bdf8] timeline-head-pulse'
                  : 'w-2.5 h-2.5 bg-[#38bdf8]/40'
              }`}
              style={dot.type === 'head' ? { boxShadow: '0 0 12px rgba(56,189,248,0.6)' } : {}}
            />
            <span className="font-mono text-[11px] text-text-muted mt-1.5">{dot.label}</span>
          </div>
          {i < dots.length - 1 && (
            <div className="w-12 sm:w-16 h-px bg-[#38bdf8]/30 mx-1" />
          )}
        </div>
      ))}
    </div>
  )
}

/* Live counter */
function LiveCounter() {
  const [count, setCount] = useState(47)

  useEffect(() => {
    const tick = () => {
      setCount((c) => c + Math.floor(Math.random() * 2) + 1)
      setTimeout(tick, 8000 + Math.random() * 4000)
    }
    const timer = setTimeout(tick, 8000 + Math.random() * 4000)
    return () => clearTimeout(timer)
  }, [])

  return (
    <p className="text-center text-[13px] text-[#38bdf8] opacity-70 mt-3">
      &uarr; {count} saves in the last hour
    </p>
  )
}

/* Particle burst on logo click */
function spawnParticles(logoEl) {
  const rect = logoEl.getBoundingClientRect()
  const cx = rect.left + rect.width / 2
  const cy = rect.top + rect.height / 2

  for (let i = 0; i < 8; i++) {
    const angle = (Math.PI * 2 * i) / 8
    const particle = document.createElement('div')
    particle.style.cssText = `
      position: fixed;
      width: 4px; height: 4px;
      background: #38bdf8;
      border-radius: 50%;
      left: ${cx}px; top: ${cy}px;
      pointer-events: none;
      z-index: 9999;
      transition: all 0.35s cubic-bezier(0.25, 0.46, 0.45, 0.94);
      opacity: 1;
    `
    document.body.appendChild(particle)

    requestAnimationFrame(() => {
      particle.style.transform = `translate(${Math.cos(angle) * 40}px, ${Math.sin(angle) * 40}px)`
      particle.style.opacity = '0'
    })

    setTimeout(() => particle.remove(), 400)
  }
}

export default function Hero() {
  const logoRef = useRef(null)

  const handleLogoClick = useCallback(() => {
    const logo = logoRef.current
    if (!logo) return
    logo.classList.remove('logo-clicked')
    void logo.offsetWidth
    logo.classList.add('logo-clicked')
    spawnParticles(logo)
    setTimeout(() => logo.classList.remove('logo-clicked'), 500)
  }, [])

  return (
    <section className="relative pt-[60px] pb-24">

      <div className="max-w-3xl mx-auto px-6 text-center relative z-[1]">
        {/* Logo — hero centerpiece */}
        <motion.div
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.5 }}
          className="mb-6"
        >
          <img
            ref={logoRef}
            src="/logo.png"
            alt="Nimbi"
            className="logo-hero w-[360px] h-[360px] mx-auto"
            onClick={handleLogoClick}
          />
        </motion.div>

        {/* Tagline above H1 */}
        <motion.p
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="font-mono text-[13px] uppercase tracking-[0.15em] mb-4 font-medium hero-tagline"
        >
          Like Ctrl+Z for your entire project
        </motion.p>

        {/* H1 */}
        <motion.h1
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.15 }}
          className="text-[48px] md:text-[72px] font-extrabold tracking-[-0.02em] mb-4 leading-[1.1]"
        >
          Stop <span className="hero-losing-gradient">losing</span> working states.
        </motion.h1>

        {/* Description */}
        <motion.p
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="text-[18px] text-[#94a3b8] leading-[1.8] mb-10 max-w-[520px] mx-auto"
        >
          Saves your entire project as a checkpoint. Go back to any point instantly.
          Tracks everything git ignores. A single binary, zero config.
        </motion.p>

        {/* CTAs */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.25 }}
          className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-4"
        >
          <Link
            to="/docs"
            className="group relative inline-flex w-full sm:w-auto items-center justify-center gap-2 px-8 py-3.5 rounded-xl text-[16px] font-semibold text-black bg-white hover:bg-gray-100 hover:shadow-glow-md transition-all duration-300 active:scale-[0.98] overflow-hidden"
          >
            <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/40 to-transparent translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-700 pointer-events-none" />
            Get started
            <ArrowRight size={16} className="group-hover:translate-x-1 transition-transform duration-300" />
          </Link>

          <a
            href={GITHUB_REPO}
            target="_blank"
            rel="noopener noreferrer"
            className="group flex w-full sm:w-auto items-center justify-center gap-2 px-8 py-3.5 rounded-xl text-[16px] font-medium text-white bg-[rgba(255,255,255,0.05)] border border-[rgba(255,255,255,0.1)] backdrop-blur-md hover:border-[rgba(56,189,248,0.5)] hover:bg-[rgba(56,189,248,0.1)] hover:shadow-glow-sm transition-all duration-300 active:scale-[0.98]"
          >
            <Star size={18} className="text-[#fbbf24] group-hover:fill-[#fbbf24]/20 transition-all duration-300" />
            Star on GitHub
          </a>
        </motion.div>

        {/* Profile Link */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.3 }}
          className="flex justify-center mb-12"
        >
          <a
            href={GITHUB_PROFILE}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 text-[14px] text-text-muted hover:text-text transition-colors duration-200"
          >
            Built by <Github size={14} className="ml-1" /> <span className="underline underline-offset-4 cursor-pointer hover:text-[#38bdf8]">itsakash-real</span>
          </a>
        </motion.div>

        {/* Quick Start Terminal */}
        <motion.div
          initial={{ opacity: 0, y: 15 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.3 }}
        >
          <QuickStart />
          <LiveCounter />
          <Timeline />
        </motion.div>
      </div>
    </section>
  )
}
