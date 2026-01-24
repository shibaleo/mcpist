import { Suspense } from "react"
import { MyConnectionsContent } from "@/components/my-connections/my-connections-content"

export default function MyConnectionsPage() {
  return (
    <Suspense fallback={null}>
      <MyConnectionsContent />
    </Suspense>
  )
}
