import { motion } from 'framer-motion'

const RELEASES = [
  {
    version: 'v0.2.0',
    date: 'March 2026',
    changes: [
      { type: 'NEW', text: 'Human-readable checkpoint IDs (S1, S2, S3)' },
      { type: 'NEW', text: 'rw health \u2014 full repository integrity check' },
      { type: 'NEW', text: 'rw repair \u2014 auto-recover corrupt objects' },
      { type: 'NEW', text: 'Write-ahead log (WAL)' },
      { type: 'NEW', text: 'rw timeline --visual' },
      { type: 'NEW', text: 'rw stash / rw stash pop' },
      { type: 'NEW', text: 'rw annotate \u2014 notes on checkpoints' },
      { type: 'NEW', text: 'rw protect \u2014 protect files from overwrites' },
      { type: 'NEW', text: 'Shell prompt integration' },
      { type: 'FIX', text: 'rw watch no longer exits on start' },
      { type: 'FIX', text: 'Debug logs hidden by default' },
      { type: 'FIX', text: 'ANSI colors on Windows CMD' },
      { type: 'FIX', text: 'Readable auto-branch names' },
      { type: 'FIX', text: 'rw run rollback returns to original branch' },
      { type: 'IMPROVED', text: '3x faster saves (parallel pipeline)' },
      { type: 'IMPROVED', text: 'rw list --all shows HEAD on all branches' },
    ],
  },
  {
    version: 'v0.1.0',
    date: 'February 2026',
    changes: [
      { type: 'NEW', text: 'Initial release' },
      { type: 'NEW', text: 'rw init, save, goto, undo, list, diff, status' },
      { type: 'NEW', text: 'rw run with auto-checkpoint and rollback' },
      { type: 'NEW', text: 'rw watch auto-save daemon' },
      { type: 'NEW', text: 'rw bisect, session, search, stats, gc' },
      { type: 'NEW', text: 'rw export / import' },
      { type: 'NEW', text: 'rw upgrade with background check' },
      { type: 'NEW', text: 'Content-addressable object store' },
      { type: 'NEW', text: 'DAG timeline with auto-branching' },
      { type: 'NEW', text: 'Go SDK' },
      { type: 'NEW', text: 'Shell completions (bash, zsh, fish, PowerShell)' },
    ],
  },
]

const TYPE_COLORS = {
  NEW: 'text-[#38bdf8]/80',
  FIX: 'text-error/70',
  IMPROVED: 'text-secondary/80',
}

export default function Changelog() {
  return (
    <div className="min-h-screen pt-24 pb-20">
      <div className="max-w-2xl mx-auto px-6">
        <motion.div
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          className="mb-16"
        >
          <h1 className="text-[48px] font-bold text-text tracking-tight mb-2">Changelog</h1>
          <p className="text-[18px] text-text-muted">What's new.</p>
        </motion.div>

        <div className="space-y-16">
          {RELEASES.map((release, ri) => (
            <motion.div
              key={release.version}
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: ri * 0.08 }}
            >
              <div className="flex items-baseline gap-3 mb-6">
                <span className="text-[22px] font-bold font-mono text-text">{release.version}</span>
                <span className="text-[14px] text-text-muted">{release.date}</span>
              </div>

              <div className="space-y-2">
                {release.changes.map((c, i) => (
                  <div key={i} className="flex items-baseline gap-3 py-1">
                    <span className={`shrink-0 text-[12px] font-mono font-medium w-20 ${TYPE_COLORS[c.type]}`}>
                      {c.type}
                    </span>
                    <span className="text-[15px] text-[#b0b0b0]">{c.text}</span>
                  </div>
                ))}
              </div>

              {ri < RELEASES.length - 1 && <div className="h-px bg-border mt-10" />}
            </motion.div>
          ))}
        </div>
      </div>
    </div>
  )
}
