import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { ArrowRight, Star } from 'lucide-react'
import TerminalDemo from './TerminalDemo'

const GITHUB = 'https://github.com/itsakash-real/rewinddb'

export default function Hero() {
  return (
    <section className="relative min-h-screen flex flex-col justify-center overflow-hidden bg-grid">
      {/* Glow blobs */}
      <div className="absolute inset-0 bg-hero-glow pointer-events-none" />
      <div className="absolute top-1/3 left-1/4 w-96 h-96 bg-purple-glow/5 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-1/4 right-1/4 w-64 h-64 bg-cyan-dim/5 rounded-full blur-3xl pointer-events-none" />

      <div className="relative max-w-6xl mx-auto px-6 pt-28 pb-20">
        <div className="grid lg:grid-cols-2 gap-12 lg:gap-16 items-center">
          {/* Left — copy */}
          <div>
            {/* Badge */}
            <motion.div
              initial={{ opacity: 0, y: 16 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5 }}
              className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-purple-glow/30 bg-purple-glow/10 text-purple-bright text-xs font-medium mb-6"
            >
              <span className="w-1.5 h-1.5 rounded-full bg-purple-bright animate-pulse" />
              Open source · MIT · Single binary
            </motion.div>

            {/* Headline */}
            <motion.h1
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.1 }}
              className="text-5xl sm:text-6xl lg:text-7xl font-bold leading-[1.05] tracking-tight mb-6"
            >
              <span className="text-gradient-white block">You broke</span>
              <span className="text-gradient-white block">your project.</span>
              <span className="text-gradient block mt-1">Again.</span>
            </motion.h1>

            {/* Sub */}
            <motion.p
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.2 }}
              className="text-lg text-text-muted leading-relaxed mb-8 max-w-lg"
            >
              Drift saves your <strong className="text-text">entire project folder</strong> as a
              checkpoint in one command. Go back to any point instantly.
              Works with every language, every stack — no setup required.
            </motion.p>

            {/* CTAs */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.3 }}
              className="flex flex-wrap gap-3 mb-10"
            >
              <Link
                to="/install"
                className="group inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-purple-glow hover:bg-purple-DEFAULT text-white font-semibold transition-all glow-purple hover:scale-[1.02]"
              >
                Install free
                <ArrowRight size={16} className="group-hover:translate-x-0.5 transition-transform" />
              </Link>
              <a
                href={GITHUB}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-surface hover:bg-border border border-border hover:border-border-light text-text font-semibold transition-all"
              >
                <Star size={16} className="text-yellow-300" />
                Star on GitHub
              </a>
            </motion.div>

            {/* Quick install */}
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.6, delay: 0.4 }}
              className="flex items-center gap-3"
            >
              <div className="flex items-center gap-2 px-4 py-2.5 rounded-lg bg-surface border border-border font-mono text-sm text-text-muted">
                <span className="text-purple-DEFAULT">$</span>
                <span>go install github.com/itsakash-real/rewinddb/cmd/rw@latest</span>
              </div>
            </motion.div>
          </div>

          {/* Right — terminal */}
          <motion.div
            initial={{ opacity: 0, x: 30 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.7, delay: 0.3 }}
            className="animate-float"
          >
            <TerminalDemo />
          </motion.div>
        </div>

        {/* Stats row */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.6 }}
          className="grid grid-cols-2 sm:grid-cols-4 gap-4 mt-20 pt-10 border-t border-border"
        >
          {[
            { value: '~180ms', label: 'to save 1000 files' },
            { value: '1 binary', label: 'no dependencies' },
            { value: 'any stack', label: 'React, Go, Python...' },
            { value: '100% local', label: 'your data stays yours' },
          ].map((s) => (
            <div key={s.label} className="text-center">
              <div className="text-2xl font-bold text-gradient mb-1">{s.value}</div>
              <div className="text-sm text-text-muted">{s.label}</div>
            </div>
          ))}
        </motion.div>
      </div>
    </section>
  )
}
