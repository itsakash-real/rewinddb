import { useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { Copy, Check, ArrowRight } from 'lucide-react'

const GITHUB = 'https://github.com/itsakash-real/rewinddb'
const INSTALL_CMD = 'go install github.com/itsakash-real/rewinddb/cmd/rw@latest'

export default function FinalCTA() {
  const [copied, setCopied] = useState(false)

  const copy = async () => {
    await navigator.clipboard.writeText(INSTALL_CMD)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <section className="py-32 relative overflow-hidden">
      {/* Glow */}
      <div className="absolute inset-0 bg-hero-glow pointer-events-none" />
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[300px] bg-purple-glow/10 rounded-full blur-3xl pointer-events-none" />

      <div className="relative max-w-3xl mx-auto px-6 text-center">
        <motion.div
          initial={{ opacity: 0, y: 24 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
        >
          <div className="inline-block text-xs font-mono text-purple-DEFAULT border border-purple-glow/30 bg-purple-glow/5 rounded-full px-3 py-1 mb-6">
            get started now
          </div>

          <h2 className="text-5xl sm:text-6xl font-bold tracking-tight mb-6">
            <span className="text-gradient-white">Stop losing</span>
            <br />
            <span className="text-gradient">working states.</span>
          </h2>

          <p className="text-text-muted text-lg mb-10 max-w-lg mx-auto">
            Install takes 10 seconds. No account. No config. Just drop the binary in your PATH
            and run <code className="font-mono text-purple-bright">rw init</code> in any project.
          </p>

          {/* Install block */}
          <div className="flex items-center gap-0 rounded-xl border border-border bg-surface overflow-hidden mb-6 max-w-xl mx-auto">
            <div className="flex-1 px-4 py-3.5 font-mono text-sm text-text-muted text-left overflow-hidden">
              <span className="text-purple-DEFAULT mr-2">$</span>
              <span className="truncate">{INSTALL_CMD}</span>
            </div>
            <button
              onClick={copy}
              className="px-4 py-3.5 border-l border-border hover:bg-border text-text-muted hover:text-text transition-colors shrink-0"
              title="Copy"
            >
              {copied ? (
                <Check size={15} className="text-green" />
              ) : (
                <Copy size={15} />
              )}
            </button>
          </div>

          {/* Platform links */}
          <div className="flex flex-wrap justify-center gap-2 text-sm text-text-muted mb-10">
            <Link to="/install" className="hover:text-text transition-colors underline underline-offset-4">
              Homebrew
            </Link>
            <span>·</span>
            <Link to="/install" className="hover:text-text transition-colors underline underline-offset-4">
              Linux curl
            </Link>
            <span>·</span>
            <Link to="/install" className="hover:text-text transition-colors underline underline-offset-4">
              Windows .exe
            </Link>
            <span>·</span>
            <Link to="/install" className="hover:text-text transition-colors underline underline-offset-4">
              Build from source
            </Link>
          </div>

          {/* CTA buttons */}
          <div className="flex flex-wrap justify-center gap-3">
            <Link
              to="/install"
              className="group inline-flex items-center gap-2 px-7 py-3.5 rounded-xl bg-purple-glow hover:bg-purple-DEFAULT text-white font-semibold transition-all glow-purple"
            >
              Full install guide
              <ArrowRight size={16} className="group-hover:translate-x-0.5 transition-transform" />
            </Link>
            <a
              href={GITHUB}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-7 py-3.5 rounded-xl bg-surface hover:bg-border border border-border hover:border-border-light text-text font-semibold transition-all"
            >
              <GitHubIcon className="w-4 h-4" />
              View source
            </a>
          </div>
        </motion.div>
      </div>
    </section>
  )
}

function GitHubIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z" />
    </svg>
  )
}
