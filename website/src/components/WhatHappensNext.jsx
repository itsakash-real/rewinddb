export default function WhatHappensNext() {
  const steps = [
    { label: 'Save', color: '#38bdf8' },
    { label: 'Break', color: '#6366f1' },
    { label: 'Undo', color: '#34d399' },
    { label: 'Back', color: '#38bdf8' },
  ]

  return (
    <section className="py-16">
      <div className="max-w-4xl mx-auto px-6">
        <div className="scroll-reveal">
          <h2 className="text-[28px] md:text-[36px] font-bold tracking-[-0.02em] text-center mb-10">
            What happens next
          </h2>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-3 sm:gap-0">
            {steps.map((step, i) => (
              <div key={step.label} className="flex items-center">
                <div
                  className="px-6 py-3.5 rounded-lg text-[16px] font-semibold text-white"
                  style={{
                    background: 'rgba(56,189,248,0.05)',
                    border: `1px solid ${step.color}33`,
                    boxShadow: `0 0 20px ${step.color}10`,
                  }}
                >
                  {step.label}
                </div>
                {i < steps.length - 1 && (
                  <span className="flow-arrow text-[20px] font-bold mx-4 hidden sm:inline">&rarr;</span>
                )}
                {i < steps.length - 1 && (
                  <span className="flow-arrow text-[20px] font-bold my-1.5 sm:hidden">&darr;</span>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
