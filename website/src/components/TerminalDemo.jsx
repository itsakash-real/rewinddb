import { useEffect, useState, useRef } from 'react'

const SEQUENCE = [
  {
    cmd: 'rw init',
    outputLines: [
      { text: '  initialized /', color: 'text' },
      { text: '  directory   .rewind/', color: 'muted' },
      { text: '  branch      main', color: 'muted' },
    ],
  },
  {
    cmd: 'rw save "auth working"',
    outputLines: [
      { text: '  checkpoint saved', color: 'text' },
      { text: '  id       a3f2b1c8', color: 'muted' },
      { text: '  message  "auth working"', color: 'muted' },
      { text: '  files    24 tracked', color: 'muted' },
    ],
  },
  {
    cmd: 'rw run "npm run build"',
    outputLines: [
      { text: '  checkpoint saved: a3f2b1c8', color: 'success' },
      { text: '  running: npm run build', color: 'muted' },
      { text: '  command failed (exit 1)', color: 'error' },
      { text: '  rolling back...', color: 'muted' },
      { text: '  rolled back to a3f2b1c8', color: 'success' },
    ],
  },
  {
    cmd: 'rw goto HEAD~3',
    outputLines: [
      { text: '  restored', color: 'text' },
      { text: '  checkpoint  a3f2b1c8', color: 'muted' },
      { text: '  written     3 file(s)', color: 'muted' },
    ],
  },
]

const COLOR_MAP = {
  accent: 'text-accent',
  success: 'text-success',
  error: 'text-error',
  muted: 'text-text-muted',
  text: 'text-text-secondary',
}

const TYPING_SPEED = 30
const PAUSE_AFTER_CMD = 250
const PAUSE_AFTER_OUTPUT = 1200
const RESTART_DELAY = 3000

export default function TerminalDemo({ className = '' }) {
  const [lines, setLines] = useState([])
  const [currentCmdText, setCurrentCmdText] = useState('')
  const [phase, setPhase] = useState('typing')
  const [seqIdx, setSeqIdx] = useState(0)
  const [outputIdx, setOutputIdx] = useState(0)
  const bottomRef = useRef(null)
  const timeoutRef = useRef(null)

  useEffect(() => () => clearTimeout(timeoutRef.current), [])

  useEffect(() => {
    const step = SEQUENCE[seqIdx]

    if (phase === 'typing') {
      const full = step.cmd
      if (currentCmdText.length < full.length) {
        timeoutRef.current = setTimeout(() => {
          setCurrentCmdText(full.slice(0, currentCmdText.length + 1))
        }, TYPING_SPEED)
      } else {
        timeoutRef.current = setTimeout(() => {
          setLines((prev) => [...prev, { type: 'cmd', text: full }])
          setCurrentCmdText('')
          setPhase('output')
          setOutputIdx(0)
        }, PAUSE_AFTER_CMD)
      }
    }

    if (phase === 'output') {
      const outLines = step.outputLines
      if (outputIdx < outLines.length) {
        timeoutRef.current = setTimeout(() => {
          setLines((prev) => [...prev, { type: 'out', ...outLines[outputIdx] }])
          setOutputIdx((i) => i + 1)
        }, 50)
      } else {
        timeoutRef.current = setTimeout(() => setPhase('pause'), PAUSE_AFTER_OUTPUT)
      }
    }

    if (phase === 'pause') {
      const next = seqIdx + 1
      if (next < SEQUENCE.length) {
        setSeqIdx(next)
        setPhase('typing')
      } else {
        timeoutRef.current = setTimeout(() => {
          setLines([])
          setSeqIdx(0)
          setPhase('typing')
          setCurrentCmdText('')
        }, RESTART_DELAY)
      }
    }
  }, [phase, seqIdx, currentCmdText, outputIdx])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'instant', block: 'nearest' })
  }, [lines, currentCmdText])

  return (
    <div className={`rounded-xl border border-border overflow-hidden ${className}`}>
      {/* Chrome — minimal */}
      <div className="flex items-center gap-2 px-4 py-2.5 bg-[#0d0d0d] border-b border-border">
        <div className="flex gap-1.5">
          <div className="w-2.5 h-2.5 rounded-full bg-white/[0.06]" />
          <div className="w-2.5 h-2.5 rounded-full bg-white/[0.06]" />
          <div className="w-2.5 h-2.5 rounded-full bg-white/[0.06]" />
        </div>
        <span className="text-[11px] text-text-muted font-mono ml-3">~/my-project</span>
      </div>

      {/* Body */}
      <div className="bg-[#080808] p-5 font-mono text-[13px] min-h-[280px] max-h-[380px] overflow-y-auto leading-[1.7]">
        {lines.map((line, i) => (
          <div key={i}>
            {line.type === 'cmd' ? (
              <div className="flex items-start gap-2 mt-1 mb-0.5">
                <span className="text-text-muted select-none">$</span>
                <span className="text-text">{line.text}</span>
              </div>
            ) : (
              <div className={`${COLOR_MAP[line.color] || 'text-text'}`}>
                {line.text || '\u00a0'}
              </div>
            )}
          </div>
        ))}

        {phase === 'typing' && (
          <div className="flex items-start gap-2 mt-1">
            <span className="text-text-muted select-none">$</span>
            <span className="text-text">
              {currentCmdText}
              <span className="animate-blink text-accent/70">_</span>
            </span>
          </div>
        )}

        {phase === 'pause' && seqIdx === SEQUENCE.length - 1 && (
          <div className="flex items-start gap-2 mt-2">
            <span className="text-text-muted select-none">$</span>
            <span className="animate-blink text-accent/70">_</span>
          </div>
        )}

        <div ref={bottomRef} />
      </div>
    </div>
  )
}
