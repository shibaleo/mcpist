"use client"

import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { usageRequests, type UsageRequest } from "@/lib/data"
import { useAuth } from "@/lib/auth-context"
import { useMediaQuery } from "@/hooks/use-media-query"
import { ServiceIcon } from "@/components/service-icon"
import { services } from "@/lib/data"
import { Clock, CheckCircle, XCircle } from "lucide-react"

export default function RequestsPage() {
  const { isAdmin } = useAuth()
  const [requests, setRequests] = useState<UsageRequest[]>(usageRequests)
  const [selectedRequest, setSelectedRequest] = useState<UsageRequest | null>(null)
  const [rejectionReason, setRejectionReason] = useState("")
  const [showRejectInput, setShowRejectInput] = useState(false)
  const isMobile = useMediaQuery("(max-width: 768px)")

  if (!isAdmin) {
    return (
      <div className="flex items-center justify-center h-[50vh]">
        <p className="text-muted-foreground">このページにアクセスする権限がありません</p>
      </div>
    )
  }

  const handleApprove = (request: UsageRequest) => {
    setRequests(
      requests.map((r) =>
        r.id === request.id
          ? {
              ...r,
              status: "approved" as const,
              reviewedAt: new Date().toISOString().replace("T", " ").slice(0, 16),
              reviewedBy: "山田 太郎",
            }
          : r,
      ),
    )
    setSelectedRequest(null)
  }

  const handleReject = (request: UsageRequest) => {
    if (!rejectionReason) return
    setRequests(
      requests.map((r) =>
        r.id === request.id
          ? {
              ...r,
              status: "rejected" as const,
              reviewedAt: new Date().toISOString().replace("T", " ").slice(0, 16),
              reviewedBy: "山田 太郎",
              rejectionReason,
            }
          : r,
      ),
    )
    setSelectedRequest(null)
    setRejectionReason("")
    setShowRejectInput(false)
  }

  const statusConfig = {
    pending: { label: "審査中", class: "bg-warning/20 text-warning border-warning/30", icon: Clock },
    approved: { label: "承認済", class: "bg-success/20 text-success border-success/30", icon: CheckCircle },
    rejected: { label: "却下", class: "bg-destructive/20 text-destructive border-destructive/30", icon: XCircle },
  }

  const getServiceIcon = (serviceId: string) => {
    const service = services.find((s) => s.id === serviceId)
    return service?.icon || "wrench"
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">利用申請一覧</h1>
        <p className="text-muted-foreground mt-1">ツールとサービスへのアクセス申請を管理します</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">申請一覧</CardTitle>
        </CardHeader>
        <CardContent>
          {isMobile ? (
            <div className="space-y-3">
              {requests.map((request) => {
                const StatusIcon = statusConfig[request.status].icon
                return (
                  <Card
                    key={request.id}
                    className="p-4 cursor-pointer hover:bg-accent/50 transition-colors"
                    onClick={() => setSelectedRequest(request)}
                  >
                    <div className="flex items-start justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <ServiceIcon icon={getServiceIcon(request.serviceId)} className="h-5 w-5" />
                        <span className="font-medium text-foreground">{request.serviceName}</span>
                      </div>
                      <Badge variant="outline" className={statusConfig[request.status].class}>
                        <StatusIcon className="h-3 w-3 mr-1" />
                        {statusConfig[request.status].label}
                      </Badge>
                    </div>
                    <p className="text-sm text-muted-foreground">{request.userName}</p>
                    <p className="text-xs text-muted-foreground mt-1">{request.requestedAt}</p>
                  </Card>
                )
              })}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>申請者</TableHead>
                  <TableHead>申請ツール</TableHead>
                  <TableHead>申請日時</TableHead>
                  <TableHead>ステータス</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {requests.map((request) => {
                  const StatusIcon = statusConfig[request.status].icon
                  return (
                    <TableRow
                      key={request.id}
                      className="cursor-pointer hover:bg-accent/50"
                      onClick={() => setSelectedRequest(request)}
                    >
                      <TableCell>
                        <div>
                          <p className="font-medium">{request.userName}</p>
                          <p className="text-sm text-muted-foreground">{request.userEmail}</p>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <ServiceIcon icon={getServiceIcon(request.serviceId)} className="h-4 w-4" />
                          <span>{request.serviceName}</span>
                          {request.toolName && <span className="text-muted-foreground">/ {request.toolName}</span>}
                        </div>
                      </TableCell>
                      <TableCell>{request.requestedAt}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className={statusConfig[request.status].class}>
                          <StatusIcon className="h-3 w-3 mr-1" />
                          {statusConfig[request.status].label}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Sheet
        open={!!selectedRequest}
        onOpenChange={() => {
          setSelectedRequest(null)
          setShowRejectInput(false)
          setRejectionReason("")
        }}
      >
        <SheetContent className="sm:max-w-md">
          {selectedRequest && (
            <>
              <SheetHeader>
                <SheetTitle>申請詳細</SheetTitle>
                <SheetDescription>利用申請の詳細情報を確認します</SheetDescription>
              </SheetHeader>
              <div className="mt-6 space-y-6">
                <div className="flex items-center gap-3 p-4 bg-secondary rounded-lg">
                  <div className="w-12 h-12 rounded-lg bg-background flex items-center justify-center">
                    <ServiceIcon icon={getServiceIcon(selectedRequest.serviceId)} className="h-6 w-6" />
                  </div>
                  <div>
                    <p className="font-medium text-foreground">{selectedRequest.serviceName}</p>
                    {selectedRequest.toolName && (
                      <p className="text-sm text-muted-foreground">{selectedRequest.toolName}</p>
                    )}
                  </div>
                </div>

                <div className="space-y-4">
                  <div>
                    <Label className="text-muted-foreground">申請者</Label>
                    <p className="font-medium text-foreground">{selectedRequest.userName}</p>
                    <p className="text-sm text-muted-foreground">{selectedRequest.userEmail}</p>
                  </div>
                  <div>
                    <Label className="text-muted-foreground">申請日時</Label>
                    <p className="text-foreground">{selectedRequest.requestedAt}</p>
                  </div>
                  <div>
                    <Label className="text-muted-foreground">申請理由</Label>
                    <p className="text-foreground bg-secondary p-3 rounded-lg mt-1">{selectedRequest.reason}</p>
                  </div>
                  {selectedRequest.status !== "pending" && (
                    <>
                      <div>
                        <Label className="text-muted-foreground">審査日時</Label>
                        <p className="text-foreground">{selectedRequest.reviewedAt}</p>
                      </div>
                      <div>
                        <Label className="text-muted-foreground">審査者</Label>
                        <p className="text-foreground">{selectedRequest.reviewedBy}</p>
                      </div>
                      {selectedRequest.rejectionReason && (
                        <div>
                          <Label className="text-muted-foreground">却下理由</Label>
                          <p className="text-foreground bg-secondary p-3 rounded-lg mt-1">
                            {selectedRequest.rejectionReason}
                          </p>
                        </div>
                      )}
                    </>
                  )}
                </div>

                {selectedRequest.status === "pending" && (
                  <div className="space-y-3 pt-4 border-t border-border">
                    {showRejectInput ? (
                      <div className="space-y-3">
                        <div className="space-y-2">
                          <Label>却下理由</Label>
                          <Textarea
                            placeholder="却下の理由を入力してください"
                            value={rejectionReason}
                            onChange={(e) => setRejectionReason(e.target.value)}
                          />
                        </div>
                        <div className="flex gap-2">
                          <Button
                            variant="outline"
                            className="flex-1 bg-transparent"
                            onClick={() => {
                              setShowRejectInput(false)
                              setRejectionReason("")
                            }}
                          >
                            戻る
                          </Button>
                          <Button
                            variant="destructive"
                            className="flex-1"
                            onClick={() => handleReject(selectedRequest)}
                            disabled={!rejectionReason}
                          >
                            却下する
                          </Button>
                        </div>
                      </div>
                    ) : (
                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          className="flex-1 bg-transparent"
                          onClick={() => setShowRejectInput(true)}
                        >
                          却下
                        </Button>
                        <Button className="flex-1" onClick={() => handleApprove(selectedRequest)}>
                          承認する
                        </Button>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </>
          )}
        </SheetContent>
      </Sheet>
    </div>
  )
}
