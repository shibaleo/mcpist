"use client"

// Services with generic lucide icon names
const services = [
  { name: "Notes", icon: "sticky-note" },
  { name: "Code", icon: "code" },
  { name: "Calendar", icon: "calendar" },
  { name: "Project", icon: "kanban" },
  { name: "Database", icon: "database" },
  { name: "Tasks", icon: "check-square" },
]

// Layout constants
const SVG_W = 920
const SVG_H = 420

// User node position (left)
const USER_X = 100
const USER_Y = SVG_H / 2

// Hub center (single large hexagon)
const HUB_X = 300
const HUB_R = 66

// Service nodes (right column) — icon placed at path endpoint
const SVC_X = 650
const SVC_GAP = 68

// Calculate service Y positions centered on USER_Y
const svcPositions = services.map((_, i) => {
  const offset = i - (services.length - 1) / 2
  return USER_Y + offset * SVC_GAP
})

// End X for service paths (dot endpoint)
const SVC_END_X = SVC_X
// Icon offset from path endpoint
const ICON_OFFSET_X = 36

// Exit points distributed along the 3 right-facing edges of the hexagon

function hexVertex(i: number): { x: number; y: number } {
  const angle = (Math.PI / 3) * i - Math.PI / 2
  return { x: HUB_X + HUB_R * Math.cos(angle), y: USER_Y + HUB_R * Math.sin(angle) }
}

const vtxTop = hexVertex(0)
const vtxTopRight = hexVertex(1)
const vtxBotRight = hexVertex(2)
const vtxBot = hexVertex(3)

const rightEdges = [
  { from: vtxTop, to: vtxTopRight },
  { from: vtxTopRight, to: vtxBotRight },
  { from: vtxBotRight, to: vtxBot },
]

function getExitPoint(index: number): { x: number; y: number } {
  const n = services.length
  const t = n > 1 ? (index + 0.5) / n : 0.5
  const edgeT = t * rightEdges.length
  const edgeIdx = Math.min(Math.floor(edgeT), rightEdges.length - 1)
  const localT = edgeT - edgeIdx
  const edge = rightEdges[edgeIdx]
  return {
    x: edge.from.x + localT * (edge.to.x - edge.from.x),
    y: edge.from.y + localT * (edge.to.y - edge.from.y),
  }
}

const exitPoints = services.map((_, i) => getExitPoint(i))

const TAN60 = Math.tan(Math.PI / 3)

function buildHubToServicePath(index: number): string {
  const svcY = svcPositions[index]
  const ex = exitPoints[index].x
  const ey = exitPoints[index].y
  const dy = svcY - ey
  if (Math.abs(dy) < 1) {
    return `M ${ex} ${ey} L ${SVC_END_X} ${svcY}`
  }
  const foldDx = Math.abs(dy) / TAN60
  const foldEndX = ex + foldDx
  return `M ${ex} ${ey} L ${foldEndX} ${svcY} L ${SVC_END_X} ${svcY}`
}

function buildUserToHubPath(): string {
  // Left edge of hexagon: midpoint of vertex 4 and vertex 5
  const v4 = hexVertex(4)
  const v5 = hexVertex(5)
  const hubEntryX = (v4.x + v5.x) / 2
  return `M ${USER_X} ${USER_Y} L ${hubEntryX} ${USER_Y}`
}

function buildBranchPath(index: number): string {
  return buildHubToServicePath(index)
}

function buildTrunkPath(): string {
  const v4 = hexVertex(4)
  const v5 = hexVertex(5)
  const hubEntryX = (v4.x + v5.x) / 2
  return `M ${USER_X} ${USER_Y} L ${hubEntryX} ${USER_Y}`
}

function hexagonPath(r: number): string {
  const points = []
  for (let i = 0; i < 6; i++) {
    const angle = (Math.PI / 3) * i - Math.PI / 2
    points.push(`${r * Math.cos(angle)},${r * Math.sin(angle)}`)
  }
  return `M ${points.join(" L ")} Z`
}

// Lucide icon SVG paths (24x24 viewBox) — stroke-based
const iconPaths: Record<string, string[]> = {
  "sticky-note": [
    "M16 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V8Z",
    "M15 3v4a2 2 0 0 0 2 2h4",
  ],
  "code": [
    "m18 16 4-4-4-4",
    "m6 8-4 4 4 4",
    "m14.5 4-5 16",
  ],
  "calendar": [
    "M8 2v4", "M16 2v4",
    "M3 10h18",
    "M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2Z",
  ],
  "kanban": [
    "M6 5v11", "M12 5v6", "M18 5v14",
  ],
  "database": [
    "M12 3C7.58 3 4 4.79 4 7s3.58 4 8 4 8-1.79 8-4-3.58-4-8-4Z",
    "M4 7v10c0 2.21 3.58 4 8 4s8-1.79 8-4V7",
    "M4 12c0 2.21 3.58 4 8 4s8-1.79 8-4",
  ],
  "check-square": [
    "m9 11 3 3L22 4",
    "M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11",
  ],
}

export function ArchitectureDiagram() {
  return (
    <div className="relative w-full max-w-6xl mx-auto">
      <svg
        viewBox={`0 0 ${SVG_W} ${SVG_H}`}
        className="w-full h-auto"
        xmlns="http://www.w3.org/2000/svg"
        style={{ overflow: "visible" }}
      >
        <defs>
          <filter id="glow-orange" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur stdDeviation="6" result="blur" />
            <feComposite in="SourceGraphic" in2="blur" operator="over" />
          </filter>
          <filter id="glow-particle" x="-200%" y="-200%" width="500%" height="500%">
            <feGaussianBlur stdDeviation="4" result="blur" />
            <feComposite in="SourceGraphic" in2="blur" operator="over" />
          </filter>
          <radialGradient id="user-glow">
            <stop offset="0%" stopColor="var(--accent-value)" stopOpacity="0.35" />
            <stop offset="25%" stopColor="var(--accent-value)" stopOpacity="0.2" />
            <stop offset="55%" stopColor="var(--accent-value)" stopOpacity="0.08" />
            <stop offset="80%" stopColor="var(--accent-value)" stopOpacity="0.02" />
            <stop offset="100%" stopColor="var(--accent-value)" stopOpacity="0" />
          </radialGradient>
          <radialGradient id="hex-glow">
            <stop offset="0%" stopColor="var(--accent-value)" stopOpacity="0.4" />
            <stop offset="50%" stopColor="var(--accent-value)" stopOpacity="0.15" />
            <stop offset="100%" stopColor="var(--accent-value)" stopOpacity="0" />
          </radialGradient>
        </defs>

        {/* === Connection paths === */}
        <path
          d={buildUserToHubPath()}
          fill="none"
          stroke="var(--accent-value)"
          strokeWidth="2"
          strokeOpacity="0.25"
        />
        {services.map((_, i) => (
          <path
            key={`path-${i}`}
            d={buildHubToServicePath(i)}
            fill="none"
            stroke="var(--accent-value)"
            strokeWidth="2"
            strokeOpacity="0.25"
          />
        ))}

        {/* === Trunk particle === */}
        <g>
          <circle r="4" fill="var(--accent-value)" filter="url(#glow-particle)">
            <animateMotion dur="1s" repeatCount="indefinite">
              <mpath href="#trunkpath" />
            </animateMotion>
          </circle>
          <circle r="2" fill="white" opacity="0.9">
            <animateMotion dur="1s" repeatCount="indefinite">
              <mpath href="#trunkpath" />
            </animateMotion>
          </circle>
        </g>

        {/* === Branch particles === */}
        {services.map((_, i) => {
          // Pseudo-random begin offsets for natural feel
          const beginOffsets = [0.1, 0.7, 0.3, 1.2, 0.5, 0.9]
          const begin = beginOffsets[i % beginOffsets.length]
          return (
            <g key={`particle-${i}`}>
              <circle r="3.5" fill="var(--accent-value)" filter="url(#glow-particle)">
                <animateMotion
                  dur={`${1.8 + i * 0.2}s`}
                  repeatCount="indefinite"
                  begin={`${begin}s`}
                >
                  <mpath href={`#branchpath-${i}`} />
                </animateMotion>
              </circle>
              <circle r="1.5" fill="white" opacity="0.9">
                <animateMotion
                  dur={`${1.8 + i * 0.2}s`}
                  repeatCount="indefinite"
                  begin={`${begin}s`}
                >
                  <mpath href={`#branchpath-${i}`} />
                </animateMotion>
              </circle>
            </g>
          )
        })}

        {/* Hidden paths for animateMotion */}
        <path id="trunkpath" d={buildTrunkPath()} fill="none" stroke="none" />
        {services.map((_, i) => (
          <path
            key={`branchpath-${i}`}
            id={`branchpath-${i}`}
            d={buildBranchPath(i)}
            fill="none"
            stroke="none"
          />
        ))}

        {/* === User node: lucide User icon === */}
        <circle cx={USER_X} cy={USER_Y} r="140" fill="url(#user-glow)" />
        <g transform={`translate(${USER_X - 24}, ${USER_Y - 24}) scale(2)`} filter="url(#glow-orange)">
          <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" fill="none" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          <circle cx="12" cy="7" r="4" fill="none" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </g>

        {/* === Hub hexagon === */}
        <g transform={`translate(${HUB_X}, ${USER_Y})`}>
          <circle r={HUB_R + 40} fill="url(#hex-glow)" />
          <path d={hexagonPath(HUB_R)} fill="none" stroke="white" strokeWidth="1.5" strokeOpacity="0.8" />
          {/* Brain icons */}
          {[
            { x: 0, y: -22 },
            { x: -24, y: 16 },
            { x: 24, y: 16 },
          ].map((pos, i) => (
            <g key={`brain-${i}`} transform={`translate(${pos.x - 12 * 1.7}, ${pos.y - 12 * 1.7}) scale(1.7)`}>
              <path d="M12 5a3 3 0 1 0-5.997.125 4 4 0 0 0-2.526 5.77 4 4 0 0 0 .556 6.588A4 4 0 1 0 12 18Z" fill="none" stroke="var(--accent-value)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" opacity="0.7" />
              <path d="M12 5a3 3 0 1 1 5.997.125 4 4 0 0 1 2.526 5.77 4 4 0 0 1-.556 6.588A4 4 0 1 1 12 18Z" fill="none" stroke="var(--accent-value)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" opacity="0.7" />
              <path d="M12 5v14" fill="none" stroke="var(--accent-value)" strokeWidth="1.2" strokeLinecap="round" opacity="0.4" />
              <path d="M6.5 8.5c2 0 3.5 1 5.5 1s3.5-1 5.5-1" fill="none" stroke="var(--accent-value)" strokeWidth="1.2" strokeLinecap="round" opacity="0.4" />
              <path d="M6.5 14.5c2 0 3.5-1 5.5-1s3.5 1 5.5 1" fill="none" stroke="var(--accent-value)" strokeWidth="1.2" strokeLinecap="round" opacity="0.4" />
            </g>
          ))}
        </g>

        {/* === Path endpoint dots === */}
        {services.map((_, i) => {
          const y = svcPositions[i]
          return (
            <circle
              key={`dot-${i}`}
              cx={SVC_END_X}
              cy={y}
              r="4"
              fill="var(--accent-value)"
              fillOpacity="0.8"
              stroke="var(--accent-value)"
              strokeWidth="1.5"
              strokeOpacity="0.8"
            />
          )
        })}

        {/* === Service icons (offset right from path endpoint) === */}
        {services.map((svc, i) => {
          const y = svcPositions[i]
          const paths = iconPaths[svc.icon] || []
          return (
            <g key={`svc-${i}`} transform={`translate(${SVC_END_X + ICON_OFFSET_X - 24}, ${y - 24}) scale(2)`}>
              {paths.map((d, j) => (
                <path
                  key={j}
                  d={d}
                  fill="none"
                  stroke="white"
                  strokeWidth="1.2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              ))}
            </g>
          )
        })}

        {/* Service labels */}
        {services.map((svc, i) => {
          const y = svcPositions[i]
          return (
            <text
              key={`label-${i}`}
              x={SVC_END_X + ICON_OFFSET_X + 30}
              y={y + 9}
              fill="white"
              fontSize="26"
              fontFamily="var(--font-sans)"
            >
              {svc.name}
            </text>
          )
        })}
      </svg>
    </div>
  )
}
