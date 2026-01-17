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
import { useAuth } from "@/lib/auth-context"
import { Download, CreditCard, Calendar, Receipt } from "lucide-react"
import { cn } from "@/lib/utils"

export const dynamic = "force-dynamic"

// 購読中のサービス（モック）
const activeSubscriptions = [
  { serviceId: "jira", serviceName: "Jira", price: 500, startedAt: "2026-01-10" },
  { serviceId: "confluence", serviceName: "Confluence", price: 500, startedAt: "2026-01-10" },
]

// 請求履歴（モック）
const billingHistory = [
  { id: "1", date: "2026-01-15", description: "Jira + Confluence", amount: 1000, status: "paid" as const },
  { id: "2", date: "2025-12-15", description: "Jira", amount: 500, status: "paid" as const },
]

export default function BillingPage() {
  const { user } = useAuth()
  const [paymentDialog, setPaymentDialog] = useState(false)
  const [saved, setSaved] = useState(false)

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat("ja-JP", { style: "currency", currency: "JPY" }).format(price)
  }

  const totalMonthly = activeSubscriptions.reduce((sum, sub) => sum + sub.price, 0)

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">請求情報</h1>
        <p className="text-muted-foreground mt-1">支払い情報と請求履歴を確認</p>
      </div>

      {/* 月額合計 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <Receipt className="h-5 w-5" />
            月額利用料
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-bold">{formatPrice(totalMonthly)}</span>
            <span className="text-muted-foreground">/月</span>
          </div>
          {activeSubscriptions.length > 0 ? (
            <div className="mt-4 space-y-2">
              <p className="text-sm text-muted-foreground">利用中の有料サービス:</p>
              <ul className="space-y-1">
                {activeSubscriptions.map((sub) => (
                  <li key={sub.serviceId} className="flex items-center justify-between text-sm">
                    <span>{sub.serviceName}</span>
                    <span className="text-muted-foreground">{formatPrice(sub.price)}/月</span>
                  </li>
                ))}
              </ul>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground mt-2">
              有料サービスを利用していません
            </p>
          )}
        </CardContent>
      </Card>

      {/* 支払い方法 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <CreditCard className="h-5 w-5" />
            支払い方法
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <CreditCard className="h-4 w-4 text-muted-foreground" />
                <span>•••• •••• •••• 4242</span>
              </div>
              <p className="text-sm text-muted-foreground">有効期限: 12/2028</p>
            </div>
            <Button variant="outline" onClick={() => setPaymentDialog(true)}>
              変更する
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* 次回請求日 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <Calendar className="h-5 w-5" />
            次回請求
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-lg font-medium">2026年2月15日</p>
              <p className="text-sm text-muted-foreground">
                請求予定額: {formatPrice(totalMonthly)}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 請求履歴 */}
      {billingHistory.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">請求履歴</CardTitle>
            <CardDescription>過去の請求と支払い状況</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>日付</TableHead>
                  <TableHead>内容</TableHead>
                  <TableHead>金額</TableHead>
                  <TableHead>ステータス</TableHead>
                  <TableHead className="text-right">請求書</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {billingHistory.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>{item.date}</TableCell>
                    <TableCell>{item.description}</TableCell>
                    <TableCell>{formatPrice(item.amount)}</TableCell>
                    <TableCell>
                      <Badge
                        variant="secondary"
                        className={cn(
                          item.status === "paid" && "bg-green-500/20 text-green-400 border-green-500/30",
                        )}
                      >
                        {item.status === "paid" ? "支払済" : "保留中"}
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
      )}

      {/* Payment Method Dialog */}
      <Dialog open={paymentDialog} onOpenChange={setPaymentDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>支払い方法の変更</DialogTitle>
            <DialogDescription>新しいクレジットカード情報を入力してください</DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <p className="text-sm text-muted-foreground text-center">
              Stripe決済フォームがここに表示されます（デモ）
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPaymentDialog(false)}>
              キャンセル
            </Button>
            <Button
              onClick={() => {
                setPaymentDialog(false)
                setSaved(true)
                setTimeout(() => setSaved(false), 2000)
              }}
            >
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {saved && (
        <div className="fixed bottom-4 right-4 bg-green-500 text-white px-4 py-2 rounded-lg shadow-lg">
          変更を保存しました
        </div>
      )}
    </div>
  )
}
