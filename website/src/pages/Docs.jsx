import { useState, useEffect, useRef } from 'react'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'
import { Copy, Check, Menu, X } from 'lucide-react'

const NAV = [
  { id: 'getting-started', label: 'Getting started' },
  { id: 'core-commands', label: 'Core commands' },
  { id: 'advanced', label: 'Advanced features' },
  { id: 'sdk', label: 'Go SDK' },
  { id: 'ignore', label: '.rewindignore' },
  { id: 'faq', label: 'FAQ' },
]

function Section({ id, title, children }) {
  return (
    <section id={id} className="scroll-mt-24 mb-16">
      <h2 className="text-2xl font-bold text-text mb-6 pb-4 border-b border-border">{title}</h2>
      <div className="space-y-4 text-text-muted leading-relaxed">{children}</div>
    </section>
  )
}

function Cmd({ code, desc }) {
  return (
    <div className="flex flex-col sm:flex-row gap-3 p-4 rounded-xl border border-border bg-surface">
      <code className="font-mono text-sm text-purple-bright shrink-0 sm:w-64">{code}</code>
      <span className="text-sm text-text-muted">{desc}</span>
    </div>
  )
}

function Code({ children }) {
  const [copied, setCopied] = useState(false)
  const text = typeof children === 'string' ? children : ''

  const copy = async () => {
    await navigator.clipboard.writeText(text.trim())
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="relative group my-4">
      <pre className="p-4 pr-12 rounded-xl border border-border bg-[#080810] font-mono text-sm text-text overflow-x-auto">
        {children}
      </pre>
      <button
        onClick={copy}
        className="absolute top-3 right-3 p-1.5 rounded-lg border border-border bg-surface opacity-0 group-hover:opacity-100 text-text-muted hover:text-text transition-all"
        title="Copy"
      >
        {copied ? <Check size={13} className="text-green" /> : <Copy size={13} />}
      </button>
    </div>
  )
}

export default function Docs() {
  const [active, setActive] = useState('getting-started')
  const [mobileOpen, setMobileOpen] = useState(false)
  const observerRef = useRef(null)

  // Scroll-spy via IntersectionObserver
  useEffect(() => {
    const sections = NAV.map((n) => document.getElementById(n.id)).filter(Boolean)

    observerRef.current = new IntersectionObserver(
      (entries) => {
        const visible = entries.filter((e) => e.isIntersecting)
        if (visible.length > 0) {
          // Pick the topmost visible section
          const topmost = visible.reduce((a, b) =>
            a.boundingClientRect.top < b.boundingClientRect.top ? a : b
          )
          setActive(topmost.target.id)
        }
      },
      { rootMargin: '-80px 0px -60% 0px', threshold: 0 }
    )

    sections.forEach((s) => observerRef.current.observe(s))
    return () => observerRef.current?.disconnect()
  }, [])

  const scrollTo = (id) => {
    setActive(id)
    setMobileOpen(false)
    document.getElementById(id)?.scrollIntoView({ behavior: 'instant' })
  }

  return (
    <div className="min-h-screen pt-24 pb-20">
      <div className="max-w-6xl mx-auto px-6">

        {/* Mobile TOC toggle */}
        <div className="lg:hidden mb-6">
          <button
            onClick={() => setMobileOpen(!mobileOpen)}
            className="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-border bg-surface text-sm text-text-muted hover:text-text transition-colors"
          >
            {mobileOpen ? <X size={15} /> : <Menu size={15} />}
            {mobileOpen ? 'Close' : 'On this page'}
          </button>
          {mobileOpen && (
            <div className="mt-2 p-3 rounded-xl border border-border bg-surface space-y-0.5">
              {NAV.map((n) => (
                <button
                  key={n.id}
                  onClick={() => scrollTo(n.id)}
                  className={`block w-full text-left px-3 py-2 rounded-lg text-sm transition-colors ${
                    active === n.id
                      ? 'bg-purple-glow/10 text-purple-bright'
                      : 'text-text-muted hover:text-text hover:bg-surface/50'
                  }`}
                >
                  {n.label}
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="lg:grid lg:grid-cols-[220px_1fr] lg:gap-12">
          {/* Sidebar */}
          <aside className="hidden lg:block">
            <div className="sticky top-24 space-y-0.5">
              <p className="text-xs font-mono text-text-muted uppercase tracking-widest mb-4 px-3">
                Documentation
              </p>
              {NAV.map((n) => (
                <button
                  key={n.id}
                  onClick={() => scrollTo(n.id)}
                  className={`block w-full text-left px-3 py-2 rounded-lg text-sm transition-colors ${
                    active === n.id
                      ? 'bg-purple-glow/10 border border-purple-glow/20 text-purple-bright'
                      : 'text-text-muted hover:text-text hover:bg-surface/50'
                  }`}
                >
                  {n.label}
                </button>
              ))}
            </div>
          </aside>

          {/* Content */}
          <motion.div
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4 }}
          >
            <h1 className="text-4xl font-bold text-gradient-white mb-2">Documentation</h1>
            <p className="text-text-muted text-lg mb-12">Everything you need to use Drift effectively.</p>

            <Section id="getting-started" title="Getting started">
              <p>Install Drift (<Link to="/install" className="text-purple-DEFAULT underline underline-offset-2">full install guide</Link>), then run these three commands in any project:</p>
              <Code>{`cd my-project
rw init                          # set up Drift (run once)
rw save "everything is working"  # save a checkpoint
rw undo                          # go back if something breaks`}</Code>
              <p>That's the core loop. Everything else is optional.</p>
            </Section>

            <Section id="core-commands" title="Core commands">
              <div className="space-y-2">
                <Cmd code="rw init" desc="Initialize Drift in the current directory. Creates .rewind/ folder." />
                <Cmd code="rw save [message]" desc="Save a checkpoint. Message is optional — auto-generated if omitted." />
                <Cmd code="rw list" desc="List checkpoints on the current branch." />
                <Cmd code="rw list --all" desc="List checkpoints on all branches." />
                <Cmd code="rw goto <id>" desc="Restore to any checkpoint by ID, tag name, or HEAD~N." />
                <Cmd code="rw undo [--n N]" desc="Go back N checkpoints (default 1). No ID needed." />
                <Cmd code="rw diff <id1> <id2>" desc="Show file-level diff between two checkpoints." />
                <Cmd code="rw status" desc="Show current branch, HEAD, and what's changed on disk." />
                <Cmd code="rw gc" desc="Remove objects no checkpoint references. Use --dry-run to preview." />
              </div>
            </Section>

            <Section id="advanced" title="Advanced features">
              <h3 className="text-lg font-semibold text-text mt-6 mb-3">rw run — Safe command execution</h3>
              <p>Checkpoints before running a command. Rolls back automatically if it fails:</p>
              <Code>{`rw run "npm run build"
rw run "python migrate.py"
rw run "cargo test"`}</Code>

              <h3 className="text-lg font-semibold text-text mt-8 mb-3">rw watch — Auto-save daemon</h3>
              <p>Watches for file changes and auto-saves in the background:</p>
              <Code>{`rw watch               # default: 30s debounce
rw watch --interval 5m # save at most every 5 minutes
rw watch --quiet       # suppress per-save output`}</Code>

              <h3 className="text-lg font-semibold text-text mt-8 mb-3">rw bisect — Find when a bug appeared</h3>
              <Code>{`rw bisect start
rw bisect good <last-known-good-id>
rw bisect bad HEAD
# test code at each midpoint, then:
rw bisect good    # or: rw bisect bad
# repeat until it finds the exact checkpoint
rw bisect reset`}</Code>

              <h3 className="text-lg font-semibold text-text mt-8 mb-3">rw session — Group work into sessions</h3>
              <Code>{`rw session start "feature: dark mode"
# ... work, save checkpoints freely ...
rw session end
rw session list
rw session restore "feature: dark mode"  # jump back to start`}</Code>

              <h3 className="text-lg font-semibold text-text mt-8 mb-3">rw export / import — Share states</h3>
              <Code>{`rw export a3f2b1c8 --output bug-repro.rwdb
rw import bug-repro.rwdb`}</Code>
            </Section>

            <Section id="sdk" title="Go SDK">
              <p>Embed Drift in any Go application:</p>
              <Code>{`import "github.com/itsakash-real/rewinddb/internal/sdk"

client, err := sdk.New("/path/to/project")

// Save a checkpoint
id, err := client.Save("before processing payment")

// Restore by ID, tag, or relative ref
err = client.Goto("HEAD~2")

// Check working directory state
status, err := client.Status()
fmt.Printf("modified: %d\\n", len(status.ModifiedFiles))`}</Code>
            </Section>

            <Section id="ignore" title=".rewindignore">
              <p>Drift ignores <code className="font-mono text-xs text-purple-bright">node_modules/</code>, <code className="font-mono text-xs text-purple-bright">.git/</code>, and common build artifacts by default.</p>
              <p>Auto-detect patterns for your project type:</p>
              <Code>{`rw ignore auto   # detects Node, Python, Go, Rust, etc.`}</Code>
              <p>Add your own:</p>
              <Code>{`rw ignore add "dist/"
rw ignore add "*.log"
rw ignore add ".env.local"`}</Code>
              <p>Or edit <code className="font-mono text-xs text-purple-bright">.rewindignore</code> directly — same format as <code className="font-mono text-xs text-purple-bright">.gitignore</code> with <code className="font-mono text-xs text-purple-bright">**</code> glob support.</p>
            </Section>

            <Section id="faq" title="FAQ">
              {[
                { q: 'Does this replace git?', a: 'No. Git is for collaboration and shared history. Drift is for your local safety net between commits. Use both.' },
                { q: 'What does it actually store?', a: 'Every file is stored by its SHA-256 hash. If the same file appears in 10 checkpoints, it is stored once. Snapshots are gzip-compressed.' },
                { q: 'What if .rewind/ gets large?', a: 'Run rw gc to remove objects no checkpoint references. Run rw gc --dry-run first to see how much it will free.' },
                { q: 'Does it work in CI/CD?', a: 'Yes. rw save "pre-deploy: ${{ github.sha }}" works in GitHub Actions and any other CI system.' },
                { q: 'Does it need git to work?', a: 'No. Drift has no dependency on git.' },
                { q: 'Is my data safe?', a: 'Everything uses atomic writes (write temp file → fsync → rename). A crash mid-save leaves nothing corrupted. On startup, Drift auto-recovers any interrupted operations.' },
              ].map((item) => (
                <div key={item.q} className="p-4 rounded-xl border border-border bg-surface">
                  <h3 className="font-semibold text-text mb-1.5">{item.q}</h3>
                  <p className="text-sm">{item.a}</p>
                </div>
              ))}
            </Section>
          </motion.div>
        </div>
      </div>
    </div>
  )
}
