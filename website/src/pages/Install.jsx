import { useState } from 'react'
import { motion } from 'framer-motion'
import { Copy, Check } from 'lucide-react'

function CopyBlock({ code }) {
  const [copied, setCopied] = useState(false)
  const copy = async () => {
    await navigator.clipboard.writeText(code)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }
  return (
    <div className="group relative flex items-stretch rounded-lg border border-border bg-[#0d1117] overflow-hidden my-4 docs-code-block">
      <div className="flex-1 p-4 font-mono text-[15px] text-text-secondary overflow-x-auto">
        <span className="text-[#38bdf8] select-none mr-2">$</span>
        {code}
      </div>
      <button
        onClick={copy}
        className="px-4 border-l border-border text-text-muted hover:text-text hover:bg-white/[0.03] transition-colors shrink-0"
        title="Copy"
      >
        {copied ? <Check size={13} className="text-success" /> : <Copy size={13} />}
      </button>
    </div>
  )
}

const METHODS = [
  {
    id: 'brew',
    label: 'macOS',
    content: (
      <div>
        <CopyBlock code="brew install itsakash-real/nimbi/rw" />
        <CopyBlock code="rw version" />
      </div>
    ),
  },
  {
    id: 'curl',
    label: 'Linux',
    content: (
      <div>
        <CopyBlock code="curl -sSL https://raw.githubusercontent.com/itsakash-real/nimbi/main/install.sh | bash" />
        <CopyBlock code="rw version" />
      </div>
    ),
  },
  {
    id: 'windows',
    label: 'Windows',
    content: (
      <div>
        <p className="text-text-muted text-[16px] mb-4">
          Download <code className="font-mono text-[14px] text-text-secondary">rw.exe</code> from{' '}
          <a href="https://github.com/itsakash-real/nimbi/releases" target="_blank" rel="noopener noreferrer" className="underline underline-offset-2 hover:text-[#38bdf8] transition-colors">Releases</a>.
          Add to PATH. Colors require Windows Terminal.
        </p>
        <CopyBlock code="go install github.com/itsakash-real/nimbi/cmd/rw@latest" />
      </div>
    ),
  },
  {
    id: 'go',
    label: 'Go',
    content: (
      <div>
        <p className="text-text-muted text-[16px] mb-4">Requires Go 1.21+.</p>
        <CopyBlock code="go install github.com/itsakash-real/nimbi/cmd/rw@latest" />
      </div>
    ),
  },
  {
    id: 'source',
    label: 'Source',
    content: (
      <div>
        <CopyBlock code="git clone https://github.com/itsakash-real/nimbi && cd nimbi && make build" />
      </div>
    ),
  },
]

export default function Install() {
  const [active, setActive] = useState('brew')
  const method = METHODS.find((m) => m.id === active)

  return (
    <div className="min-h-screen pt-24 pb-20">
      <div className="max-w-2xl mx-auto px-6">
        <motion.div
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          className="mb-16"
        >
          <h1 className="text-[48px] font-bold text-text tracking-tight mb-2">Install</h1>
          <p className="text-[18px] text-text-muted">Single binary. No config.</p>
        </motion.div>

        <div className="flex gap-1 mb-6">
          {METHODS.map((m) => (
            <button
              key={m.id}
              onClick={() => setActive(m.id)}
              className={`px-3 py-1.5 rounded-md text-[15px] font-mono transition-all duration-150 ${
                active === m.id
                  ? 'bg-white/[0.06] text-text border border-border-hover'
                  : 'text-text-muted hover:text-text-secondary border border-transparent'
              }`}
            >
              {m.label}
            </button>
          ))}
        </div>

        <motion.div
          key={active}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.15 }}
          className="mb-16"
        >
          {method.content}
        </motion.div>

        <div className="mb-16">
          <h2 className="text-[22px] font-medium text-text mb-6">Then</h2>
          <div className="space-y-4">
            {[
              { cmd: 'rw init', desc: 'Initialize in any project.' },
              { cmd: 'rw save "working state"', desc: 'Save your first checkpoint.' },
              { cmd: 'rw undo', desc: 'Go back if you break it.' },
            ].map((s) => (
              <div key={s.cmd} className="flex items-baseline gap-4">
                <code className="font-mono text-[15px] text-[#38bdf8]/80 shrink-0 w-48">{s.cmd}</code>
                <span className="text-[15px] text-[#94a3b8]">{s.desc}</span>
              </div>
            ))}
          </div>
        </div>

        <div>
          <div className="flex items-center gap-3 mb-4">
            <h2 className="text-[18px] font-medium text-[#94a3b8]">Updates</h2>
            <div className="h-px flex-1 bg-border" />
          </div>
          <CopyBlock code="rw upgrade" />
          <p className="text-[14px] text-text-muted mt-2">
            Also checks silently once per day.
          </p>
        </div>
      </div>
    </div>
  )
}
