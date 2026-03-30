import { useState, useEffect, useRef } from 'react'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'
import { Copy, Check, Menu, X } from 'lucide-react'

const NAV = [
  { id: 'getting-started', label: 'Getting started' },
  { id: 'core-commands', label: 'Core commands' },
  { id: 'advanced', label: 'Advanced' },
  { id: 'sdk', label: 'Go SDK' },
  { id: 'ignore', label: '.rewindignore' },
  { id: 'faq', label: 'FAQ' },
]

function Section({ id, title, children }) {
  return (
    <section id={id} className="scroll-mt-24 mb-20">
      <h2 className="text-[28px] font-semibold text-text mb-8 pb-3 border-b border-border border-l-[3px] border-l-[#38bdf8] pl-4">{title}</h2>
      <div className="space-y-4 text-[#94a3b8] leading-[1.8] text-[16px]">{children}</div>
    </section>
  )
}

function Cmd({ code, desc }) {
  return (
    <div className="flex flex-col sm:flex-row gap-3 py-3 border-b border-border/50 last:border-0">
      <code className="font-mono text-[14px] text-[#38bdf8]/80 shrink-0 sm:w-56">{code}</code>
      <span className="text-[14px] text-[#94a3b8]">{desc}</span>
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
    <div className="relative group my-5 docs-code-block">
      <pre className="p-5 pr-12 font-mono text-[14px] text-text-secondary overflow-x-auto leading-[1.7]">
        {children}
      </pre>
      <button
        onClick={copy}
        className="absolute top-3 right-3 p-1.5 rounded-md border border-border bg-surface opacity-0 group-hover:opacity-100 text-text-muted hover:text-text transition-all"
        title="Copy"
      >
        {copied ? <Check size={12} className="text-success" /> : <Copy size={12} />}
      </button>
    </div>
  )
}

export default function Docs() {
  const [active, setActive] = useState('getting-started')
  const [mobileOpen, setMobileOpen] = useState(false)
  const observerRef = useRef(null)

  useEffect(() => {
    const sections = NAV.map((n) => document.getElementById(n.id)).filter(Boolean)
    observerRef.current = new IntersectionObserver(
      (entries) => {
        const visible = entries.filter((e) => e.isIntersecting)
        if (visible.length > 0) {
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
      <div className="max-w-content mx-auto px-6">

        {/* Mobile TOC */}
        <div className="lg:hidden mb-6">
          <button
            onClick={() => setMobileOpen(!mobileOpen)}
            className="flex items-center gap-2 text-[16px] text-text-muted hover:text-text transition-colors"
          >
            {mobileOpen ? <X size={14} /> : <Menu size={14} />}
            {mobileOpen ? 'Close' : 'On this page'}
          </button>
          {mobileOpen && (
            <div className="mt-3 space-y-0.5">
              {NAV.map((n) => (
                <button
                  key={n.id}
                  onClick={() => scrollTo(n.id)}
                  className={`block w-full text-left px-3 py-2.5 rounded text-[15px] transition-colors ${active === n.id ? 'text-[#38bdf8] bg-[rgba(56,189,248,0.1)] font-medium' : 'text-[#94a3b8] hover:text-white'
                    }`}
                >
                  {n.label}
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="lg:grid lg:grid-cols-[200px_1fr] lg:gap-16">
          {/* Sidebar */}
          <aside className="hidden lg:block">
            <div className="sticky top-24 space-y-1">
              {NAV.map((n) => (
                <button
                  key={n.id}
                  onClick={() => scrollTo(n.id)}
                  className={`block w-full text-left px-3 py-2.5 text-[14px] rounded transition-all duration-150 ${active === n.id
                      ? 'text-[#38bdf8] bg-[rgba(56,189,248,0.1)] border-l-2 border-l-[#38bdf8] font-medium'
                      : 'text-[#94a3b8] hover:text-white'
                    }`}
                >
                  {n.label}
                </button>
              ))}
            </div>
          </aside>

          {/* Content */}
          <motion.div
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
            className="max-w-2xl"
          >
            <h1 className="text-[48px] font-bold text-text mb-2 tracking-tight">Documentation</h1>
            <p className="text-[18px] text-text-muted mb-16">Everything you need to use Nimbi.</p>

            <Section id="getting-started" title="Getting started">
              <p>
                <Link to="/install" className="text-text-secondary underline underline-offset-2 hover:text-[#38bdf8] transition-colors">Install Nimbi</Link>, then:
              </p>
              <Code>{`cd my-project
rw init                          # set up (once per project)
rw save "everything is working"  # save a checkpoint
rw undo                          # go back if you break it`}</Code>
              <p>That's the core loop.</p>
            </Section>

            <Section id="core-commands" title="Core commands">
              <div>
                <Cmd code="rw init" desc="Initialize in the current directory." />
                <Cmd code="rw save [message]" desc="Checkpoint. Message auto-generated if omitted." />
                <Cmd code="rw list" desc="Show checkpoints on current branch." />
                <Cmd code="rw list --all" desc="Show all branches." />
                <Cmd code="rw goto <id>" desc="Restore by ID, tag, or HEAD~N." />
                <Cmd code="rw undo [--n N]" desc="Go back N steps. Default 1." />
                <Cmd code="rw diff <a> <b>" desc="File-level diff between checkpoints." />
                <Cmd code="rw status" desc="Current branch, HEAD, changed files." />
                <Cmd code='rw run "cmd"' desc="Run with auto-checkpoint and rollback." />
                <Cmd code="rw watch" desc="Auto-save daemon for background use." />
                <Cmd code="rw bisect" desc="Binary search to find when a bug appeared." />
                <Cmd code="rw tag <name>" desc="Tag current checkpoint." />
                <Cmd code="rw search <q>" desc="Search messages and tags." />
                <Cmd code="rw stats" desc="Timeline and storage stats." />
                <Cmd code="rw gc" desc="Remove unreferenced objects." />
                <Cmd code="rw upgrade" desc="Self-update." />
              </div>
            </Section>

            <Section id="advanced" title="Advanced">
              <h3 className="text-[20px] font-medium text-text mt-6 mb-3">rw run</h3>
              <p>Checkpoints before running. Rolls back on failure.</p>
              <Code>{`rw run "npm run build"
rw run "python migrate.py"
rw run "cargo test"`}</Code>

              <h3 className="text-[20px] font-medium text-text mt-10 mb-3">rw watch</h3>
              <p>Auto-saves when files change.</p>
              <Code>{`rw watch               # 30s debounce
rw watch --interval 5m # every 5 minutes
rw watch --quiet`}</Code>

              <h3 className="text-[20px] font-medium text-text mt-10 mb-3">rw bisect</h3>
              <Code>{`rw bisect start
rw bisect good <id>
rw bisect bad HEAD
# test, then: rw bisect good / bad
# repeat until found
rw bisect reset`}</Code>

              <h3 className="text-[20px] font-medium text-text mt-10 mb-3">rw session</h3>
              <Code>{`rw session start "feature: dark mode"
# work, save checkpoints...
rw session end
rw session list
rw session restore "feature: dark mode"`}</Code>

              <h3 className="text-[20px] font-medium text-text mt-10 mb-3">rw export / import</h3>
              <Code>{`rw export a3f2b1c8 --output bug-repro.rwdb
rw import bug-repro.rwdb`}</Code>
            </Section>

            <Section id="sdk" title="Go SDK">
              <p>Embed Nimbi in Go applications.</p>
              <Code>{`import "github.com/itsakash-real/nimbi/internal/sdk"

client, err := sdk.New("/path/to/project")

id, err := client.Save("before payment processing")
err = client.Goto("HEAD~2")

status, err := client.Status()
fmt.Printf("modified: %d\\n", len(status.ModifiedFiles))`}</Code>
            </Section>

            <Section id="ignore" title=".rewindignore">
              <p>
                <code className="font-mono text-[14px] text-[#38bdf8]/70">node_modules/</code>,{' '}
                <code className="font-mono text-[14px] text-[#38bdf8]/70">.git/</code>, and common build artifacts are ignored by default.
              </p>
              <Code>{`rw ignore auto          # auto-detect patterns
rw ignore add "dist/"
rw ignore add "*.log"`}</Code>
              <p>
                Or edit <code className="font-mono text-[14px] text-[#38bdf8]/70">.rewindignore</code> directly. Same syntax as <code className="font-mono text-[14px] text-[#38bdf8]/70">.gitignore</code>.
              </p>
            </Section>

            <Section id="faq" title="FAQ">
              <div className="space-y-6">
                {[
                  { q: 'Is this the same as git?', a: 'No. Git is for collaboration. Nimbi is your local safety net between commits.' },
                  { q: 'Does it slow my machine down?', a: 'No. Only runs when called. No daemon, no network. Saves in ~180ms.' },
                  { q: 'What does it store?', a: 'Files stored by SHA-256 hash. Same file in 10 checkpoints = stored once.' },
                  { q: 'What if I save with no message?', a: 'Auto-generates one based on changed files.' },
                  { q: 'Can I use it in CI/CD?', a: 'Yes. Works in GitHub Actions and any CI system.' },
                  { q: 'Does it need git?', a: 'No. Completely independent.' },
                  { q: 'What if .rewind/ gets corrupted?', a: 'rw health checks integrity. rw repair auto-fixes. WAL prevents corruption.' },
                  { q: 'Difference between undo and goto?', a: 'undo goes back N steps (no ID needed). goto jumps to a specific checkpoint.' },
                  { q: 'Storage usage?', a: 'Minimal. Content-addressable dedup. rw gc reclaims unused space.' },
                  { q: 'Windows support?', a: 'Yes. Download rw.exe. Colors work in Windows Terminal.' },
                ].map((item) => (
                  <div key={item.q}>
                    <h3 className="text-[18px] font-medium text-text mb-1">{item.q}</h3>
                    <p className="text-[16px] text-text-muted">{item.a}</p>
                  </div>
                ))}
              </div>
            </Section>
          </motion.div>
        </div>
      </div>
    </div>
  )
}
