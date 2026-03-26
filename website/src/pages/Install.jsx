import { useState } from 'react'
import { motion } from 'framer-motion'
import { Copy, Check } from 'lucide-react'

function CopyBlock({ code, lang = 'bash' }) {
  const [copied, setCopied] = useState(false)
  const copy = async () => {
    await navigator.clipboard.writeText(code)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }
  return (
    <div className="group relative flex items-stretch rounded-xl border border-border bg-surface overflow-hidden my-4">
      <div className="flex-1 p-4 font-mono text-sm text-text overflow-x-auto">
        <span className="text-purple-DEFAULT select-none mr-2">$</span>
        {code}
      </div>
      <button
        onClick={copy}
        className="px-4 border-l border-border text-text-muted hover:text-text hover:bg-border transition-colors shrink-0"
        title="Copy"
      >
        {copied ? <Check size={14} className="text-green" /> : <Copy size={14} />}
      </button>
    </div>
  )
}

const METHODS = [
  {
    id: 'go',
    label: 'Go install',
    badge: 'Recommended',
    platforms: 'Any platform',
    content: (
      <div>
        <p className="text-text-muted mb-4">Requires Go 1.22+. Installs the latest release directly to your <code className="font-mono text-purple-bright text-xs">$GOPATH/bin</code>.</p>
        <CopyBlock code="go install github.com/itsakash-real/rewinddb/cmd/rw@latest" />
        <CopyBlock code="rw version" />
      </div>
    ),
  },
  {
    id: 'brew',
    label: 'Homebrew',
    badge: null,
    platforms: 'macOS · Linux',
    content: (
      <div>
        <CopyBlock code="brew install itsakash-real/tap/rewinddb" />
        <CopyBlock code="rw version" />
        <p className="text-text-muted text-sm mt-3">Updates automatically with <code className="font-mono text-xs text-purple-bright">brew upgrade rewinddb</code>.</p>
      </div>
    ),
  },
  {
    id: 'curl',
    label: 'curl',
    badge: null,
    platforms: 'Linux · macOS',
    content: (
      <div>
        <CopyBlock code="curl -sSL https://raw.githubusercontent.com/itsakash-real/rewinddb/main/install.sh | bash" />
        <p className="text-text-muted text-sm mt-3">Installs to <code className="font-mono text-xs text-purple-bright">~/.local/bin</code> by default. Pass <code className="font-mono text-xs text-purple-bright">INSTALL_DIR=/usr/local/bin</code> to override.</p>
      </div>
    ),
  },
  {
    id: 'windows',
    label: 'Windows',
    badge: null,
    platforms: 'Windows',
    content: (
      <div>
        <p className="text-text-muted mb-4">
          Download <code className="font-mono text-xs text-purple-bright">rw.exe</code> from the{' '}
          <a href="https://github.com/itsakash-real/rewinddb/releases" target="_blank" rel="noopener noreferrer" className="text-purple-DEFAULT underline underline-offset-2">Releases page</a>,
          then add it to a folder in your <code className="font-mono text-xs text-purple-bright">PATH</code>.
        </p>
        <p className="text-text-muted text-sm">Colors require <strong className="text-text">Windows Terminal</strong> — not the old CMD prompt. PowerShell works too.</p>
        <div className="mt-4">
          <p className="text-text-muted text-sm mb-2">Or, if you have Go installed:</p>
          <CopyBlock code="go install github.com/itsakash-real/rewinddb/cmd/rw@latest" />
        </div>
      </div>
    ),
  },
  {
    id: 'source',
    label: 'From source',
    badge: null,
    platforms: 'Any platform',
    content: (
      <div>
        <CopyBlock code="git clone https://github.com/itsakash-real/rewinddb" />
        <CopyBlock code="cd rewinddb && make build" />
        <CopyBlock code="./rw version" />
        <p className="text-text-muted text-sm mt-3">Requires Go 1.22+ and make.</p>
      </div>
    ),
  },
]

export default function Install() {
  const [active, setActive] = useState('go')
  const method = METHODS.find((m) => m.id === active)

  return (
    <div className="min-h-screen pt-24 pb-20">
      <div className="max-w-3xl mx-auto px-6">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-center mb-12"
        >
          <h1 className="text-4xl sm:text-5xl font-bold tracking-tight text-gradient-white mb-4">
            Install Drift
          </h1>
          <p className="text-text-muted text-lg">
            Single binary. No config. Drop it in your PATH and go.
          </p>
        </motion.div>

        {/* Method tabs */}
        <div className="flex flex-wrap gap-2 mb-8">
          {METHODS.map((m) => (
            <button
              key={m.id}
              onClick={() => setActive(m.id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition-all ${
                active === m.id
                  ? 'bg-purple-glow text-white'
                  : 'bg-surface border border-border text-text-muted hover:text-text hover:border-border-light'
              }`}
            >
              {m.label}
              {m.badge && (
                <span className="text-[10px] bg-green-dim/20 text-green rounded px-1.5 py-0.5">
                  {m.badge}
                </span>
              )}
              <span className="text-[10px] text-text-dim">{m.platforms}</span>
            </button>
          ))}
        </div>

        {/* Active method content */}
        <motion.div
          key={active}
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.25 }}
          className="p-6 rounded-2xl border border-border bg-surface mb-10"
        >
          {method.content}
        </motion.div>

        {/* Verify */}
        <div className="p-6 rounded-2xl border border-border bg-surface mb-10">
          <h2 className="font-semibold text-text mb-4">Verify the install</h2>
          <CopyBlock code="rw version" />
          <p className="text-text-muted text-sm mt-3">
            Should print something like: <code className="font-mono text-xs text-purple-bright">Drift v1.0.0 (linux/amd64)</code>
          </p>
        </div>

        {/* First steps */}
        <div className="p-6 rounded-2xl border border-border bg-surface">
          <h2 className="font-semibold text-text mb-6">Your first 3 commands</h2>
          <div className="space-y-4">
            {[
              { step: '1', cmd: 'cd my-project && rw init', desc: 'Initialize Drift in any project folder.' },
              { step: '2', cmd: 'rw save "initial working state"', desc: 'Save your first checkpoint. Message is optional.' },
              { step: '3', cmd: 'rw undo', desc: 'If you break something, this puts it back.' },
            ].map((s) => (
              <div key={s.step} className="flex gap-4">
                <div className="w-7 h-7 rounded-lg bg-bg border border-border flex items-center justify-center text-xs font-mono text-text-muted shrink-0 mt-0.5">
                  {s.step}
                </div>
                <div>
                  <code className="block font-mono text-sm text-purple-bright mb-1">{s.cmd}</code>
                  <p className="text-text-muted text-sm">{s.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Keeping up to date */}
        <div className="mt-8 p-6 rounded-2xl border border-border bg-surface">
          <h2 className="font-semibold text-text mb-3">Keeping up to date</h2>
          <p className="text-text-muted text-sm mb-4">
            Run this any time to upgrade to the latest release:
          </p>
          <CopyBlock code="rw upgrade" />
          <p className="text-text-muted text-sm">
            Drift also quietly checks for updates once a day and shows a one-line notice if a newer version exists.
          </p>
        </div>
      </div>
    </div>
  )
}
