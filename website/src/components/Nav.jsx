import { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { AnimatePresence, motion } from 'framer-motion'
import { Menu, X } from 'lucide-react'

import { Github, Star } from 'lucide-react'

const GITHUB_REPO = 'https://github.com/itsakash-real/nimbi'
const GITHUB_PROFILE = 'https://github.com/itsakash-real'

export default function Nav() {
  const [scrolled, setScrolled] = useState(false)
  const [open, setOpen] = useState(false)
  const { pathname } = useLocation()

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 10)
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  useEffect(() => setOpen(false), [pathname])

  const links = [
    { to: '/docs', label: 'Docs' },
    { to: '/install', label: 'Install' },
    { to: '/changelog', label: 'Changelog' },
    { to: '/roadmap', label: 'Roadmap' },
  ]

  return (
    <header
      className={`fixed top-0 left-0 right-0 z-50 transition-all duration-300 ${
        scrolled ? 'bg-[#060d14]/70 backdrop-blur-2xl border-b border-[rgba(255,255,255,0.05)] shadow-[0_4px_30px_rgba(0,0,0,0.1)]' : 'bg-transparent'
      }`}
    >
      <div className="max-w-4xl mx-auto px-6 h-14 flex items-center justify-between">
        <Link to="/" className="flex items-center gap-2 group">
          <img src="/logo.svg" alt="Nimbi" className="w-7 h-7" />
          <span className="text-sm font-semibold text-text tracking-tight">Nimbi</span>
        </Link>

        <nav className="hidden md:flex items-center gap-1">
          {links.map((l) => (
            <Link
              key={l.to}
              to={l.to}
              className={`px-3 py-1.5 rounded-md text-[13px] transition-colors ${
                pathname === l.to ? 'text-text' : 'text-text-muted hover:text-text-secondary'
              }`}
            >
              {l.label}
            </Link>
          ))}
          {/* GitHub Profile */}
          <a
            href={GITHUB_PROFILE}
            target="_blank"
            rel="noopener noreferrer"
            className="ml-2 flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[13px] text-text-muted hover:text-text hover:bg-[rgba(255,255,255,0.05)] transition-all duration-200"
          >
            <Github size={14} /> Profile
          </a>

          {/* GitHub Repo Star Button Premium */}
          <a
            href={GITHUB_REPO}
            target="_blank"
            rel="noopener noreferrer"
            className="ml-2 flex items-center gap-2 px-3.5 py-1.5 rounded-md text-[13px] font-medium text-white bg-[rgba(255,255,255,0.05)] border border-[rgba(255,255,255,0.1)] hover:border-[rgba(56,189,248,0.5)] hover:bg-[rgba(56,189,248,0.1)] hover:shadow-glow-sm transition-all duration-300"
          >
            <Star size={14} className="text-[#fbbf24] fill-[#fbbf24]/20" /> Star
          </a>
        </nav>

        <button
          onClick={() => setOpen(!open)}
          className="md:hidden p-2 text-text-muted"
          aria-label="Menu"
        >
          {open ? <X size={18} /> : <Menu size={18} />}
        </button>
      </div>

      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="md:hidden border-t border-border bg-bg/95 backdrop-blur-xl"
          >
            <div className="px-6 py-3 flex flex-col gap-1">
              {links.map((l) => (
                <Link key={l.to} to={l.to} className="py-2 text-sm text-text-muted hover:text-text">
                  {l.label}
                </Link>
              ))}
              {/* Mobile Profile & Star */}
              <div className="flex flex-col gap-2 border-t border-[rgba(255,255,255,0.05)] mt-1 pt-3">
                <a href={GITHUB_PROFILE} target="_blank" rel="noopener noreferrer" className="flex items-center gap-2 py-2 text-sm text-text-muted hover:text-text">
                  <Github size={16} /> itsakash-real
                </a>
                <a href={GITHUB_REPO} target="_blank" rel="noopener noreferrer" className="flex items-center justify-center gap-2 py-2.5 mt-2 rounded-lg text-sm font-medium text-black bg-white hover:bg-gray-100 transition-colors">
                  <Star size={16} className="text-[#fbbf24] fill-[#fbbf24]" /> Star on GitHub
                </a>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </header>
  )
}
