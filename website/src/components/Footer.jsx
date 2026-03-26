import { Link } from 'react-router-dom'
import { LogoMark } from './Logo'

const GITHUB = 'https://github.com/itsakash-real/rewinddb'

export default function Footer() {
  return (
    <footer className="border-t border-border py-12">
      <div className="max-w-6xl mx-auto px-6">
        <div className="flex flex-col sm:flex-row items-center justify-between gap-6">
          {/* Logo */}
          <Link to="/" className="flex items-center gap-2.5">
            <LogoMark className="w-7 h-7" />
            <span className="font-semibold text-text">Drift</span>
          </Link>

          {/* Links */}
          <nav className="flex flex-wrap justify-center gap-x-6 gap-y-2 text-sm text-text-muted">
            <Link to="/" className="hover:text-text transition-colors">Home</Link>
            <Link to="/docs" className="hover:text-text transition-colors">Docs</Link>
            <Link to="/install" className="hover:text-text transition-colors">Install</Link>
            <a href={GITHUB} target="_blank" rel="noopener noreferrer" className="hover:text-text transition-colors">GitHub</a>
            <a href={`${GITHUB}/blob/main/LICENSE`} target="_blank" rel="noopener noreferrer" className="hover:text-text transition-colors">MIT License</a>
          </nav>

          {/* Right */}
          <div className="text-sm text-text-muted text-center sm:text-right">
            <div>Built in Go · Open Source</div>
            <div className="mt-0.5 text-xs text-text-dim">
              &copy; {new Date().getFullYear()} Drift
            </div>
          </div>
        </div>
      </div>
    </footer>
  )
}
