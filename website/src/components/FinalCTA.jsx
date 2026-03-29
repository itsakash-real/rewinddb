import { motion } from 'framer-motion'
import { Github, BookOpen, Map, ArrowRight } from 'lucide-react'
import { Link } from 'react-router-dom'

const LINKS = [
  { icon: Github, label: 'GitHub', desc: 'View the source', href: 'https://github.com/itsakash-real/nimbi', external: true },
  { icon: BookOpen, label: 'Documentation', desc: 'Learn the ropes', to: '/docs' },
  { icon: Map, label: 'Roadmap', desc: "What's next", to: '/roadmap' },
]

const cardClass = "group flex flex-col items-center text-center p-8 rounded-xl border border-[rgba(255,255,255,0.08)] bg-[rgba(255,255,255,0.02)] hover:border-[rgba(56,189,248,0.4)] hover:bg-[rgba(56,189,248,0.05)] hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgba(56,189,248,0.1)] transition-all duration-200"

export default function FinalCTA() {
  return (
    <section className="py-24">
      <div className="max-w-4xl mx-auto px-6">
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true, margin: '-50px' }}
          transition={{ duration: 0.4 }}
          className="grid md:grid-cols-3 gap-4 mb-20 scroll-reveal"
        >
          {LINKS.map((l) => {
            const inner = (
              <>
                <l.icon size={32} className="text-[#38bdf8] mb-3" />
                <h3 className="text-[18px] font-semibold text-white mb-1">{l.label}</h3>
                <p className="text-[14px] text-[#94a3b8] mb-2">{l.desc}</p>
                <ArrowRight size={16} className="text-[#38bdf8] opacity-0 group-hover:opacity-100 translate-x-0 group-hover:translate-x-1 transition-all duration-200" />
              </>
            )
            return l.external ? (
              <a
                key={l.label}
                href={l.href}
                target="_blank"
                rel="noopener noreferrer"
                className={cardClass}
              >
                {inner}
              </a>
            ) : (
              <Link
                key={l.label}
                to={l.to}
                className={cardClass}
              >
                {inner}
              </Link>
            )
          })}
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 10 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true, margin: '-50px' }}
          transition={{ duration: 0.4 }}
          className="text-center rounded-2xl border border-border bg-surface/50 py-16 px-6 scroll-reveal"
        >
          <img
            src="/logo.png"
            alt="Nimbi"
            className="logo-hero w-[80px] h-[80px] mx-auto mb-5 drop-shadow-[0_0_20px_rgba(56,189,248,0.25)]"
            style={{ mixBlendMode: 'screen' }}
          />
          <h2 className="text-[28px] md:text-[48px] font-bold tracking-[-0.02em] text-text mb-3">
            Stop losing working states.
          </h2>
          <p className="text-[18px] text-[#94a3b8] mb-8 max-w-md mx-auto">
            Install takes 10 seconds. No account. No config. Just save and go back.
          </p>
          <Link
            to="/docs"
            className="cta-primary group inline-flex items-center gap-2 px-7 py-3.5 rounded-lg text-[16px] font-semibold text-white transition-all duration-150 active:scale-[0.98]"
          >
            Get started
            <ArrowRight size={16} className="group-hover:translate-x-0.5 transition-transform" />
          </Link>
        </motion.div>
      </div>
    </section>
  )
}
