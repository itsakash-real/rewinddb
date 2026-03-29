import { Link } from 'react-router-dom'

export default function Footer() {
  return (
    <footer className="border-t border-border/50">
      <div className="max-w-4xl mx-auto px-6 py-10">
        <div className="flex flex-col items-center gap-4">
          <nav className="flex flex-wrap items-center justify-center gap-4 text-[14px] text-text-muted">
            <Link to="/docs" className="hover:text-[#38bdf8] hover:underline transition-colors">Docs</Link>
            <span className="text-border">&middot;</span>
            <Link to="/install" className="hover:text-[#38bdf8] hover:underline transition-colors">Install</Link>
            <span className="text-border">&middot;</span>
            <Link to="/changelog" className="hover:text-[#38bdf8] hover:underline transition-colors">Changelog</Link>
            <span className="text-border">&middot;</span>
            <Link to="/roadmap" className="hover:text-[#38bdf8] hover:underline transition-colors">Roadmap</Link>
            <span className="text-border">&middot;</span>
            <a href="https://github.com/itsakash-real/nimbi" target="_blank" rel="noopener noreferrer" className="hover:text-[#38bdf8] hover:underline transition-colors">GitHub</a>
          </nav>

          <p className="text-[12px] text-text-muted text-center">
            Built by{' '}
            <a
              href="https://github.com/itsakash-real"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-[#38bdf8] transition-colors"
            >
              Akash
            </a>
            . Free and open source. MIT License.
          </p>

          <p className="text-[10px] text-text-muted/50">
            &copy; {new Date().getFullYear()} Nimbi
          </p>
        </div>
      </div>
    </footer>
  )
}
