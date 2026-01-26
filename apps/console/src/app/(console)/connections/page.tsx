import { redirect } from "next/navigation"

// connectionsページは/toolsに統合されました
export default function ConnectionsPage() {
  redirect("/tools")
}
