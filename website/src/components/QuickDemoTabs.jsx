import { useState } from 'react'
import { motion } from 'framer-motion'

const DEMOS = [
  {
    id: 'basic',
    label: 'Basic workflow',
    lines: [
      { type: 'cmd', text: '$ rw init' },
      { type: 'out', text: '  initialized on branch \'main\'', color: 'text' },
      { type: 'gap' },
      { type: 'cmd', text: '$ rw save "login working"' },
      { type: 'out', text: '  S1  login working  \u00B7  main  \u00B7  12 files', color: 'text' },
      { type: 'gap' },
      { type: 'cmd', text: '$ rw save "added payments"' },
      { type: 'out', text: '  S2  added payments  \u00B7  main  \u00B7  3 changed', color: 'text' },
      { type: 'gap' },
      { type: 'cmd', text: '$ rw goto HEAD~1' },
      { type: 'out', text: '  restored to S1  \u00B7  3 files written', color: 'text' },
    ],
  },
  {
    id: 'rollback',
    label: 'Auto-rollback',
    lines: [
      { type: 'cmd', text: '$ rw run "npm run build"' },
      { type: 'out', text: '  checkpoint saved: a3f2b1c8', color: 'success' },
      { type: 'out', text: '  running: npm run build...', color: 'muted' },
      { type: 'out', text: '  command failed (exit 1)', color: 'error' },
      { type: 'out', text: '  rolling back...', color: 'muted' },
      { type: 'out', text: '  rolled back to a3f2b1c8', color: 'success' },
      { type: 'gap' },
      { type: 'cmd', text: '$ rw run "go test ./..."' },
      { type: 'out', text: '  checkpoint saved: b2e1a0f4', color: 'success' },
      { type: 'out', text: '  running tests...', color: 'muted' },
      { type: 'out', text: '  all tests passed', color: 'success' },
    ],
  },
  {
    id: 'explore',
    label: 'Two approaches',
    lines: [
      { type: 'cmd', text: '$ rw save "base: working auth"' },
      { type: 'out', text: '  a3f2b1c8  base: working auth', color: 'text' },
      { type: 'gap' },
      { type: 'cmd', text: '$ rw save "approach A: JWT"' },
      { type: 'out', text: '  b2e1a0f4  approach A: JWT', color: 'text' },
      { type: 'gap' },
      { type: 'cmd', text: '$ rw goto a3f2b1c8' },
      { type: 'out', text: '  # auto-creates new branch', color: 'muted' },
      { type: 'gap' },
      { type: 'cmd', text: '$ rw save "approach B: sessions"' },
      { type: 'out', text: '  c1d0e9f2  on branch: experiment-1', color: 'text' },
      { type: 'gap' },
      { type: 'cmd', text: '$ rw list --all' },
      { type: 'out', text: '  main:        a3f2b1c8 \u2500\u2500 b2e1a0f4', color: 'text' },
      { type: 'out', text: '  experiment:  a3f2b1c8 \u2500\u2500 c1d0e9f2 (HEAD)', color: 'accent' },
    ],
  },
  {
    id: 'bisect',
    label: 'Find the bug',
    lines: [
      { type: 'cmd', text: '$ rw bisect start' },
      { type: 'cmd', text: '$ rw bisect good a3f2b1c8' },
      { type: 'cmd', text: '$ rw bisect bad HEAD' },
      { type: 'out', text: '  jumping to midpoint: d0d2536c', color: 'text' },
      { type: 'out', text: '  test your code, then:', color: 'muted' },
      { type: 'out', text: '  rw bisect good  OR  rw bisect bad', color: 'muted' },
      { type: 'gap' },
      { type: 'out', text: '  # After 3 steps:', color: 'muted' },
      { type: 'out', text: '  found: bug introduced at e1f2a3b4', color: 'success' },
      { type: 'out', text: '  "added new auth middleware"', color: 'muted' },
    ],
  },
]

const COLOR_MAP = {
  accent: 'text-accent/80',
  success: 'text-success',
  error: 'text-error',
  muted: 'text-text-muted',
  text: 'text-text-secondary',
}

export default function QuickDemoTabs() {
  const [active, setActive] = useState('basic')
  const demo = DEMOS.find((d) => d.id === active)

  return (
    <section className="py-32">
      <div className="max-w-content mx-auto px-6">
        <div className="mb-10">
          <h2 className="text-[clamp(1.8rem,4vw,2.5rem)] font-bold tracking-[-0.03em] text-text">
            See it in action
          </h2>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 mb-6 overflow-x-auto">
          {DEMOS.map((d) => (
            <button
              key={d.id}
              onClick={() => setActive(d.id)}
              className={`px-3 py-1.5 rounded-md text-[13px] font-mono whitespace-nowrap transition-all duration-150 ${
                active === d.id
                  ? 'bg-white/[0.06] text-text border border-border-hover'
                  : 'text-text-muted hover:text-text-secondary border border-transparent'
              }`}
            >
              {d.label}
            </button>
          ))}
        </div>

        {/* Terminal */}
        <motion.div
          key={active}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.15 }}
          className="max-w-3xl"
        >
          <div className="rounded-xl border border-border">
            <div className="flex items-center gap-2 px-4 py-2.5 bg-[#0d0d0d] border-b border-border">
              <div className="flex gap-1.5">
                <div className="w-2.5 h-2.5 rounded-full bg-white/[0.06]" />
                <div className="w-2.5 h-2.5 rounded-full bg-white/[0.06]" />
                <div className="w-2.5 h-2.5 rounded-full bg-white/[0.06]" />
              </div>
              <span className="text-[11px] text-text-muted font-mono ml-3">{demo.label}</span>
            </div>
            <div className="bg-[#080808] p-5 font-mono text-[13px] leading-[1.7]">
              {demo.lines.map((line, i) => {
                if (line.type === 'gap') return <div key={i} className="h-2" />
                if (line.type === 'cmd')
                  return <div key={i} className="text-text mb-0.5">{line.text}</div>
                return (
                  <div key={i} className={`mb-0.5 ${COLOR_MAP[line.color] || 'text-text'}`}>
                    {line.text}
                  </div>
                )
              })}
            </div>
          </div>
        </motion.div>
      </div>
    </section>
  )
}
