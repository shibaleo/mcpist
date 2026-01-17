import type React from "react"
import {
  Calendar,
  FileText,
  Github,
  MessageSquare,
  Wallet,
  Calculator,
  Cloud,
  HardDrive,
  LayoutGrid,
  Kanban,
  CheckSquare,
  TrendingUp,
  Wrench,
} from "lucide-react"

const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  calendar: Calendar,
  "file-text": FileText,
  github: Github,
  "message-square": MessageSquare,
  wallet: Wallet,
  calculator: Calculator,
  cloud: Cloud,
  "hard-drive": HardDrive,
  "layout-grid": LayoutGrid,
  kanban: Kanban,
  "check-square": CheckSquare,
  "trending-up": TrendingUp,
}

interface ServiceIconProps {
  icon: string
  className?: string
}

export function ServiceIcon({ icon, className }: ServiceIconProps) {
  const Icon = iconMap[icon] || Wrench
  return <Icon className={className} />
}
