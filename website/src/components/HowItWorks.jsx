import { useState } from 'react'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'

const BORDER_COLORS = ['border-l-[#38bdf8]', 'border-l-[#6366f1]', 'border-l-[#34d399]']
const NUM_COLORS = ['text-[#38bdf8]', 'text-[#6366f1]', 'text-[#34d399]']
const GLOW_SHADOWS = [
  '0 4px 20px rgba(56,189,248,0.15)',
  '0 4px 20px rgba(99,102,241,0.15)',
  '0 4px 20px rgba(52,211,153,0.15)',
]

const STEPS = [
  {
    num: '01',
    title: 'Save',
    cmd: 'rw save "auth working"',
    body: 'Checkpoint your entire project state. Message optional — Nimbi writes one for you.',
    output: [
      { text: '  checkpoint saved', c: 'text' },
      { text: '  id       a3f2b1c8', c: 'muted' },
      { text: '  files    24 tracked', c: 'muted' },
    ],
  },
  {
    num: '02',
    title: 'Work',
    cmd: 'rw run "npm run build"',
    body: 'Wrap risky commands. Auto-checkpoints before, auto-rolls back on failure.',
    output: [
      { text: '  checkpoint saved: a3f2b1c8', c: 'success' },
      { text: '  running: npm run build', c: 'muted' },
      { text: '  command failed (exit 1)', c: 'error' },
      { text: '  rolled back to a3f2b1c8', c: 'success' },
    ],
  },
  {
    num: '03',
    title: 'Go back',
    cmd: 'rw goto HEAD~3',
    body: 'Restore any checkpoint by ID, tag, or relative ref. Works on everything.',
    output: [
      { text: '  restored to a3f2b1c8', c: 'text' },
      { text: '  written  3 file(s)', c: 'muted' },
    ],
  },
]

const OUT_COLORS = {
  text: 'text-text-secondary',
  muted: 'text-text-muted',
  success: 'text-success',
  error: 'text-error',
}

export default function HowItWorks() {
  const [active, setActive] = useState(0)

  return (
    <section className="py-24">
      <div className="max-w-4xl mx-auto px-6">
        <div className="scroll-reveal text-center mb-14">
          <h2 className="text-[28px] md:text-[48px] font-bold tracking-[-0.02em] text-text mb-3">
            Quick Start
          </h2>
          <p className="text-[18px] text-text-muted max-w-md mx-auto">
            Three commands. That's the whole thing.
          </p>
        </div>

        <div className="grid lg:grid-cols-2 gap-10 items-start max-w-[900px] mx-auto">
          {/* Left — steps */}
          <div className="space-y-2 max-w-[640px] mx-auto w-full">
            {STEPS.map((s, i) => (
              <button
                key={s.num}
                onClick={() => setActive(i)}
                className={`w-full text-left p-6 rounded-xl border-l-[3px] border transition-all duration-200 hover:translate-y-[-3px] ${BORDER_COLORS[i]} ${
                  active === i
                    ? 'border-r-transparent border-t-transparent border-b-transparent bg-surface'
                    : 'border-r-transparent border-t-transparent border-b-transparent hover:bg-surface/50'
                }`}
                style={active === i ? { boxShadow: GLOW_SHADOWS[i] } : {}}
              >
                <div className="flex items-baseline gap-3 mb-2">
                  <span className={`text-[32px] font-mono font-bold opacity-40 ${NUM_COLORS[i]}`}>{s.num}</span>
                  <span className="text-[22px] font-semibold text-text">{s.title}</span>
                </div>
                <code className="text-[15px] font-mono text-[#38bdf8]/80 block mb-2 cmd-tooltip" data-tooltip={
                  i === 0 ? 'Saves entire project state' : i === 1 ? 'Auto-checkpoint + rollback' : 'Restore any checkpoint'
                }>{s.cmd}</code>
                <p className="text-[15px] text-text-muted leading-relaxed">{s.body}</p>
              </button>
            ))}
          </div>

          {/* Right — terminal */}
          <div className="lg:sticky lg:top-24 scroll-reveal" style={{ transitionDelay: '80ms' }}>
            <div className="rounded-xl border border-border overflow-hidden">
              <div className="flex items-center gap-2 px-4 py-2.5 bg-[#0d0d0d] border-b border-border">
                <div className="flex gap-1.5">
                  <div className="w-2.5 h-2.5 rounded-full bg-[#ff5f57]" />
                  <div className="w-2.5 h-2.5 rounded-full bg-[#febc2e]" />
                  <div className="w-2.5 h-2.5 rounded-full bg-[#28c840]" />
                </div>
              </div>
              <div className="bg-[#080808] p-5 font-mono text-[15px] min-h-[200px] leading-[1.7]">
                <div className="flex items-start gap-2 mb-1">
                  <span className="text-[#38bdf8] select-none">$</span>
                  <motion.span
                    key={active}
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    className="text-text"
                  >
                    {STEPS[active].cmd}
                  </motion.span>
                </div>
                {STEPS[active].output.map((line, i) => (
                  <motion.div
                    key={`${active}-${i}`}
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ delay: i * 0.06 }}
                    className={OUT_COLORS[line.c]}
                  >
                    {line.text}
                  </motion.div>
                ))}
              </div>
            </div>

            <p className="text-[16px] text-text-muted mt-5 text-center">
              That's the core.{' '}
              <Link to="/docs" className="text-[#38bdf8]/80 hover:text-[#38bdf8] underline underline-offset-2 transition-colors">
                See all commands &rarr;
              </Link>
            </p>
          </div>
        </div>
      </div>
    </section>
  )
}
