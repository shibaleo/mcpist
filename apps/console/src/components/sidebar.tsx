"use client"

import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import { useState, useEffect } from "react"
import { cn } from "@/lib/utils"
import { useAuth } from "@/lib/auth-context"
import { useAppearance, accentColors } from "@/lib/appearance-context"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { getServiceConnections } from "@/lib/credits"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  LayoutDashboard,
  LogOut,
  Settings,
  PanelLeft,
  Link2,
  Server,
  CreditCard,
  Shield,
  HelpCircle,
  KeyRound,
  MessageSquareText,
  Wrench,
  ChevronsUpDown,
} from "lucide-react"

const SIDEBAR_WIDTH = 256
const COLLAPSED_WIDTH = 64

// コネクション数のモジュールレベルキャッシュ
let cachedConnectionCount: number | null = null

// ナビゲーションアイテム
const navItems = {
  dashboard: { href: "/dashboard", label: "ダッシュボード", icon: LayoutDashboard },
  mcp: [
    { href: "/mcp-server", label: "MCPサーバー", icon: Server },
    { href: "/services", label: "サービス", icon: Link2 },
    { href: "/tools", label: "ツール", icon: Wrench },
    { href: "/templates", label: "テンプレート", icon: MessageSquareText },
  ],
  general: [
    { href: "/credits", label: "クレジット", icon: CreditCard },
    { href: "/settings", label: "設定", icon: Settings },
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
  const { accentColor } = useAppearance()
  const accentPreview = accentColors.find(c => c.id === accentColor)?.preview ?? "#22c55e"

  const [connectionCount, setConnectionCount] = useState(cachedConnectionCount ?? 0)

  // モバイル: ページ遷移時にサイドバーを閉じる（初回マウントは除外）
  const [prevPathname, setPrevPathname] = useState(pathname)
  useEffect(() => {
    if (pathname !== prevPathname) {
      setPrevPathname(pathname)
      if (onClose) {
        onClose()
      }
    }
  }, [pathname, prevPathname, onClose])

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
  }, [user])

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
    const isActive = pathname === item.href || pathname.startsWith(item.href + "/")
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
      className="relative flex flex-col h-full glass-sidebar border-r border-sidebar-border overflow-hidden transition-all duration-300"
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

      {/* Help Link */}
      <div className="py-1 px-3">
        {collapsed ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <a
                href="https://docs.mcpist.com"
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="flex items-center gap-2 h-9 rounded-md text-sm text-sidebar-foreground/80 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground transition-colors"
              >
                <div className="flex items-center justify-center shrink-0" style={{ width: ICON_AREA_WIDTH }}>
                  <HelpCircle className="h-[18px] w-[18px]" />
                </div>
                <span className="opacity-0 w-0 overflow-hidden whitespace-nowrap">ヘルプ</span>
              </a>
            </TooltipTrigger>
            <TooltipContent side="right">ヘルプ</TooltipContent>
          </Tooltip>
        ) : (
          <a
            href="https://docs.mcpist.com"
            target="_blank"
            rel="noopener noreferrer"
            onClick={(e) => e.stopPropagation()}
            className="flex items-center gap-2 h-9 rounded-md text-sm text-sidebar-foreground/80 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground transition-colors"
          >
            <div className="flex items-center justify-center shrink-0" style={{ width: ICON_AREA_WIDTH }}>
              <HelpCircle className="h-[18px] w-[18px]" />
            </div>
            <span className="whitespace-nowrap">ヘルプ</span>
          </a>
        )}
      </div>

      {/* User Profile */}
      <div className="py-3 px-3 border-t border-sidebar-border">
        {collapsed ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="flex items-center">
                <div className="flex items-center justify-center shrink-0" style={{ width: ICON_AREA_WIDTH }}>
                  <Avatar className="h-8 w-8 shrink-0">
                    {user?.avatar && <AvatarImage src={user.avatar} />}
                    <AvatarFallback className="bg-primary text-primary-foreground text-xs">
                      {user?.name?.slice(0, 2) || "U"}
                    </AvatarFallback>
                  </Avatar>
                </div>
              </div>
            </TooltipTrigger>
            <TooltipContent side="right">{user?.name}</TooltipContent>
          </Tooltip>
        ) : (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button className="w-full flex items-center gap-2 py-1.5 rounded-md hover:bg-sidebar-accent/50 transition-colors text-left">
                <div className="flex items-center justify-center shrink-0" style={{ width: ICON_AREA_WIDTH }}>
                  <Avatar className="h-8 w-8 shrink-0">
                    {user?.avatar && <AvatarImage src={user.avatar} />}
                    <AvatarFallback className="bg-primary text-primary-foreground text-xs">
                      {user?.name?.slice(0, 2) || "U"}
                    </AvatarFallback>
                  </Avatar>
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-sidebar-foreground truncate">{user?.name}</div>
                </div>
                <ChevronsUpDown className="h-4 w-4 text-muted-foreground shrink-0" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent side="right" align="end" className="w-56">
              <DropdownMenuItem className="text-destructive" onClick={handleSignOut}>
                <LogOut className="mr-2 h-4 w-4" />
                ログアウト
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </aside>
    </TooltipProvider>
  )
}
