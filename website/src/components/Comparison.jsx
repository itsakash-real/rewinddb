import { Check, X } from 'lucide-react'

const ROWS = [
  { label: 'Purpose',                        git: 'Share with team',    rw: 'Personal safety net' },
  { label: 'Requires commit message',        git: 'Yes, always',        rw: true },
  { label: 'Tracks node_modules / binaries', git: false,                rw: true },
  { label: 'Tracks .env files',              git: false,                rw: true },
  { label: 'Auto-branches on time-travel',   git: false,                rw: true },
  { label: 'Rollback on script failure',     git: false,                rw: 'rw run "cmd"' },
  { label: 'Works in CI/CD',                 git: true,                 rw: true },
  { label: 'Works without internet',         git: true,                 rw: 'Fully local' },
  { label: 'Collaboration / push / pull',    git: true,                 rw: 'coming' },
  { label: 'Speed for 1000 files',           git: 'Slow for binaries',  rw: '~180ms' },
]

function Cell({ value, isRw }) {
  if (value === true)
    return <Check size={12} className={isRw ? 'text-accent' : 'text-text-secondary'} />
  if (value === false)
    return <X size={12} className="text-text-muted/40" />
  if (value === 'coming')
    return <span className="text-[11px] font-mono text-warning/70">Soon</span>
  return (
    <span className={`text-[11px] font-mono ${isRw ? 'text-accent/80' : 'text-text-muted'}`}>
      {value}
    </span>
  )
}

export default function Comparison() {
  return (
    <section className="py-32">
      <div className="max-w-3xl mx-auto px-6">
        <div className="mb-12">
          <h2 className="text-[clamp(1.8rem,4vw,2.5rem)] font-bold tracking-[-0.03em] text-text mb-3">
            Git is for sharing. This is for the messy middle.
          </h2>
          <p className="text-text-muted text-sm">
            They work great together &mdash; most people use both.
          </p>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="text-left py-3 pr-4 text-text-muted font-normal text-xs uppercase tracking-wider">Feature</th>
                <th className="text-center py-3 px-4 text-text-muted font-normal text-xs uppercase tracking-wider w-28">Git</th>
                <th className="text-center py-3 pl-4 text-text font-medium text-xs uppercase tracking-wider w-28">Nimbi</th>
              </tr>
            </thead>
            <tbody>
              {ROWS.map((row) => (
                <tr key={row.label} className="border-b border-border/50 last:border-0">
                  <td className="py-3 pr-4 text-text-secondary">{row.label}</td>
                  <td className="py-3 px-4 text-center"><Cell value={row.git} isRw={false} /></td>
                  <td className="py-3 pl-4 text-center"><Cell value={row.rw} isRw={true} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  )
}
