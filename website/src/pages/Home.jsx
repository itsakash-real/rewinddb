import { useEffect } from 'react'
import Hero from '../components/Hero'
import HowItWorks from '../components/HowItWorks'
import WhatHappensNext from '../components/WhatHappensNext'
import Features from '../components/Features'
import FinalCTA from '../components/FinalCTA'

export default function Home() {
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add('visible')
          }
        })
      },
      { threshold: 0.1 }
    )

    const els = document.querySelectorAll('.scroll-reveal')
    els.forEach((el) => observer.observe(el))
    return () => observer.disconnect()
  }, [])

  return (
    <>
      <Hero />
      <HowItWorks />
      <WhatHappensNext />
      <Features />
      <FinalCTA />
    </>
  )
}
