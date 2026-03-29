import { BrowserRouter, Routes, Route, useLocation, Link } from 'react-router-dom'
import { useEffect } from 'react'
import Home from './pages/Home'
import Docs from './pages/Docs'
import Install from './pages/Install'
import Changelog from './pages/Changelog'
import Roadmap from './pages/Roadmap'
import Footer from './components/Footer'
import SquaresBackground from './components/SquaresBackground'

function ScrollToTop() {
  const { pathname } = useLocation()
  useEffect(() => { window.scrollTo(0, 0) }, [pathname])
  return null
}

function BackArrow() {
  const { pathname } = useLocation()
  if (pathname === '/') return null
  return (
    <div className="fixed top-5 left-6 z-50">
      <Link
        to="/"
        className="text-[15px] text-[#64748b] hover:text-[#94a3b8] transition-colors no-underline"
      >
        &larr; Nimbi
      </Link>
    </div>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <ScrollToTop />
      <SquaresBackground />
      <BackArrow />
      <main>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/docs" element={<Docs />} />
          <Route path="/install" element={<Install />} />
          <Route path="/changelog" element={<Changelog />} />
          <Route path="/roadmap" element={<Roadmap />} />
        </Routes>
      </main>
      <Footer />
    </BrowserRouter>
  )
}
