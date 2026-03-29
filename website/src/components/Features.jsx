import { motion } from 'framer-motion'
import { Home, Clock, GitBranch, Shield, Zap, Package } from 'lucide-react'

const FEATURES = [
  {
    icon: Home,
    title: 'Runs Locally',
    body: 'Mac, Windows, or Linux. Single binary, no daemon, no server, no account. Your data stays yours.',
    color: '#38bdf8',
  },
  {
    icon: Clock,
    title: 'Instant Checkpoints',
    body: 'rw save. Entire project saved — binaries, configs, build artifacts. No staging, no commit message needed.',
    color: '#6366f1',
  },
  {
    icon: Shield,
    title: 'Auto-Rollback',
    body: 'rw run wraps any command. Checkpoint before, rollback on failure. Build scripts, migrations, tests.',
    color: '#34d399',
  },
  {
    icon: Package,
    title: 'Tracks Everything',
    body: 'node_modules, .env, compiled binaries, game assets. Nimbi tracks what git ignores. Dedup keeps storage tiny.',
    color: '#f59e0b',
  },
  {
    icon: GitBranch,
    title: 'Auto-Branching',
    body: 'Restore an old checkpoint and save? New branch auto-created. Both timelines preserved.',
    color: '#ec4899',
  },
  {
    icon: Zap,
    title: 'Blazing Fast',
    body: '~180ms for 1000 files. Content-addressable store with SHA-256 dedup and gzip compression.',
    color: '#38bdf8',
  },
]

export default function Features() {
  return (
    <section className="py-24">
      <div className="max-w-4xl mx-auto px-6">
        <div className="scroll-reveal text-center mb-14">
          <h2 className="text-[28px] md:text-[48px] font-bold tracking-[-0.02em] text-text mb-3">
            What It Does
          </h2>
        </div>

        <div className="grid md:grid-cols-3 gap-5">
          {FEATURES.map((f, i) => (
            <motion.div
              key={f.title}
              initial={{ opacity: 0, y: 12 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true, margin: '-30px' }}
              transition={{ duration: 0.35, delay: i * 0.08 }}
              className="scroll-reveal p-6 rounded-xl border border-[rgba(255,255,255,0.08)] bg-[rgba(255,255,255,0.02)] hover:border-[rgba(56,189,248,0.3)] hover:bg-[rgba(56,189,248,0.04)] transition-all duration-250 min-h-[160px]"
              style={{ transitionDelay: `${i * 80}ms` }}
            >
              <f.icon size={28} style={{ color: f.color }} className="mb-3" />
              <h3 className="text-[17px] font-semibold text-white mb-2">{f.title}</h3>
              <p className="text-[14px] text-[#94a3b8] leading-relaxed">{f.body}</p>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  )
}
