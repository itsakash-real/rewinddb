import { useEffect, useState, useRef } from 'react'

const SEQUENCE = [
  {
    cmd: 'rw init',
    outputLines: [
      { text: '  ◆  initialized  ──────────────────────', color: 'purple' },
      { text: '', color: 'dim' },
      { text: '     directory    .rewind/', color: 'dim' },
      { text: '     branch       main', color: 'dim' },
    ],
  },
  {
    cmd: 'rw save "auth working"',
    outputLines: [
      { text: '  ◆  checkpoint saved  ─────────────────', color: 'purple' },
      { text: '', color: 'dim' },
      { text: '     id       a3f2b1c8', color: 'cyan' },
      { text: '     message  "auth working"', color: 'text' },
      { text: '     files    24 tracked  ·  0 changed', color: 'dim' },
      { text: '     saved    just now', color: 'dim' },
    ],
  },
  {
    cmd: '# edit some files, break everything...',
    outputLines: [],
    comment: true,
  },
  {
    cmd: 'rw status',
    outputLines: [
      { text: '  ~  src/auth.js', color: 'yellow' },
      { text: '  ~  src/db.js', color: 'yellow' },
      { text: '  +  src/broken.js', color: 'green' },
      { text: '', color: 'dim' },
      { text: '  →  run rw save or rw undo', color: 'dim' },
    ],
  },
  {
    cmd: 'rw undo',
    outputLines: [
      { text: '  ✓  restored to a3f2b1c8', color: 'green' },
      { text: '     1 checkpoint back · 3 files written', color: 'dim' },
    ],
  },
]

const COLOR_MAP = {
  purple: 'text-purple-bright',
  cyan: 'text-cyan',
  green: 'text-green',
  yellow: 'text-yellow-300',
  dim: 'text-text-muted',
  text: 'text-text',
  comment: 'text-text-muted italic',
}

const TYPING_SPEED = 40
const PAUSE_AFTER_CMD = 300
const PAUSE_AFTER_OUTPUT = 1800
const RESTART_DELAY = 2000

export default function TerminalDemo({ className = '' }) {
  const [lines, setLines] = useState([])
  const [currentCmdText, setCurrentCmdText] = useState('')
  const [phase, setPhase] = useState('typing') // typing | output | pause
  const [seqIdx, setSeqIdx] = useState(0)
  const [outputIdx, setOutputIdx] = useState(0)
  const bottomRef = useRef(null)
  const timeoutRef = useRef(null)

  const clear = () => clearTimeout(timeoutRef.current)

  useEffect(() => {
    return () => clear()
  }, [])

  useEffect(() => {
    const step = SEQUENCE[seqIdx]

    if (phase === 'typing') {
      const full = step.cmd
      if (currentCmdText.length < full.length) {
        timeoutRef.current = setTimeout(() => {
          setCurrentCmdText(full.slice(0, currentCmdText.length + 1))
        }, TYPING_SPEED)
      } else {
        // Done typing — commit command line
        timeoutRef.current = setTimeout(() => {
          setLines((prev) => [
            ...prev,
            {
              type: 'cmd',
              text: full,
              comment: step.comment,
            },
          ])
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
          setLines((prev) => [
            ...prev,
            { type: 'out', ...outLines[outputIdx] },
          ])
          setOutputIdx((i) => i + 1)
        }, 80)
      } else {
        timeoutRef.current = setTimeout(() => {
          setPhase('pause')
        }, PAUSE_AFTER_OUTPUT)
      }
    }

    if (phase === 'pause') {
      const next = seqIdx + 1
      if (next < SEQUENCE.length) {
        setSeqIdx(next)
        setPhase('typing')
      } else {
        // Restart
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
    <div className={`rounded-xl border border-border overflow-hidden glow-purple ${className}`}>
      {/* Window chrome */}
      <div className="flex items-center gap-2 px-4 py-3 bg-surface border-b border-border">
        <div className="flex gap-1.5">
          <div className="w-3 h-3 rounded-full bg-[#ff5f56]" />
          <div className="w-3 h-3 rounded-full bg-[#febc2e]" />
          <div className="w-3 h-3 rounded-full bg-[#28c840]" />
        </div>
        <div className="flex-1 flex justify-center">
          <span className="text-xs text-text-muted font-mono">~/my-project</span>
        </div>
      </div>

      {/* Terminal body */}
      <div className="bg-[#080810] p-5 font-mono text-sm min-h-[320px] max-h-[420px] overflow-y-auto">
        {lines.map((line, i) => (
          <div key={i} className="leading-relaxed">
            {line.type === 'cmd' ? (
              <div className="flex items-start gap-2 mb-1">
                <span className="text-purple-DEFAULT select-none mt-0.5">❯</span>
                <span className={line.comment ? 'text-text-muted italic' : 'text-text'}>
                  {line.text}
                </span>
              </div>
            ) : (
              <div
                className={`mb-0.5 ${
                  COLOR_MAP[line.color] || 'text-text'
                }`}
              >
                {line.text || '\u00a0'}
              </div>
            )}
          </div>
        ))}

        {/* Currently typing command */}
        {phase === 'typing' && (
          <div className="flex items-start gap-2">
            <span className="text-purple-DEFAULT select-none mt-0.5">❯</span>
            <span className="text-text">
              {currentCmdText}
              <span className="animate-blink text-purple-DEFAULT">▋</span>
            </span>
          </div>
        )}

        {/* Idle cursor when pausing */}
        {phase === 'pause' && seqIdx === SEQUENCE.length - 1 && (
          <div className="flex items-start gap-2 mt-1">
            <span className="text-purple-DEFAULT select-none">❯</span>
            <span className="animate-blink text-purple-DEFAULT">▋</span>
          </div>
        )}

        <div ref={bottomRef} />
      </div>
    </div>
  )
}
