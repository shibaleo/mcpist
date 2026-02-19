import type React from "react"
import {
  SiNotion,
  SiGithub,
  SiJira,
  SiConfluence,
  SiSupabase,
  SiAirtable,
  SiGooglecalendar,
  SiGoogletasks,
  SiGoogledrive,
  SiGoogledocs,
  SiGooglesheets,
  SiTodoist,
  SiTrello,
  SiAsana,
  SiGrafana,
  SiDropbox,
  SiPostgresql,
  SiTicktick,
} from "react-icons/si"
import { VscAzure } from "react-icons/vsc"
import { SiGoogleappsscript } from "react-icons/si"
import { Wrench } from "lucide-react"

// Brand colors for each service
const brandColors: Record<string, string> = {
  notion: "#000000",
  github: "#181717",
  jira: "#0052CC",
  confluence: "#172B4D",
  supabase: "#3FCF8E",
  airtable: "#18BFFF",
  google_calendar: "#4285F4",
  google_tasks: "#4285F4",
  google_drive: "#4285F4",
  google_docs: "#4285F4",
  google_sheets: "#0F9D58",
  google_apps_script: "#4285F4",
  microsoft_todo: "#0078D4",
  postgresql: "#4169E1",
  ticktick: "#4772FA",
  todoist: "#E44332",
  trello: "#0052CC",
  asana: "#F06A6A",
  grafana: "#F46800",
  dropbox: "#0061FF",
}

const iconComponents: Record<string, React.ComponentType<{ className?: string; style?: React.CSSProperties }>> = {
  notion: SiNotion,
  github: SiGithub,
  jira: SiJira,
  confluence: SiConfluence,
  supabase: SiSupabase,
  airtable: SiAirtable,
  google_calendar: SiGooglecalendar,
  google_tasks: SiGoogletasks,
  google_drive: SiGoogledrive,
  google_docs: SiGoogledocs,
  google_sheets: SiGooglesheets,
  google_apps_script: SiGoogleappsscript,
  microsoft_todo: VscAzure,
  postgresql: SiPostgresql,
  ticktick: SiTicktick,
  todoist: SiTodoist,
  trello: SiTrello,
  asana: SiAsana,
  grafana: SiGrafana,
  dropbox: SiDropbox,
}

interface ModuleIconProps {
  moduleName: string
  className?: string
  colored?: boolean
}

export function ModuleIcon({ moduleName, className, colored = true }: ModuleIconProps) {
  const Icon = iconComponents[moduleName]
  if (!Icon) {
    return <Wrench className={className} />
  }
  const color = colored ? brandColors[moduleName] : undefined
  return <Icon className={className} style={color ? { color } : undefined} />
}
