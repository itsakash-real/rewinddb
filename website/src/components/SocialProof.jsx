const ROW1 = [
  { handle: '@jakemorris_dev', text: 'AI kept breaking my code mid-session. rw run catches every failed build and rolls back automatically.' },
  { handle: '@sana_builds', text: 'This is what git should have been for solo devs. Binary files as first-class citizens.' },
  { handle: '@rustacean99', text: 'rw run is black magic. Failed build, auto rollback. Wrap every risky command in it.' },
  { handle: '@priya_codes', text: 'Set up in 2 minutes. rw save, rw undo. Already can\'t code without it.' },
  { handle: '@the_refactorer', text: 'Finally something that works on my Godot project. Git hates binary assets. RewindDB doesn\'t care.' },
  { handle: '@lucasm_eng', text: 'Auto-branching when you restore an old checkpoint is genius. Both timelines preserved.' },
]

const ROW2 = [
  { handle: '@0xdevguy', text: 'RewindDB + AI coding = absolute safety net. Break everything. Restore anything.' },
  { handle: '@amelia_fullstack', text: 'rw bisect found exactly which checkpoint introduced the bug. Like git bisect for your whole project.' },
  { handle: '@tobiaswerk', text: '1000 files saved in 180ms. Faster than thinking about whether to commit.' },
  { handle: '@nina_writes_code', text: 'rw watch in background means I never lose more than 30 seconds of work.' },
  { handle: '@danpham_io', text: 'Fully local, zero config, single binary. This is what CLI tools should feel like.' },
  { handle: '@kieran_dev', text: 'Built exactly for AI-assisted coding. The tool this moment in software needed.' },
]

function Card({ handle, text }) {
  return (
    <div className="shrink-0 w-[300px] px-5 py-4 rounded-lg border border-border hover:border-border-hover transition-colors duration-200">
      <span className="text-[11px] text-text-muted font-mono block mb-2">{handle}</span>
      <p className="text-[13px] text-text-secondary leading-relaxed">{text}</p>
    </div>
  )
}

function MarqueeRow({ items, direction = 'left' }) {
  const doubled = [...items, ...items]
  return (
    <div className="overflow-x-hidden marquee-container">
      <div
        className={`flex gap-4 w-max ${
          direction === 'left' ? 'animate-marquee-left' : 'animate-marquee-right'
        }`}
      >
        {doubled.map((item, i) => (
          <Card key={`${item.handle}-${i}`} {...item} />
        ))}
      </div>
    </div>
  )
}

export default function SocialProof() {
  return (
    <section className="py-20">
      <p className="text-center text-sm text-text-muted mb-8 tracking-wide">
        What developers are saying
      </p>
      <div className="space-y-4">
        <MarqueeRow items={ROW1} direction="left" />
        <MarqueeRow items={ROW2} direction="right" />
      </div>
    </section>
  )
}
