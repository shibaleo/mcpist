"use client"

import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import { useState, useCallback, useEffect, useRef } from "react"
import { cn } from "@/lib/utils"
import { useAuth } from "@/lib/auth-context"
import { Button } from "@/components/ui/button"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
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
  ChevronLeft,
  Link2,
  Server,
  Settings2,
  CreditCard,
  Store,
  Zap,
  Shield,
  HelpCircle,
} from "lucide-react"

const MIN_WIDTH = 150
const MAX_WIDTH = 400
const DEFAULT_WIDTH = 256
const COLLAPSED_WIDTH = 64

// ナビゲーションアイテム
const navItems = {
  dashboard: { href: "/dashboard", label: "ダッシュボード", icon: LayoutDashboard },
  mcp: [
    { href: "/my/mcp-connection", label: "サーバー接続", icon: Server },
    { href: "/my/connections", label: "サービス連携", icon: Link2 },
    { href: "/my/preferences", label: "ツール設定", icon: Settings2 },
  ],
  general: [
    { href: "/marketplace", label: "マーケットプレイス", icon: Store },
    { href: "/billing", label: "請求", icon: CreditCard },
    { href: "/settings", label: "設定", icon: Settings },
  ],
  admin: [
    { href: "/admin", label: "管理者パネル", icon: Shield },
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

  const [width, setWidth] = useState(DEFAULT_WIDTH)
  const [isResizing, setIsResizing] = useState(false)
  const sidebarRef = useRef<HTMLElement>(null)

  const handleSignOut = async () => {
    await signOut()
    router.push("/login")
  }

  const startResizing = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    setIsResizing(true)
  }, [])

  const stopResizing = useCallback(() => {
    setIsResizing(false)
  }, [])

  const resize = useCallback((e: MouseEvent) => {
    if (isResizing && sidebarRef.current) {
      const newWidth = e.clientX - sidebarRef.current.getBoundingClientRect().left
      if (newWidth >= MIN_WIDTH && newWidth <= MAX_WIDTH) {
        setWidth(newWidth)
      }
    }
  }, [isResizing])

  useEffect(() => {
    if (isResizing) {
      window.addEventListener("mousemove", resize)
      window.addEventListener("mouseup", stopResizing)
    }
    return () => {
      window.removeEventListener("mousemove", resize)
      window.removeEventListener("mouseup", stopResizing)
    }
  }, [isResizing, resize, stopResizing])

  const toggleSidebar = useCallback(() => {
    if (onCollapsedChange) {
      onCollapsedChange(!collapsed)
    } else if (onClose) {
      onClose()
    }
  }, [collapsed, onCollapsedChange, onClose])

  const renderNavItem = (item: NavItem) => {
    const isActive = pathname === item.href || pathname.startsWith(item.href + "/")
    return (
      <Link
        key={item.href}
        href={item.href}
        onClick={(e) => e.stopPropagation()}
      >
        <Button
          variant={isActive ? "secondary" : "ghost"}
          className={cn(
            "h-10 w-full justify-start gap-3 px-2",
            isActive && "bg-sidebar-accent text-sidebar-accent-foreground",
          )}
        >
          <item.icon className="h-5 w-5 shrink-0" />
          <span className={cn(
            "transition-opacity duration-300 whitespace-nowrap",
            collapsed ? "opacity-0" : "opacity-100"
          )}>{item.label}</span>
        </Button>
      </Link>
    )
  }

  const sidebarWidth = collapsed ? COLLAPSED_WIDTH : width

  return (
    <aside
      ref={sidebarRef}
      className={cn(
        "relative flex flex-col h-full bg-sidebar border-r border-sidebar-border overflow-hidden transition-all duration-300",
        isResizing && "select-none transition-none"
      )}
      style={{ width: sidebarWidth }}
    >
      {/* Header with Logo and Collapse Button */}
      <div className="flex items-center justify-between h-16 px-4 border-b border-sidebar-border">
        <div
          className="flex items-center gap-3 cursor-pointer"
          onClick={(e) => {
            e.stopPropagation()
            toggleSidebar()
          }}
        >
          <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center shrink-0">
            <Zap className="h-4 w-4 text-primary-foreground" />
          </div>
          <span className={cn(
            "font-semibold text-sidebar-foreground transition-opacity duration-300 whitespace-nowrap",
            collapsed ? "opacity-0" : "opacity-100"
          )}>MCPist</span>
        </div>
        <Button
          variant="ghost"
          size="icon"
          className={cn(
            "h-8 w-8 shrink-0 transition-opacity duration-300",
            collapsed ? "opacity-0 pointer-events-none" : "opacity-100"
          )}
          onClick={() => onCollapsedChange ? onCollapsedChange(true) : onClose?.()}
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
      </div>

      {/* Connected Services Count */}
      <div
        className="h-14 px-4 border-b border-sidebar-border cursor-pointer flex items-center"
        onClick={toggleSidebar}
      >
        <div className="flex items-center gap-3">
          <div
            className="flex items-center justify-center w-8 h-8 rounded-full text-base font-semibold shrink-0"
            style={{
              color: "white",
              backgroundColor: "var(--accent-custom)",
            }}
          >
            3
          </div>
          <span className={cn(
            "text-sm text-muted-foreground transition-opacity duration-300 whitespace-nowrap",
            collapsed ? "opacity-0" : "opacity-100"
          )}>アクティブコネクション</span>
        </div>
      </div>

      {/* Navigation */}
      <nav
        className="flex-1 px-3 py-3 overflow-hidden"
        onClick={toggleSidebar}
      >
        <div className="space-y-1">
          {renderNavItem(navItems.dashboard)}
          {navItems.mcp.map(renderNavItem)}
          {navItems.general.map(renderNavItem)}
          {isAdmin && navItems.admin.map(renderNavItem)}
        </div>
      </nav>

      {/* Help Link */}
      <div className="h-12 px-3 flex items-center">
        <a
          href="https://docs.mcpist.com"
          target="_blank"
          rel="noopener noreferrer"
          onClick={(e) => e.stopPropagation()}
          className="w-full"
        >
          <Button
            variant="ghost"
            className="w-full h-10 justify-start gap-3 px-2"
          >
            <HelpCircle className="h-5 w-5 shrink-0" />
            <span className={cn(
              "transition-opacity duration-300 whitespace-nowrap",
              collapsed ? "opacity-0" : "opacity-100"
            )}>ヘルプ</span>
          </Button>
        </a>
      </div>

      {/* User Profile */}
      <div className="h-14 px-4 border-t border-sidebar-border flex items-center">
        {collapsed ? (
          <Avatar className="h-8 w-8 shrink-0">
            {user?.avatar && <AvatarImage src={user.avatar} />}
            <AvatarFallback className="bg-primary text-primary-foreground text-xs">
              {user?.name?.slice(0, 2) || "U"}
            </AvatarFallback>
          </Avatar>
        ) : (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                className="w-full h-10 justify-start gap-3 px-0"
              >
                <Avatar className="h-8 w-8 shrink-0">
                  {user?.avatar && <AvatarImage src={user.avatar} />}
                  <AvatarFallback className="bg-primary text-primary-foreground text-xs">
                    {user?.name?.slice(0, 2) || "U"}
                  </AvatarFallback>
                </Avatar>
                <span className="text-sm font-medium text-sidebar-foreground truncate">{user?.name}</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent side="right" align="start" className="w-56">
              <DropdownMenuItem className="text-destructive" onClick={handleSignOut}>
                <LogOut className="mr-2 h-4 w-4" />
                ログアウト
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      {/* Resize Handle */}
      {!collapsed && (
        <div
          className="absolute top-0 right-0 w-1.5 h-full cursor-ew-resize bg-sidebar-border hover:bg-primary/30 active:bg-primary/40 transition-colors"
          onMouseDown={startResizing}
        />
      )}
    </aside>
  )
}
