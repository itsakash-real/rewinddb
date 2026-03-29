const STATS = [
  { value: '~180ms', label: 'Save 1,000 files' },
  { value: '~40ms', label: 'Restore' },
  { value: '~60ms', label: 'Status check' },
  { value: '~90ms', label: 'Garbage collect' },
]

export default function PerformanceBar() {
  return (
    <section className="py-20">
      <div className="max-w-content mx-auto px-6">
        <div className="flex items-center gap-3 mb-8">
          <h2 className="text-sm font-medium text-text-muted">Performance</h2>
          <div className="h-px flex-1 bg-border" />
        </div>

        <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
          {STATS.map((s) => (
            <div key={s.value}>
              <div className="text-2xl font-bold font-mono text-text tracking-tight mb-1">{s.value}</div>
              <div className="text-xs text-text-muted">{s.label}</div>
            </div>
          ))}
        </div>

        <p className="text-[11px] text-text-muted mt-6">
          Apple M2. Varies by disk speed and file sizes.
        </p>
      </div>
    </section>
  )
}
