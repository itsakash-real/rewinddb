import { motion } from 'framer-motion'

const COLUMNS = [
  {
    title: 'Done',
    color: 'text-[#38bdf8]/70',
    items: [
      'init, save, goto, undo, list, diff, status',
      'rw run with auto-rollback',
      'rw watch auto-save',
      'rw bisect',
      'rw session, search, stats, gc',
      'rw export / import',
      'rw upgrade self-update',
      'Content-addressable store with dedup',
      'DAG timeline with auto-branching',
      'Go SDK',
      'Shell completions',
      'rw health + repair',
      'Write-ahead log (WAL)',
      'Shell prompt integration',
      'rw stash / stash pop',
      'rw annotate, rw protect',
    ],
  },
  {
    title: 'In progress',
    color: 'text-warning/70',
    items: [
      'Human readable IDs (S1, S2, S3)',
      'WAL corruption prevention',
      'TUI checkpoint browser',
      'Semantic diff (tree-sitter AST)',
      'Block-level deduplication',
      'rw doctor diagnostic',
    ],
  },
  {
    title: 'Planned',
    color: 'text-secondary/70',
    items: [
      'Remote storage (S3 / R2 / NAS)',
      'P2P sync',
      'VS Code extension',
      'rw time "2 hours ago"',
      'CRDT conflict resolution',
      'Session replay',
      'Web UI for DAG visualization',
      'rw interactive TUI browser',
    ],
  },
]

export default function Roadmap() {
  return (
    <div className="min-h-screen pt-24 pb-20">
      <div className="max-w-content mx-auto px-6">
        <motion.div
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          className="mb-16"
        >
          <h1 className="text-[48px] font-bold text-text tracking-tight mb-2">Roadmap</h1>
          <p className="text-[18px] text-text-muted">What's built, what's building, what's next.</p>
        </motion.div>

        <div className="grid md:grid-cols-3 gap-8">
          {COLUMNS.map((col, ci) => (
            <motion.div
              key={col.title}
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: ci * 0.08 }}
            >
              <div className="flex items-center gap-2 mb-5">
                <span className={`text-[14px] font-mono font-medium uppercase tracking-wider ${col.color}`}>
                  {col.title}
                </span>
                <span className="text-[12px] text-text-muted font-mono">{col.items.length}</span>
              </div>

              <div className="space-y-1.5">
                {col.items.map((item, i) => (
                  <div
                    key={i}
                    className="py-2.5 px-4 rounded-md border border-border/50 text-[15px] text-[#b0b0b0] hover:border-[rgba(56,189,248,0.3)] hover:text-white transition-all duration-150"
                  >
                    {item}
                  </div>
                ))}
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </div>
  )
}
