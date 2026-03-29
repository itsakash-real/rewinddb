import { useEffect, useRef } from 'react'

export default function SquaresBackground() {
  const canvasRef = useRef(null)

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const ctx = canvas.getContext('2d')
    const DPR = Math.min(devicePixelRatio, 2)
    let stars = []
    let t = 0
    let raf

    function resize() {
      canvas.width = window.innerWidth * DPR
      canvas.height = window.innerHeight * DPR
      canvas.style.width = window.innerWidth + 'px'
      canvas.style.height = window.innerHeight + 'px'
      ctx.setTransform(DPR, 0, 0, DPR, 0, 0)
      buildStars()
    }

    function buildStars() {
      stars = []
      for (let i = 0; i < 200; i++) {
        stars.push({
          x: Math.random() * window.innerWidth,
          y: Math.random() * window.innerHeight,
          r: Math.random() * 1.2 + 0.2,
          phase: Math.random() * Math.PI * 2,
          speed: Math.random() * 0.0008 + 0.0003,
        })
      }
    }

    function drawBg() {
      const W2 = window.innerWidth
      const H2 = window.innerHeight
      const g = ctx.createLinearGradient(0, 0, 0, H2)
      g.addColorStop(0, '#060d14')
      g.addColorStop(0.5, '#080c10')
      g.addColorStop(1, '#050a0f')
      ctx.fillStyle = g
      ctx.fillRect(0, 0, W2, H2)
    }

    function drawOrb(x, y, r, color) {
      const W2 = window.innerWidth
      const H2 = window.innerHeight
      const g = ctx.createRadialGradient(x, y, 0, x, y, r)
      g.addColorStop(0, color)
      g.addColorStop(1, 'rgba(0,0,0,0)')
      ctx.fillStyle = g
      ctx.fillRect(0, 0, W2, H2)
    }

    function drawDots() {
      const sp = 30
      const W2 = window.innerWidth
      const H2 = window.innerHeight
      const cx = W2 / 2
      const cy = H2 * 0.4
      const maxD = Math.sqrt(cx * cx + cy * cy)
      for (let x = 0; x < W2; x += sp) {
        for (let y = 0; y < H2; y += sp) {
          const dx = x - cx
          const dy = y - cy
          const dist = Math.sqrt(dx * dx + dy * dy)
          const a = Math.max(0, 0.07 * (1 - dist / maxD))
          ctx.beginPath()
          ctx.arc(x, y, 0.8, 0, Math.PI * 2)
          ctx.fillStyle = `rgba(255,255,255,${a})`
          ctx.fill()
        }
      }
    }

    function drawStars(t) {
      stars.forEach((s) => {
        const a = (Math.sin(t * s.speed + s.phase) + 1) / 2
        ctx.beginPath()
        ctx.arc(s.x, s.y, s.r, 0, Math.PI * 2)
        ctx.fillStyle = `rgba(180,220,255,${a * 0.7})`
        ctx.fill()
      })
    }

    function drawScanline(t) {
      const W2 = window.innerWidth
      const H2 = window.innerHeight
      const y = ((t * 0.04) % (H2 + 160)) - 80
      const g = ctx.createLinearGradient(0, y - 80, 0, y + 80)
      g.addColorStop(0, 'rgba(56,189,248,0)')
      g.addColorStop(0.5, 'rgba(56,189,248,0.018)')
      g.addColorStop(1, 'rgba(56,189,248,0)')
      ctx.fillStyle = g
      ctx.fillRect(0, y - 80, W2, 160)
    }

    function drawHorizon(t) {
      const W2 = window.innerWidth
      const H2 = window.innerHeight
      const pulse = 0.03 + Math.sin(t * 0.0008) * 0.015
      const g = ctx.createLinearGradient(0, H2 * 0.5, 0, H2 * 0.65)
      g.addColorStop(0, 'rgba(56,189,248,0)')
      g.addColorStop(0.5, `rgba(56,189,248,${pulse})`)
      g.addColorStop(1, 'rgba(56,189,248,0)')
      ctx.fillStyle = g
      ctx.fillRect(0, H2 * 0.5, W2, H2 * 0.15)
    }

    function drawFloatingParticles(t) {
      const W2 = window.innerWidth
      const H2 = window.innerHeight
      for (let i = 0; i < 12; i++) {
        const seed = i * 137.5
        const x = (Math.sin(seed) * 0.5 + 0.5) * W2
        const baseY = (Math.cos(seed * 0.7) * 0.5 + 0.5) * H2
        const y = baseY + Math.sin(t * 0.0005 + seed) * 30
        const size = 1.5 + Math.sin(seed * 0.3) * 0.8
        const alpha = 0.2 + Math.sin(t * 0.0006 + seed) * 0.15
        const glow = ctx.createRadialGradient(x, y, 0, x, y, size * 4)
        glow.addColorStop(0, `rgba(56,189,248,${alpha})`)
        glow.addColorStop(1, 'rgba(56,189,248,0)')
        ctx.fillStyle = glow
        ctx.fillRect(x - size * 4, y - size * 4, size * 8, size * 8)
        ctx.beginPath()
        ctx.arc(x, y, size, 0, Math.PI * 2)
        ctx.fillStyle = `rgba(180,230,255,${alpha + 0.1})`
        ctx.fill()
      }
    }

    function loop(ts) {
      t = ts
      const W2 = window.innerWidth
      const H2 = window.innerHeight

      ctx.clearRect(0, 0, W2, H2)
      drawBg()

      const ox1 = W2 * 0.12 + Math.sin(t * 0.0004) * W2 * 0.03
      const oy1 = H2 * 0.18 + Math.cos(t * 0.0003) * H2 * 0.04
      drawOrb(ox1, oy1, W2 * 0.42, 'rgba(56,189,248,0.08)')

      const ox2 = W2 * 0.88 + Math.cos(t * 0.0003) * W2 * 0.03
      const oy2 = H2 * 0.82 + Math.sin(t * 0.0004) * H2 * 0.03
      drawOrb(ox2, oy2, W2 * 0.38, 'rgba(99,102,241,0.07)')

      const ox3 = W2 * 0.5 + Math.sin(t * 0.0005) * W2 * 0.04
      const oy3 = H2 * 0.28 + Math.cos(t * 0.0004) * H2 * 0.03
      drawOrb(ox3, oy3, W2 * 0.28, 'rgba(56,189,248,0.04)')

      drawDots()
      drawStars(t)
      drawFloatingParticles(t)
      drawHorizon(t)
      drawScanline(t)

      raf = requestAnimationFrame(loop)
    }

    resize()
    raf = requestAnimationFrame(loop)

    window.addEventListener('resize', resize)
    return () => {
      window.removeEventListener('resize', resize)
      cancelAnimationFrame(raf)
    }
  }, [])

  return (
    <canvas
      ref={canvasRef}
      id="nimbi-bg"
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        width: '100vw',
        height: '100vh',
        zIndex: 0,
        pointerEvents: 'none',
        display: 'block',
      }}
    />
  )
}
