"use client"

import { useState } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardFooter, CardHeader } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { ServiceIcon } from "@/components/service-icon"
import { Check, Info } from "lucide-react"

const defaultPermissions = [
  { id: "read-calendar", label: "予定の読み取り", description: "カレンダーの予定を表示できます", required: true },
  { id: "write-calendar", label: "予定の作成・編集", description: "新しい予定を作成・変更できます", required: true },
  { id: "delete-calendar", label: "予定の削除", description: "カレンダーから予定を削除できます", required: false },
]

export default function ConsentPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const serviceName = searchParams.get("service") || "Google Calendar"
  const serviceIcon = searchParams.get("icon") || "calendar"

  const [selectedPermissions, setSelectedPermissions] = useState<string[]>(
    defaultPermissions.filter((p) => p.required).map((p) => p.id),
  )

  const handlePermissionToggle = (permissionId: string, required: boolean) => {
    if (required) return
    setSelectedPermissions((prev) =>
      prev.includes(permissionId) ? prev.filter((id) => id !== permissionId) : [...prev, permissionId],
    )
  }

  const handleAllow = () => {
    router.push("/tools")
  }

  const handleCancel = () => {
    router.back()
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center space-y-4">
          <div className="flex justify-center">
            <div className="w-16 h-16 rounded-xl bg-secondary flex items-center justify-center">
              <ServiceIcon icon={serviceIcon} className="h-8 w-8 text-foreground" />
            </div>
          </div>
          <div>
            <h1 className="text-xl font-bold text-foreground">{serviceName}</h1>
            <p className="text-sm text-muted-foreground mt-2">MCPistが以下の権限を要求しています:</p>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          {defaultPermissions.map((permission) => (
            <div
              key={permission.id}
              className="flex items-start gap-3 p-3 rounded-lg bg-secondary/50 hover:bg-secondary transition-colors"
            >
              <Checkbox
                id={permission.id}
                checked={selectedPermissions.includes(permission.id)}
                onCheckedChange={() => handlePermissionToggle(permission.id, permission.required)}
                disabled={permission.required}
                className="mt-0.5"
              />
              <div className="flex-1 min-w-0">
                <label htmlFor={permission.id} className="text-sm font-medium text-foreground cursor-pointer">
                  {permission.label}
                  {permission.required && <span className="text-destructive ml-1">*</span>}
                </label>
                <p className="text-xs text-muted-foreground mt-0.5">{permission.description}</p>
              </div>
              <Check className="h-4 w-4 text-success shrink-0 mt-0.5" />
            </div>
          ))}
        </CardContent>
        <CardFooter className="flex flex-col gap-4">
          <div className="flex gap-3 w-full">
            <Button variant="outline" className="flex-1 bg-transparent" onClick={handleCancel}>
              キャンセル
            </Button>
            <Button className="flex-1" onClick={handleAllow}>
              許可する
            </Button>
          </div>
          <div className="flex items-start gap-2 text-xs text-muted-foreground">
            <Info className="h-4 w-4 shrink-0 mt-0.5" />
            <span>この権限はいつでも設定画面から解除できます</span>
          </div>
        </CardFooter>
      </Card>
    </div>
  )
}
