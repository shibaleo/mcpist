import { Suspense } from "react"
import { ServiceAuthContent } from "@/components/service-auth/service-auth-content"

export default function ServiceAuthPage() {
  return (
    <Suspense fallback={null}>
      <ServiceAuthContent />
    </Suspense>
  )
}
