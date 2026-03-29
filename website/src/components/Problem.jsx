export default function Problem() {
  return (
    <section className="py-32">
      <div className="max-w-content mx-auto px-6">
        <div className="grid lg:grid-cols-2 gap-16 items-start">
          {/* Left — the story */}
          <div>
            <p className="text-[11px] font-mono text-text-muted uppercase tracking-[0.2em] mb-6">
              Why this exists
            </p>
            <blockquote className="text-[clamp(1.3rem,3vw,1.7rem)] text-text leading-[1.5] tracking-[-0.01em] font-light">
              <p className="mb-6">
                You're experimenting. Something works.
                <br />
                You change one thing.
                <br />
                <span className="text-text-secondary">Now nothing works and you can't get back.</span>
              </p>
              <p className="mb-6">
                Git commits felt too heavy mid-experiment.
                <br />
                I didn't want a commit.
                <br />
                <span className="text-text-secondary">I wanted a quicksave.</span>
              </p>
              <p className="text-accent/80">So I built this.</p>
            </blockquote>
            <p className="text-sm text-text-muted mt-8">
              &mdash; Akash
            </p>
          </div>

          {/* Right — pain points */}
          <div className="space-y-10 lg:pt-16">
            {[
              {
                label: 'AI-generated code broke everything',
                detail: 'rw save before pasting. rw undo if it breaks. One command back to working.',
              },
              {
                label: 'Lost a working state mid-refactor',
                detail: 'rw undo --n 3 takes you back 3 checkpoints. No ID needed.',
              },
              {
                label: 'Build script corrupted your project',
                detail: 'rw run wraps any command. Fails? Automatically rolled back.',
              },
            ].map((p, i) => (
              <div key={p.label}>
                <h3 className="text-sm font-medium text-text mb-1.5">{p.label}</h3>
                <p className="text-sm text-text-muted leading-relaxed">{p.detail}</p>
                {i < 2 && <div className="h-px bg-border mt-10" />}
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
