"use client"

import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import { useState, useEffect, useRef } from "react"
import { cn } from "@/lib/utils"
import { useAuth } from "@/lib/auth/auth-context"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { getServiceConnections } from "@/lib/billing/plan"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Blocks,
  PanelsTopLeft,
  LogOut,
  Settings,
  PanelLeft,
  Server,
  CreditCard,
  Shield,
  KeyRound,
  MessageSquareText,
  Settings2,
  ChevronsUpDown,
} from "lucide-react"

const SIDEBAR_WIDTH = 256
const COLLAPSED_WIDTH = 64

// コネクション数のモジュールレベルキャッシュ
let cachedConnectionCount: number | null = null

// ナビゲーションアイテム
const navItems = {
  dashboard: { href: "/dashboard", label: "ダッシュボード", icon: PanelsTopLeft },
  mcp: [
    { href: "/mcp-server", label: "MCPサーバー", icon: Server },
    { href: "/services", label: "サービス", icon: Blocks },
    { href: "/tools", label: "ツール", icon: Settings2 },
    { href: "/templates", label: "テンプレート", icon: MessageSquareText },
  ],
  general: [
    { href: "/plans", label: "プラン", icon: CreditCard },
  ],
  admin: [
    { href: "/admin", label: "管理者パネル", icon: Shield },
    { href: "/admin/oauth-apps", label: "OAuth設定", icon: KeyRound },
  ],
}

type NavItem = (typeof navItems.mcp)[0]

interface SidebarProps {
  collapsed?: boolean
  onCollapsedChange?: (collapsed: boolean) => void
  onClose?: () => void
}

export function Sidebar({ collapsed = false, onCollapsedChange, onClose }: SidebarProps) {
  const pathname = usePathname()
  const router = useRouter()
  const { user, isAdmin, signOut } = useAuth()
  const accentPreview = "#d07850"

  const [connectionCount, setConnectionCount] = useState(cachedConnectionCount ?? 0)

  // モバイル: ページ遷移時にサイドバーを閉じる（初回マウントは除外）
  const prevPathnameRef = useRef(pathname)
  useEffect(() => {
    if (pathname !== prevPathnameRef.current) {
      prevPathnameRef.current = pathname
      if (onClose) {
        onClose()
      }
    }
  }, [pathname, onClose])

  // 接続済みサービス数を取得
  useEffect(() => {
    async function fetchConnectionCount() {
      try {
        const connections = await getServiceConnections()
        cachedConnectionCount = connections.length
        setConnectionCount(connections.length)
      } catch (error) {
        console.error("Failed to fetch connection count:", error)
      }
    }
    if (user) {
      fetchConnectionCount()
    }
  }, [user?.id])

  const handleSignOut = async () => {
    await signOut()
    router.push("/login")
  }

  const toggleSidebar = () => {
    if (onCollapsedChange) {
      onCollapsedChange(!collapsed)
    } else if (onClose) {
      onClose()
    }
  }

  // 折りたたみ時の内幅 = COLLAPSED_WIDTH - px-3*2 = 64 - 24 = 40px
  // アイコン領域を40px固定にすることで、折りたたみ時に中央配置され、展開時もアイコン位置が変わらない
  const ICON_AREA_WIDTH = COLLAPSED_WIDTH - 24 // px-3 (12px) * 2 = 24px

  const renderNavItem = (item: NavItem) => {
    const isActive = pathname === item.href || (item.href !== "/admin" && pathname.startsWith(item.href + "/"))
    const link = (
      <Link
        key={item.href}
        href={item.href}
        onClick={(e) => e.stopPropagation()}
        className={cn(
          "flex items-center gap-2 h-9 rounded-md text-sm transition-colors",
          isActive
            ? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
            : "text-sidebar-foreground/80 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground",
        )}
      >
        <div className="flex items-center justify-center shrink-0" style={{ width: ICON_AREA_WIDTH }}>
          <item.icon className="h-[18px] w-[18px]" />
        </div>
        <span className={cn(
          "transition-opacity duration-200 whitespace-nowrap",
          collapsed ? "opacity-0 w-0 overflow-hidden" : "opacity-100"
        )}>{item.label}</span>
      </Link>
    )

    if (!collapsed) return <div key={item.href}>{link}</div>

    return (
      <Tooltip key={item.href}>
        <TooltipTrigger asChild>{link}</TooltipTrigger>
        <TooltipContent side="right">{item.label}</TooltipContent>
      </Tooltip>
    )
  }

  const sidebarWidth = collapsed ? COLLAPSED_WIDTH : SIDEBAR_WIDTH

  return (
    <TooltipProvider>
    <aside
      className="relative flex flex-col h-full bg-sidebar border-r border-sidebar-border overflow-hidden transition-all duration-300"
      style={{ width: sidebarWidth }}
    >
      {/* Header with Logo and Toggle */}
      <div className="flex items-center h-14 px-3">
        <div className="flex items-center justify-center shrink-0" style={{ width: ICON_AREA_WIDTH }}>
          <button
            className="h-7 w-7 flex items-center justify-center rounded-md text-muted-foreground hover:text-foreground hover:bg-sidebar-accent/50 transition-colors"
            onClick={toggleSidebar}
          >
            <PanelLeft className="h-4 w-4" />
          </button>
        </div>
        <span className={cn(
          "font-semibold text-foreground text-sm whitespace-nowrap transition-opacity duration-200 ml-2",
          collapsed ? "opacity-0 w-0 overflow-hidden ml-0" : "opacity-100"
        )}>MCPist</span>
      </div>

      {/* Connection Count */}
      <div
        className={cn("mb-2 mx-3 flex items-center", collapsed && "cursor-pointer")}
        onClick={collapsed ? toggleSidebar : undefined}
      >
        <div className="flex items-center gap-2">
          <div className="flex items-center justify-center shrink-0" style={{ width: ICON_AREA_WIDTH }}>
            <div
              className="flex items-center justify-center w-7 h-7 rounded-full text-xs font-semibold"
              style={{
                color: accentPreview,
                backgroundColor: `${accentPreview}15`,
              }}
            >
              {connectionCount}
            </div>
          </div>
          <span className={cn(
            "text-sm text-muted-foreground transition-opacity duration-200 whitespace-nowrap",
            collapsed ? "opacity-0 w-0 overflow-hidden" : "opacity-100"
          )}>コネクション</span>
        </div>
      </div>

      {/* Navigation */}
      <nav
        className={cn("flex-1 py-1 px-3 overflow-hidden", collapsed && "cursor-pointer")}
        onClick={collapsed ? toggleSidebar : undefined}
      >
        <div className="space-y-0.5">
          {renderNavItem(navItems.dashboard)}
          {navItems.mcp.map(renderNavItem)}
          {navItems.general.map(renderNavItem)}
          {isAdmin && navItems.admin.map(renderNavItem)}
        </div>
      </nav>

      {/* User Profile - 固定高さで開閉時のズレを防止 */}
      <div className="h-14 px-3 border-t border-sidebar-border flex items-center">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button className={cn(
              "flex items-center h-10 rounded-md hover:bg-sidebar-accent/50 transition-colors text-left",
              collapsed ? "" : "w-full gap-2"
            )}>
              <div className="flex items-center justify-center shrink-0" style={{ width: ICON_AREA_WIDTH }}>
                <Avatar className="h-8 w-8 shrink-0">
                  {user?.avatar && <AvatarImage src={user.avatar} />}
                  <AvatarFallback className="bg-primary text-primary-foreground text-xs">
                    {user?.name?.slice(0, 2) || "U"}
                  </AvatarFallback>
                </Avatar>
              </div>
              <div className={cn(
                "flex-1 min-w-0 transition-opacity duration-200",
                collapsed ? "opacity-0 w-0 overflow-hidden" : "opacity-100"
              )}>
                <div className="text-sm font-medium text-sidebar-foreground truncate">{user?.name}</div>
              </div>
              <ChevronsUpDown className={cn(
                "h-4 w-4 text-muted-foreground shrink-0 transition-opacity duration-200",
                collapsed ? "opacity-0 w-0 overflow-hidden" : "opacity-100"
              )} />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent side="top" align={collapsed ? "center" : "start"} className="w-56">
            {/* ユーザー情報 */}
            <div className="px-2 py-1.5">
              <p className="text-sm font-medium truncate">{user?.name}</p>
              {user?.email && (
                <p className="text-xs text-muted-foreground truncate">{user.email}</p>
              )}
            </div>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <Link href="/settings">
                <Settings className="mr-2 h-4 w-4" />
                設定
              </Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleSignOut}>
              <LogOut className="mr-2 h-4 w-4" />
              ログアウト
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </aside>
    </TooltipProvider>
  )
}
