import { motion } from 'framer-motion'

const FEATURES = [
  {
    icon: '⚡',
    title: 'Instant checkpoints',
    body: '~180ms to snapshot 1,000 files using parallel SHA-256 hashing across all CPU cores.',
    tag: 'Performance',
  },
  {
    icon: '🌿',
    title: 'Auto-branching timelines',
    body: 'Go back to an old checkpoint and save something new? Drift auto-creates a branch. Both timelines preserved.',
    tag: 'Smart',
  },
  {
    icon: '🔍',
    title: 'Zero duplication',
    body: 'Same file in 10 checkpoints? Stored once. Content-addressable storage means only changed files take space.',
    tag: 'Storage',
  },
  {
    icon: '🛡️',
    title: 'Safe by default',
    body: 'Warns before overwriting. Auto-stashes unsaved changes. Atomic writes — a crash mid-save leaves nothing corrupted.',
    tag: 'Safety',
  },
  {
    icon: '🤖',
    title: 'Runs before risky commands',
    body: '`rw run "npm run build"` checkpoints before running. Fails? Rolled back automatically. Passes? Saves a "✓ passed" checkpoint.',
    tag: 'Automation',
  },
  {
    icon: '🐛',
    title: 'Bisect to find bugs',
    body: 'Binary-search your checkpoint history to find exactly when a bug appeared. Like git bisect, but for full project state.',
    tag: 'Debug',
  },
  {
    icon: '👁️',
    title: 'Auto-save daemon',
    body: '`rw watch` runs in the background and saves automatically when files change. Never think about saving again.',
    tag: 'Workflow',
  },
  {
    icon: '📦',
    title: 'Export & share states',
    body: 'Export any checkpoint as a `.rwdb` file. Send it to a teammate. They can import the exact state — including untracked files.',
    tag: 'Sharing',
  },
  {
    icon: '🌐',
    title: 'Works with anything',
    body: 'React, Python, Go, Rust, Rails, Laravel — any language, any framework. Even plain folders with no build system.',
    tag: 'Universal',
  },
]

export default function Features() {
  return (
    <section className="py-28">
      <div className="max-w-6xl mx-auto px-6">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          className="text-center mb-16"
        >
          <div className="inline-block text-xs font-mono text-purple-DEFAULT border border-purple-glow/30 bg-purple-glow/5 rounded-full px-3 py-1 mb-4">
            features
          </div>
          <h2 className="text-4xl sm:text-5xl font-bold tracking-tight text-gradient-white mb-4">
            Everything you need.
            <br />
            Nothing you don't.
          </h2>
          <p className="text-text-muted text-lg max-w-lg mx-auto">
            No config. No cloud account. No background services eating RAM. One binary, drop it in your PATH.
          </p>
        </motion.div>

        {/* Grid */}
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {FEATURES.map((f, i) => (
            <motion.div
              key={f.title}
              initial={{ opacity: 0, y: 24 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: (i % 3) * 0.08, duration: 0.5 }}
              className="group p-5 rounded-2xl bg-surface border border-border hover:border-border-light hover:bg-[#11111a] transition-all duration-300"
            >
              <div className="flex items-start justify-between mb-3">
                <span className="text-2xl">{f.icon}</span>
                <span className="text-[10px] font-mono text-text-muted border border-border rounded-full px-2 py-0.5">
                  {f.tag}
                </span>
              </div>
              <h3 className="font-semibold text-text mb-1.5">{f.title}</h3>
              <p className="text-text-muted text-sm leading-relaxed">{f.body}</p>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  )
}
