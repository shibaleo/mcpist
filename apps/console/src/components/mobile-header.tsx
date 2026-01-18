"use client"

import { Button } from "@/components/ui/button"
import { Sheet, SheetContent, SheetTrigger, SheetTitle } from "@/components/ui/sheet"
import * as VisuallyHidden from "@radix-ui/react-visually-hidden"
import { Menu, Zap } from "lucide-react"
import { Sidebar } from "./sidebar"
import { useState } from "react"

export function MobileHeader() {
  const [open, setOpen] = useState(false)

  return (
    <header className="flex items-center h-14 px-4 border-b border-border bg-background md:hidden">
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetTrigger asChild>
          <Button variant="ghost" size="icon" className="mr-3">
            <Menu className="h-5 w-5" />
            <span className="sr-only">メニューを開く</span>
          </Button>
        </SheetTrigger>
        <SheetContent side="left" className="p-0 w-64" hideCloseButton>
          <VisuallyHidden.Root>
            <SheetTitle>ナビゲーションメニュー</SheetTitle>
          </VisuallyHidden.Root>
          <Sidebar onClose={() => setOpen(false)} />
        </SheetContent>
      </Sheet>
      <div className="flex items-center gap-2">
        <div className="w-7 h-7 rounded-lg bg-primary flex items-center justify-center">
          <Zap className="h-4 w-4 text-primary-foreground" />
        </div>
        <span className="font-semibold">MCPist</span>
      </div>
    </header>
  )
}
