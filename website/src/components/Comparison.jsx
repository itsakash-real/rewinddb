import { motion } from 'framer-motion'
import { Check, X, Minus } from 'lucide-react'

const ROWS = [
  { label: 'Saves full project state',       git: false,  rw: true  },
  { label: 'Tracks .env & build artifacts',  git: false,  rw: true  },
  { label: 'Requires a commit message',      git: true,   rw: false },
  { label: 'Auto-branches on time-travel',   git: false,  rw: true  },
  { label: 'Rollback on command failure',    git: false,  rw: true  },
  { label: 'Background auto-save',           git: false,  rw: true  },
  { label: 'Works without internet',         git: true,   rw: true  },
  { label: 'Works in CI/CD',                 git: true,   rw: true  },
  { label: 'Collaboration & push/pull',      git: true,   rw: false },
  { label: 'Line-level history',             git: true,   rw: false },
]

function Cell({ value }) {
  if (value === true)
    return (
      <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-green-dim/20">
        <Check size={12} className="text-green" />
      </span>
    )
  if (value === false)
    return (
      <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-surface">
        <X size={12} className="text-text-dim" />
      </span>
    )
  return <Minus size={14} className="text-text-dim mx-auto" />
}

export default function Comparison() {
  return (
    <section className="py-28">
      <div className="max-w-4xl mx-auto px-6">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          className="text-center mb-16"
        >
          <div className="inline-block text-xs font-mono text-purple-DEFAULT border border-purple-glow/30 bg-purple-glow/5 rounded-full px-3 py-1 mb-4">
            comparison
          </div>
          <h2 className="text-4xl sm:text-5xl font-bold tracking-tight text-gradient-white mb-4">
            Not vs Git. Alongside Git.
          </h2>
          <p className="text-text-muted text-lg max-w-xl mx-auto">
            They solve different problems. Use git for history and collaboration.
            Use Drift for everything in between.
          </p>
        </motion.div>

        {/* Table */}
        <motion.div
          initial={{ opacity: 0, y: 24 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          className="rounded-2xl border border-border overflow-hidden"
        >
          {/* Column headers */}
          <div className="grid grid-cols-[1fr_100px_100px] bg-surface border-b border-border">
            <div className="px-6 py-4 text-sm text-text-muted">Feature</div>
            <div className="px-4 py-4 text-sm font-semibold text-text text-center">
              Git
            </div>
            <div className="px-4 py-4 text-sm font-semibold text-purple-bright text-center">
              Drift
            </div>
          </div>

          {ROWS.map((row, i) => (
            <div
              key={row.label}
              className={`grid grid-cols-[1fr_100px_100px] border-b border-border last:border-0 ${
                i % 2 === 0 ? 'bg-bg' : 'bg-surface/40'
              }`}
            >
              <div className="px-6 py-3.5 text-sm text-text-muted flex items-center">
                {row.label}
              </div>
              <div className="px-4 py-3.5 flex items-center justify-center">
                <Cell value={row.git} />
              </div>
              <div className="px-4 py-3.5 flex items-center justify-center">
                <Cell value={row.rw} />
              </div>
            </div>
          ))}
        </motion.div>

        {/* Note */}
        <motion.p
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true }}
          transition={{ delay: 0.3 }}
          className="text-center text-text-muted text-sm mt-6"
        >
          Drift is not a git replacement. Commit to git when your work is shareable.
          Save to Drift as you go.
        </motion.p>
      </div>
    </section>
  )
}
