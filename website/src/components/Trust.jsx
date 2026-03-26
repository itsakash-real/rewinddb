import { motion } from 'framer-motion'
import { Shield, Package, Cpu, Lock, Zap, Code } from 'lucide-react'

const PILLS = [
  { icon: Package,  label: 'Single binary' },
  { icon: Lock,     label: 'No cloud. Ever.' },
  { icon: Shield,   label: 'MIT license' },
  { icon: Cpu,      label: 'Parallel hashing' },
  { icon: Zap,      label: 'Atomic writes' },
  { icon: Code,     label: 'Open source' },
]

const QUOTES = [
  {
    text: "This is what I wish existed when I was neck-deep in an AI code session. One command to undo everything.",
    author: "Developer on Reddit",
  },
  {
    text: "Finally. Git commits mid-experiment feel like overkill. rw save just works.",
    author: "Hacker News comment",
  },
  {
    text: "I use this before every 'quick' config change that always breaks something.",
    author: "Developer, Twitter",
  },
]

export default function Trust() {
  return (
    <section className="py-28 relative">
      <div className="absolute inset-0 bg-hero-glow opacity-50 pointer-events-none" />

      <div className="relative max-w-6xl mx-auto px-6">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          className="text-center mb-12"
        >
          <div className="inline-block text-xs font-mono text-purple-DEFAULT border border-purple-glow/30 bg-purple-glow/5 rounded-full px-3 py-1 mb-4">
            trust
          </div>
          <h2 className="text-4xl sm:text-5xl font-bold tracking-tight text-gradient-white mb-4">
            Built for developers
            <br />
            who hate surprises.
          </h2>
        </motion.div>

        {/* Pills */}
        <motion.div
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true }}
          className="flex flex-wrap justify-center gap-3 mb-16"
        >
          {PILLS.map((p, i) => (
            <motion.div
              key={p.label}
              initial={{ opacity: 0, scale: 0.9 }}
              whileInView={{ opacity: 1, scale: 1 }}
              viewport={{ once: true }}
              transition={{ delay: i * 0.07 }}
              className="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-border bg-surface text-sm font-medium text-text hover:border-border-light hover:bg-[#11111a] transition-all"
            >
              <p.icon size={15} className="text-purple-DEFAULT" />
              {p.label}
            </motion.div>
          ))}
        </motion.div>

        {/* Technical detail */}
        <div className="grid sm:grid-cols-3 gap-4 mb-16">
          {[
            {
              title: 'Crash-safe writes',
              body: 'Every save goes to a temp file first, fsynced to disk, then atomically renamed. Power cut mid-save? Nothing corrupted.',
            },
            {
              title: 'Content deduplication',
              body: 'Files stored once by SHA-256 hash. Save 50 checkpoints — unchanged files take zero extra space.',
            },
            {
              title: 'Fast recovery',
              body: 'Auto-detects and recovers from interrupted saves on startup. Your repository is always in a valid state.',
            },
          ].map((item, i) => (
            <motion.div
              key={item.title}
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: i * 0.1 }}
              className="p-5 rounded-2xl border border-border bg-surface"
            >
              <div className="w-1 h-8 rounded-full bg-purple-glow mb-4" />
              <h3 className="font-semibold text-text mb-2">{item.title}</h3>
              <p className="text-text-muted text-sm leading-relaxed">{item.body}</p>
            </motion.div>
          ))}
        </div>

        {/* Testimonials */}
        <div className="grid sm:grid-cols-3 gap-4">
          {QUOTES.map((q, i) => (
            <motion.div
              key={i}
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: i * 0.1 }}
              className="p-5 rounded-2xl border border-border bg-surface/50"
            >
              <p className="text-text text-sm leading-relaxed mb-4 italic">"{q.text}"</p>
              <p className="text-text-muted text-xs">— {q.author}</p>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  )
}
