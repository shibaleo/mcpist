"use client"

import { useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { useAuth } from "@/lib/auth-context"
import {
  plans,
  organizationPlan as initialOrgPlan,
  billingHistory,
  type PlanType,
  type OrganizationPlan,
} from "@/lib/data"
import { Check, Download, CreditCard, Sparkles } from "lucide-react"
import { cn } from "@/lib/utils"
import { toast } from "sonner"

export default function BillingPage() {
  const { isAdmin } = useAuth()
  const [orgPlan, setOrgPlan] = useState<OrganizationPlan>(initialOrgPlan)
  const [upgradeDialog, setUpgradeDialog] = useState<PlanType | null>(null)
  const [paymentDialog, setPaymentDialog] = useState(false)

  if (!isAdmin) {
    return (
      <div className="flex items-center justify-center h-[50vh]">
        <p className="text-muted-foreground">このページにアクセスする権限がありません</p>
      </div>
    )
  }

  const handleUpgrade = (planId: PlanType) => {
    setUpgradeDialog(planId)
  }

  const confirmUpgrade = () => {
    if (!upgradeDialog) return
    const plan = plans.find((p) => p.id === upgradeDialog)
    if (!plan) return

    setOrgPlan((prev) => ({
      ...prev,
      currentPlan: upgradeDialog,
      planName: plan.name,
    }))
    setUpgradeDialog(null)
    toast.success(`${plan.name}プランにアップグレードしました`)
  }

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat("ja-JP", { style: "currency", currency: "JPY" }).format(price)
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">プランと請求</h1>
        <p className="text-muted-foreground mt-1">組織のプランと支払い情報を管理</p>
      </div>

      {/* Current Plan */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">現在のプラン</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <span className="text-2xl font-bold">{orgPlan.planName}</span>
                <Badge>現在のプラン</Badge>
              </div>
              <p className="text-muted-foreground">
                {formatPrice(plans.find((p) => p.id === orgPlan.currentPlan)?.price || 0)}/月
              </p>
              <p className="text-sm text-muted-foreground">
                ユーザー: {orgPlan.userCount}/{orgPlan.userLimit}名
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Plan Selection */}
      <div>
        <h2 className="text-lg font-semibold mb-4">プランを選択</h2>
        <div className="grid gap-4 md:grid-cols-3">
          {plans.map((plan) => {
            const isCurrent = orgPlan.currentPlan === plan.id
            const isRecommended = plan.id === "pro"
            const isDowngrade =
              (orgPlan.currentPlan === "max" && plan.id !== "max") ||
              (orgPlan.currentPlan === "pro" && plan.id === "free")

            return (
              <Card
                key={plan.id}
                className={cn(
                  "relative transition-all",
                  isCurrent && "border-primary",
                  isRecommended && !isCurrent && "border-blue-500/50",
                )}
              >
                {isRecommended && !isCurrent && (
                  <div className="absolute -top-3 left-1/2 -translate-x-1/2">
                    <Badge className="bg-blue-500 text-white">
                      <Sparkles className="h-3 w-3 mr-1" />
                      おすすめ
                    </Badge>
                  </div>
                )}
                <CardHeader>
                  <CardTitle className="flex items-center justify-between">
                    {plan.name}
                    {isCurrent && <Badge variant="secondary">現在</Badge>}
                  </CardTitle>
                  <CardDescription>{plan.description}</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="text-3xl font-bold">
                    {formatPrice(plan.price)}
                    <span className="text-sm font-normal text-muted-foreground">/月</span>
                  </div>
                  <ul className="space-y-2">
                    {plan.features.map((feature, i) => (
                      <li key={i} className="flex items-center gap-2 text-sm">
                        <Check className="h-4 w-4 text-success shrink-0" />
                        {feature}
                      </li>
                    ))}
                  </ul>
                  <Button
                    className="w-full"
                    variant={isCurrent ? "secondary" : isDowngrade ? "outline" : "default"}
                    disabled={isCurrent}
                    onClick={() => handleUpgrade(plan.id)}
                  >
                    {isCurrent ? "現在のプラン" : isDowngrade ? "ダウングレード" : "アップグレード"}
                  </Button>
                </CardContent>
              </Card>
            )
          })}
        </div>
      </div>

      {/* Billing Info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">請求情報</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="space-y-1">
              <p className="text-sm text-muted-foreground">
                請求サイクル: {orgPlan.billingCycle === "monthly" ? "月払い" : "年払い"}
              </p>
              <p className="text-sm text-muted-foreground">次回請求日: {orgPlan.nextBillingDate}</p>
              <div className="flex items-center gap-2 text-sm">
                <CreditCard className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">•••• 4242</span>
              </div>
            </div>
            <Button variant="outline" onClick={() => setPaymentDialog(true)}>
              支払い方法を変更
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Billing History */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">請求履歴</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>日付</TableHead>
                <TableHead>プラン</TableHead>
                <TableHead>金額</TableHead>
                <TableHead>ステータス</TableHead>
                <TableHead className="text-right">請求書</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {billingHistory.map((item) => (
                <TableRow key={item.id}>
                  <TableCell>{item.date}</TableCell>
                  <TableCell>{item.plan}</TableCell>
                  <TableCell>{formatPrice(item.amount)}</TableCell>
                  <TableCell>
                    <Badge
                      variant="secondary"
                      className={cn(
                        item.status === "paid" && "bg-success/20 text-success border-success/30",
                        item.status === "pending" && "bg-warning/20 text-warning border-warning/30",
                        item.status === "failed" && "bg-destructive/20 text-destructive border-destructive/30",
                      )}
                    >
                      {item.status === "paid" ? "支払済" : item.status === "pending" ? "保留中" : "失敗"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm">
                      <Download className="h-4 w-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Upgrade Confirmation */}
      <AlertDialog open={!!upgradeDialog} onOpenChange={(open) => !open && setUpgradeDialog(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>プランを変更しますか？</AlertDialogTitle>
            <AlertDialogDescription>
              {upgradeDialog && plans.find((p) => p.id === upgradeDialog)?.name}プランに変更します。
              次回の請求から新しい料金が適用されます。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>キャンセル</AlertDialogCancel>
            <AlertDialogAction onClick={confirmUpgrade}>変更する</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Payment Method Dialog */}
      <Dialog open={paymentDialog} onOpenChange={setPaymentDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>支払い方法の変更</DialogTitle>
            <DialogDescription>新しいクレジットカード情報を入力してください</DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <p className="text-sm text-muted-foreground text-center">Stripe決済フォームがここに表示されます（デモ）</p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPaymentDialog(false)}>
              キャンセル
            </Button>
            <Button
              onClick={() => {
                setPaymentDialog(false)
                toast.success("支払い方法を更新しました")
              }}
            >
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
