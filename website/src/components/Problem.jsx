import { motion } from 'framer-motion'

const PAINS = [
  {
    icon: '💥',
    title: 'You changed one config.',
    body: 'Now the whole thing is on fire. You have no idea which file did it. Git shows nothing because you never committed.',
  },
  {
    icon: '🤖',
    title: 'AI wrote 80 lines of "improvements."',
    body: "It looked great. You accepted it. Now nothing compiles. There's no obvious undo. Ctrl+Z doesn't go back that far.",
  },
  {
    icon: '🪴',
    title: 'The experiment spiral.',
    body: 'You try approach A. Breaks. Try approach B. Also breaks. Now you\'ve lost the version that kind of worked.',
  },
  {
    icon: '😭',
    title: '"It worked 20 minutes ago."',
    body: "Classic. You know it was working. You just don't know when it stopped. And you didn't save anything.",
  },
]

const fade = {
  hidden: { opacity: 0, y: 24 },
  show: (i) => ({
    opacity: 1,
    y: 0,
    transition: { delay: i * 0.1, duration: 0.5 },
  }),
}

export default function Problem() {
  return (
    <section className="py-28 relative">
      <div className="max-w-6xl mx-auto px-6">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="text-center mb-16"
        >
          <div className="inline-block text-xs font-mono text-purple-DEFAULT border border-purple-glow/30 bg-purple-glow/5 rounded-full px-3 py-1 mb-4">
            the problem
          </div>
          <h2 className="text-4xl sm:text-5xl font-bold tracking-tight mb-4 text-gradient-white">
            Every developer knows this feeling.
          </h2>
          <p className="text-text-muted text-lg max-w-xl mx-auto">
            Git is great — for commits. But commits are heavy. Between commits, you're flying blind.
          </p>
        </motion.div>

        {/* Pain cards */}
        <div className="grid sm:grid-cols-2 gap-4">
          {PAINS.map((p, i) => (
            <motion.div
              key={p.title}
              custom={i}
              initial="hidden"
              whileInView="show"
              viewport={{ once: true }}
              variants={fade}
              className="group p-6 rounded-2xl bg-surface border border-border hover:border-border-light transition-all duration-300"
            >
              <div className="text-3xl mb-4">{p.icon}</div>
              <h3 className="font-semibold text-text text-lg mb-2">{p.title}</h3>
              <p className="text-text-muted text-sm leading-relaxed">{p.body}</p>
            </motion.div>
          ))}
        </div>

        {/* Callout */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.4 }}
          className="mt-12 text-center"
        >
          <p className="text-2xl font-semibold text-text">
            Git can't save you here.{' '}
            <span className="text-gradient">Drift can.</span>
          </p>
        </motion.div>
      </div>
    </section>
  )
}
