import { motion } from 'framer-motion'

const STEPS = [
  {
    step: '01',
    cmd: 'rw init',
    title: 'Set it up once',
    body: 'Run inside any project folder. Creates a hidden `.rewind/` directory. No config files, no accounts, no cloud.',
    badge: '5 seconds',
  },
  {
    step: '02',
    cmd: 'rw save "auth working"',
    title: 'Save whenever it works',
    body: 'Snapshots your entire folder — including files git ignores like `.env`, build outputs, and compiled binaries. Message is optional.',
    badge: '~180ms',
  },
  {
    step: '03',
    cmd: 'rw undo',
    title: 'Go back instantly',
    body: "No IDs, no hunting through history. Just `rw undo`. Or `rw goto <id>` if you want a specific point. Only writes files that actually changed.",
    badge: '~40ms',
  },
]

export default function HowItWorks() {
  return (
    <section className="py-28 relative">
      {/* Background gradient */}
      <div className="absolute inset-0 bg-purple-glow pointer-events-none" />

      <div className="relative max-w-6xl mx-auto px-6">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          className="text-center mb-16"
        >
          <div className="inline-block text-xs font-mono text-purple-DEFAULT border border-purple-glow/30 bg-purple-glow/5 rounded-full px-3 py-1 mb-4">
            how it works
          </div>
          <h2 className="text-4xl sm:text-5xl font-bold tracking-tight text-gradient-white">
            Three commands. That's it.
          </h2>
        </motion.div>

        {/* Steps */}
        <div className="space-y-4">
          {STEPS.map((s, i) => (
            <motion.div
              key={s.step}
              initial={{ opacity: 0, x: -24 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
              transition={{ delay: i * 0.12, duration: 0.5 }}
              className="group flex gap-6 p-6 rounded-2xl bg-surface border border-border hover:border-border-light transition-all"
            >
              {/* Step number */}
              <div className="hidden sm:flex items-center justify-center w-12 h-12 shrink-0 rounded-xl border border-border bg-bg font-mono text-sm text-text-muted group-hover:border-purple-glow/30 group-hover:text-purple-DEFAULT transition-colors">
                {s.step}
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex flex-wrap items-center gap-3 mb-2">
                  <code className="px-3 py-1 rounded-lg bg-bg border border-border font-mono text-sm text-purple-bright">
                    {s.cmd}
                  </code>
                  <span className="text-xs font-mono text-cyan border border-cyan-dim/30 bg-cyan-dim/10 rounded-full px-2 py-0.5">
                    {s.badge}
                  </span>
                </div>
                <h3 className="font-semibold text-text text-lg mb-1">{s.title}</h3>
                <p className="text-text-muted text-sm leading-relaxed">{s.body}</p>
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  )
}
